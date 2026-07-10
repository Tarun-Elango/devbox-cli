package service

import (
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"

	awsclient "outpost-cli/service/aws"
)

// ResizeInfo contains the current AWS values needed before prompting.
type ResizeInfo struct {
	Instance     *Instance
	VolumeSizeGB int
}

// GetResizeInfo returns live instance and root volume details for a resize prompt.
func (r *Runtime) GetResizeInfo(instanceID, userID string) (*ResizeInfo, error) {
	if _, err := requireOwnedInstance(r.DB(), instanceID, userID); err != nil {
		return nil, err
	}

	awsInstance, err := r.describeInstanceFromAWS(instanceID)
	if err != nil {
		return nil, err
	}
	volume, err := r.getRootVolumeFromAWS(awsInstance)
	if err != nil {
		return nil, err
	}

	return &ResizeInfo{
		Instance:     instanceFromAWS(awsInstance),
		VolumeSizeGB: int(aws.ToInt32(volume.Size)),
	}, nil
}

// ResizeInstance applies stopped-instance type and root volume changes in AWS.
// The local DB is synced only after AWS accepts the requested changes.
func (r *Runtime) ResizeInstance(instanceID, userID, newInstanceType string, newVolumeSizeGB int) (*Instance, error) {
	if _, err := requireOwnedInstance(r.DB(), instanceID, userID); err != nil {
		return nil, err
	}

	awsInstance, err := r.describeInstanceFromAWS(instanceID) // get the raw instance from AWS
	if err != nil {
		return nil, err
	}
	current := instanceFromAWS(awsInstance) // convert the raw instance to the custom Instance struct
	if current.Status != "stopped" {
		return nil, fmt.Errorf("box is %s, not stopped; stop it before resizing", current.Status)
	}

	ec2Client, err := r.EC2ForInstance(instanceID) // get ec2 client for the instance
	if err != nil {
		return nil, err
	}
	ctx := r.Context()

	// Apply instance type first (reversible while stopped); volume resize last (irreversible).
	if newInstanceType != "" && newInstanceType != current.InstanceType {
		if err := ValidateInstanceType(newInstanceType); err != nil {
			return nil, err
		}
		_, err := ec2Client.ModifyInstanceAttribute(ctx, &ec2.ModifyInstanceAttributeInput{
			InstanceId: aws.String(instanceID),
			InstanceType: &types.AttributeValue{
				Value: aws.String(newInstanceType),
			},
		})
		if err != nil {
			return nil, awsclient.WrapError("modify instance type", err)
		}
	}

	if newVolumeSizeGB > 0 {
		volume, err := r.getRootVolumeFromAWS(awsInstance)
		if err != nil {
			return nil, err
		}
		currentSizeGB := int(aws.ToInt32(volume.Size))
		if newVolumeSizeGB < currentSizeGB {
			return nil, fmt.Errorf("new volume size %d GB is smaller than current size %d GB", newVolumeSizeGB, currentSizeGB)
		}
		if newVolumeSizeGB != currentSizeGB {
			if err := ValidateVolumeSizeGB(newVolumeSizeGB); err != nil {
				return nil, err
			}
			_, err = ec2Client.ModifyVolume(ctx, &ec2.ModifyVolumeInput{
				VolumeId: volume.VolumeId,
				Size:     aws.Int32(int32(newVolumeSizeGB)),
			})
			if err != nil {
				return nil, awsclient.WrapError("modify root volume", err)
			}
		}
	}

	// finally, we will sync the instance from AWS to the local DB
	return r.syncInstanceFromAWSByID(instanceID)
}

func (r *Runtime) getRootVolumeFromAWS(inst types.Instance) (*types.Volume, error) {
	volumeID := rootVolumeID(inst)
	if volumeID == "" {
		return nil, fmt.Errorf("root volume not found for instance %s", aws.ToString(inst.InstanceId))
	}

	ec2Client, err := r.EC2ForInstance(aws.ToString(inst.InstanceId)) // get ec2 client for the instance
	if err != nil {
		return nil, err
	}
	resp, err := ec2Client.DescribeVolumes(r.Context(), &ec2.DescribeVolumesInput{
		VolumeIds: []string{volumeID},
	})
	if err != nil {
		return nil, awsclient.WrapError("describe root volume", err)
	}
	if len(resp.Volumes) == 0 {
		return nil, fmt.Errorf("root volume not found in AWS: %s", volumeID)
	}
	return &resp.Volumes[0], nil
}

func rootVolumeID(inst types.Instance) string {
	rootDeviceName := aws.ToString(inst.RootDeviceName)
	for _, mapping := range inst.BlockDeviceMappings {
		if aws.ToString(mapping.DeviceName) == rootDeviceName && mapping.Ebs != nil {
			return aws.ToString(mapping.Ebs.VolumeId)
		}
	}
	return ""
}
