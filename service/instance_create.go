package service

import (
	"context"
	"database/sql"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/google/uuid"

	awsclient "devbox-cli/service/aws"
)

// Create defaults — mirrors Lighthouse application.properties.
const (
	defaultAmiID           = "ami-096f34d377a72cea5" // amazon linux 2023 ami
	defaultSecurityGroupID = ""                      // we dont have one, so we will default to creating in the code
	defaultSubnetID        = ""

	isolatedSecurityGroupName = "devbox-isolated"

	readyMessage = "the user data script is completed"
	readyPath    = "/var/lib/devbox/ready"
)

// CreateInstance creates a new box locally.
// Mirrors Lighthouse POST /v2/boxes: launchInstancev2(name, publicKey, snapshotAmiId, userId).
func (r *Runtime) CreateInstance(name, publicKey, snapshotAmiID, userID, instanceType string, volumeSizeGB int) (*Instance, error) {
	return r.createInstanceWithStartupScripts(name, publicKey, snapshotAmiID, userID, instanceType, volumeSizeGB, nil)
}

func (r *Runtime) createInstanceWithStartupScripts(name, publicKey, snapshotAmiID, userID, instanceType string, volumeSizeGB int, startupScripts []string) (*Instance, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("box name is required")
	}

	instanceType = strings.TrimSpace(instanceType)
	if instanceType == "" {
		instanceType = DefaultInstanceType
	}
	if err := ValidateInstanceType(instanceType); err != nil {
		return nil, err
	}

	if volumeSizeGB == 0 {
		volumeSizeGB = DefaultVolumeSizeGB
	}
	if err := ValidateVolumeSizeGB(volumeSizeGB); err != nil {
		return nil, err
	}

	snapshotAmiID = strings.TrimSpace(snapshotAmiID) // snapshotAmiId will only have the id
	fromSnapshot := snapshotAmiID != ""

	ctx := r.Context()
	db := r.DB()
	if err := db.ValidateInstanceNameAvailable(name, userID); err != nil {
		return nil, err
	}

	appCfg, err := awsclient.LoadConfig()
	if err != nil {
		return nil, err
	}

	launchRegion := appCfg.AwsRegion
	effectiveAmiID := defaultAmiID
	var ec2Client *ec2.Client

	if fromSnapshot {
		record, err := db.GetSnapshotByAmiIDAndUserID(snapshotAmiID, userID)
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("snapshot not found: %s", snapshotAmiID)
		}
		if err != nil {
			return nil, err
		}

		launchRegion, err = regionForSnapshotRecord(*record)
		if err != nil {
			return nil, err
		}

		ec2Client, err = r.ec2ForSnapshotRecord(*record)
		if err != nil {
			return nil, err
		}

		resp, err := ec2Client.DescribeImages(ctx, &ec2.DescribeImagesInput{
			Owners:   []string{"self"},
			ImageIds: []string{snapshotAmiID},
			Filters: []types.Filter{
				{Name: aws.String("state"), Values: []string{"available"}},
			},
		})
		if err != nil {
			return nil, awsclient.WrapError("describe images", err)
		}
		if len(resp.Images) == 0 {
			return nil, fmt.Errorf("snapshot AMI not found or not available: %s", snapshotAmiID)
		}

		effectiveAmiID = snapshotAmiID
	} else {
		ec2Client, err = r.EC2ForRegion(launchRegion)
		if err != nil {
			return nil, err
		}
	}

	userData, err := buildUserDataV2(publicKey, startupScripts)
	if err != nil {
		return nil, err
	}

	effectiveSgID := defaultSecurityGroupID
	if effectiveSgID == "" {
		effectiveSgID, err = ensureIsolatedSecurityGroup(ctx, ec2Client)
		if err != nil {
			return nil, err
		}
	}

	input := &ec2.RunInstancesInput{
		ImageId:      aws.String(effectiveAmiID),
		InstanceType: types.InstanceType(instanceType),
		MinCount:     aws.Int32(1),
		MaxCount:     aws.Int32(1),
		TagSpecifications: []types.TagSpecification{
			{
				ResourceType: types.ResourceTypeInstance,
				Tags: []types.Tag{
					{Key: aws.String("Name"), Value: aws.String(name)},
				},
			},
		},
		SecurityGroupIds: []string{effectiveSgID},
		MetadataOptions: &types.InstanceMetadataOptionsRequest{
			HttpTokens:              types.HttpTokensStateRequired,
			HttpPutResponseHopLimit: aws.Int32(1),
		},
		UserData: aws.String(userData),
	}
	// When using the default base AMI apply our standard storage spec.
	// When restoring from a snapshot, honour the AMI's own block device mapping.
	if !fromSnapshot {
		input.BlockDeviceMappings = []types.BlockDeviceMapping{
			{
				DeviceName: aws.String("/dev/xvda"),
				Ebs: &types.EbsBlockDevice{
					VolumeSize:          aws.Int32(int32(volumeSizeGB)),
					VolumeType:          types.VolumeTypeGp3,
					DeleteOnTermination: aws.Bool(true), // make sure to delete the volume when the instance is terminated
				},
			},
		}
	}
	if defaultSubnetID != "" {
		input.SubnetId = aws.String(defaultSubnetID)
	}

	resp, err := ec2Client.RunInstances(ctx, input)
	if err != nil {
		return nil, awsclient.WrapError("run instances", err)
	}
	if len(resp.Instances) == 0 {
		return nil, fmt.Errorf("run instances: no instances returned")
	}

	launched := resp.Instances[0]
	awsInstanceID := aws.ToString(launched.InstanceId)
	provider := ProviderForRegion(launchRegion)

	if err := db.InsertInstance(
		uuid.New().String(),
		awsInstanceID,
		name,
		userID,
		string(launched.State.Name),
		string(launched.InstanceType),
		launchRegion,
		provider,
	); err != nil {
		// if the instance cannot be added to the database, terminate it
		_, termErr := ec2Client.TerminateInstances(ctx, &ec2.TerminateInstancesInput{
			InstanceIds: []string{awsInstanceID},
		})
		if termErr != nil {
			return nil, fmt.Errorf(
				"failed to save box %s to local database and automatic cleanup also failed; instance %s may still be running in AWS and should be terminated manually in the EC2 console: db: %w, terminate: %v",
				name, awsInstanceID, err, termErr,
			)
		}
		return nil, err
	}

	dto := instanceFromAWS(launched)
	dto.Region = launchRegion
	dto.Provider = provider
	return dto, nil
}

// ensureIsolatedSecurityGroup creates or reuses a security group named
// "devbox-isolated" that allows only inbound SSH (TCP 22).
// Mirrors Ec2Service.ensureIsolatedSecurityGroup in Lighthouse.
func ensureIsolatedSecurityGroup(ctx context.Context, ec2Client *ec2.Client) (string, error) {
	var vpcID string
	if defaultSubnetID != "" {
		resp, err := ec2Client.DescribeSubnets(ctx, &ec2.DescribeSubnetsInput{
			SubnetIds: []string{defaultSubnetID},
		})
		if err != nil {
			return "", awsclient.WrapError("describe subnets", err)
		}
		if len(resp.Subnets) == 0 {
			return "", fmt.Errorf("subnet not found: %s", defaultSubnetID)
		}
		vpcID = aws.ToString(resp.Subnets[0].VpcId)
	} else {
		resp, err := ec2Client.DescribeVpcs(ctx, &ec2.DescribeVpcsInput{
			Filters: []types.Filter{
				{Name: aws.String("isDefault"), Values: []string{"true"}},
			},
		})
		if err != nil {
			return "", awsclient.WrapError("describe vpcs", err)
		}
		if len(resp.Vpcs) == 0 {
			return "", fmt.Errorf("no default vpc found")
		}
		vpcID = aws.ToString(resp.Vpcs[0].VpcId)
	}

	// check if security group exists
	existing, err := ec2Client.DescribeSecurityGroups(ctx, &ec2.DescribeSecurityGroupsInput{
		Filters: []types.Filter{
			{Name: aws.String("group-name"), Values: []string{isolatedSecurityGroupName}},
			{Name: aws.String("vpc-id"), Values: []string{vpcID}},
		},
	})
	if err != nil {
		return "", awsclient.WrapError("describe security groups", err)
	}
	var sgID string
	if len(existing.SecurityGroups) > 0 {
		sgID = aws.ToString(existing.SecurityGroups[0].GroupId)
	} else {
		createResp, err := ec2Client.CreateSecurityGroup(ctx, &ec2.CreateSecurityGroupInput{
			GroupName:   aws.String(isolatedSecurityGroupName),
			Description: aws.String("Devbox isolated: SSH inbound only"),
			VpcId:       aws.String(vpcID),
		})
		if err != nil {
			return "", awsclient.WrapError("create security group", err)
		}
		sgID = aws.ToString(createResp.GroupId)
	}

	// Always ensure SSH ingress exists (existing groups may have had the rule removed).
	_, err = ec2Client.AuthorizeSecurityGroupIngress(ctx, &ec2.AuthorizeSecurityGroupIngressInput{
		GroupId: aws.String(sgID),
		IpPermissions: []types.IpPermission{
			{
				IpProtocol: aws.String("tcp"),
				FromPort:   aws.Int32(22),
				ToPort:     aws.Int32(22),
				IpRanges: []types.IpRange{
					{
						CidrIp:      aws.String("0.0.0.0/0"),
						Description: aws.String("SSH access"),
					},
				},
			},
		},
	})
	if err != nil && !awsclient.HasErrorCode(err, "InvalidPermission.Duplicate") {
		return "", awsclient.WrapError("authorize security group ingress", err)
	}

	return sgID, nil
}

// buildUserDataV2 mirrors Ec2Service.buildUserDataV2 in Lighthouse.
func buildUserDataV2(publicKey string, startupScripts []string) (string, error) {
	var sb strings.Builder
	sb.WriteString("#!/bin/bash\n")

	if strings.TrimSpace(publicKey) != "" {
		encodedKey := base64.StdEncoding.EncodeToString([]byte(publicKey))
		sb.WriteString("USER=ec2-user\n")
		sb.WriteString("SSH_DIR=/home/$USER/.ssh\n")
		sb.WriteString("mkdir -p \"$SSH_DIR\"\n")
		fmt.Fprintf(&sb, "echo '%s' | base64 -d >> \"$SSH_DIR/authorized_keys\"\n", encodedKey)
		sb.WriteString("chmod 700 \"$SSH_DIR\"\n")
		sb.WriteString("chmod 600 \"$SSH_DIR/authorized_keys\"\n")
		sb.WriteString("chown -R $USER:$USER \"$SSH_DIR\"\n")
	}

	for _, script := range startupScripts {
		if strings.TrimSpace(script) == "" {
			continue
		}
		sb.WriteString(script)
		if !strings.HasSuffix(script, "\n") {
			sb.WriteString("\n")
		}
	}

	appendReadyMarker(&sb)

	raw := []byte(sb.String())
	if len(raw) > 16*1024 {
		return "", fmt.Errorf("user data exceeds EC2 limit of 16 KiB")
	}
	return base64.StdEncoding.EncodeToString(raw), nil
}

func appendReadyMarker(sb *strings.Builder) {
	sb.WriteString("mkdir -p /var/lib/devbox\n")
	fmt.Fprintf(sb, "echo \"%s\" > %s\n", readyMessage, readyPath)
}
