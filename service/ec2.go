package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"

	awsclient "outpost-cli/service/aws"
	localDb "outpost-cli/service/localDb"
)

// LocalUserID is the fixed user id for local-mode CLI usage.
const LocalUserID = localDb.LocalUserID

type Instance struct {
	ID               string
	Name             string
	Status           string
	InstanceType     string
	IPAddress        string
	PrivateIPAddress string
	Region           string
	Provider         string
}

// SSHHost returns the public IP if set, otherwise the private IP.
func (i *Instance) SSHHost() (string, error) {
	host := i.IPAddress
	if host == "" {
		host = i.PrivateIPAddress
	}
	if host == "" {
		return "", fmt.Errorf("box has no IP address (is it running?)")
	}
	return host, nil
}

// if user does not have it then we will see no rows
func requireOwnedInstance(db *localDb.DB, awsID, userID string) (*localDb.InstanceRecord, error) {
	inst, err := db.GetInstanceByAwsInstanceIDAndUserID(awsID, userID)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("box not found: %s", awsID)
	}
	if err != nil {
		return nil, err
	}
	return inst, nil
}

// ListInstances returns instances for userID, using AWS as the source of truth.
func (r *Runtime) ListInstances(userID string) ([]*Instance, error) {
	db := r.DB()

	records, err := db.ListInstancesByUserID(userID)
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, nil
	}

	cfg, err := awsclient.LoadConfig()
	if err != nil {
		return nil, err
	}
	defaultRegion := cfg.AwsRegion // default region for the user from the config

	byRegion := make(map[string][]localDb.InstanceRecord)
	for _, record := range records {
		region := regionForRecord(record, defaultRegion)
		byRegion[region] = append(byRegion[region], record)
	}

	ctx := r.Context()
	instances := make([]*Instance, 0, len(records))
	var regionErrs []error

	for region, regionRecords := range byRegion {
		awsIDs := make([]string, len(regionRecords))
		for i, record := range regionRecords {
			awsIDs[i] = record.AwsInstanceID
		}

		ec2Client, err := r.EC2ForRegion(region)
		if err != nil {
			regionErrs = append(regionErrs, fmt.Errorf("region %s (%d %s skipped): %w", region, len(regionRecords), boxWord(len(regionRecords)), err))
			continue
		}

		resp, err := ec2Client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
			Filters: []types.Filter{
				{
					Name:   aws.String("instance-id"),
					Values: awsIDs,
				},
			},
		})
		if err != nil {
			regionErrs = append(regionErrs, fmt.Errorf("region %s (%d %s skipped): %w", region, len(regionRecords), boxWord(len(regionRecords)), awsclient.WrapError("describe instances", err)))
			continue
		}

		found := make(map[string]types.Instance)
		for _, reservation := range resp.Reservations {
			for _, inst := range reservation.Instances {
				found[aws.ToString(inst.InstanceId)] = inst
			}
		}

		err = reconcileLocalAgainstRemote(regionRecords,
			func(record localDb.InstanceRecord) string { return record.AwsInstanceID },
			found,
			func(record localDb.InstanceRecord) error {
				if err := db.DeleteInstanceByAwsInstanceID(record.AwsInstanceID); err != nil {
					return err
				}
				_ = DeleteHost(record.Name)
				return nil
			},
			func(record localDb.InstanceRecord, inst types.Instance) error {
				dto := instanceFromAWS(inst)
				dto.Region = region
				dto.Provider = providerForRecord(record, region)
				ip := dto.IPAddress
				if ip == "" {
					ip = dto.PrivateIPAddress
				}

				if record.NeedsAWSSync(dto.Status, ip, dto.InstanceType, dto.Name) {
					if err := db.SyncInstanceFromAWS(
						dto.ID,
						dto.Status,
						ip,
						dto.InstanceType,
						dto.Name,
					); err != nil {
						return err
					}

					if ip != "" {
						if err := syncSSHHostIP(record.Name, ip); err != nil {
							return fmt.Errorf("update SSH config for %q: %w", record.Name, err)
						}
					}
				}
				instances = append(instances, dto)
				return nil
			},
		)
		if err != nil {
			return nil, err
		}
	}

	return instances, errors.Join(regionErrs...)
}

func boxWord(n int) string {
	if n == 1 {
		return "box"
	}
	return "boxes"
}

func regionForRecord(record localDb.InstanceRecord, defaultRegion string) string {
	if region := localDb.StringValue(record.Region); region != "" {
		return region
	}
	return defaultRegion
}

func providerForRecord(record localDb.InstanceRecord, region string) string {
	if provider := localDb.StringValue(record.Provider); provider != "" {
		return provider
	}
	return ProviderForRegion(region)
}

// EC2ForInstance returns an EC2 client scoped to the box's stored AWS region.
func (r *Runtime) EC2ForInstance(awsInstanceID string) (*ec2.Client, error) {
	region, err := r.regionForAwsInstanceID(awsInstanceID) // region for the instance id
	if err != nil {
		return nil, err
	}
	return r.EC2ForRegion(region)
}

func (r *Runtime) regionForAwsInstanceID(awsInstanceID string) (string, error) {
	cfg, err := awsclient.LoadConfig()
	if err != nil {
		return "", err
	}
	record, err := r.DB().GetInstanceByAwsInstanceID(awsInstanceID)
	if err == sql.ErrNoRows {
		return cfg.AwsRegion, nil
	}
	if err != nil {
		return "", err
	}
	return regionForRecord(*record, cfg.AwsRegion), nil
}

// convert to custom Instance struct, originall from aws struct
func instanceFromAWS(inst types.Instance) *Instance {
	name := ""
	for _, tag := range inst.Tags {
		if tag.Key != nil && *tag.Key == "Name" && tag.Value != nil {
			name = *tag.Value
			break
		}
	}

	return &Instance{
		ID:               aws.ToString(inst.InstanceId),
		Name:             name,
		Status:           string(inst.State.Name),
		InstanceType:     string(inst.InstanceType),
		IPAddress:        aws.ToString(inst.PublicIpAddress),
		PrivateIPAddress: aws.ToString(inst.PrivateIpAddress),
	}
}

// GetInstance returns live instance details from AWS for a box owned by userID.
func (r *Runtime) GetInstance(instanceId, userID string) (*Instance, error) {
	db := r.DB()

	if _, err := requireOwnedInstance(db, instanceId, userID); err != nil {
		return nil, err
	}

	return r.getInstanceFromAWS(instanceId)
}

func (r *Runtime) getInstanceFromAWS(instanceId string) (*Instance, error) {
	inst, err := r.describeInstanceFromAWS(instanceId)
	if err != nil {
		return nil, err
	}
	dto := instanceFromAWS(inst)

	cfg, err := awsclient.LoadConfig()
	if err != nil {
		return nil, err
	}
	record, err := r.DB().GetInstanceByAwsInstanceID(instanceId)
	if err == nil {
		dto.Region = regionForRecord(*record, cfg.AwsRegion)
		dto.Provider = providerForRecord(*record, dto.Region)
	} else if err != sql.ErrNoRows {
		return nil, err
	} else {
		dto.Region = cfg.AwsRegion
		dto.Provider = ProviderForRegion(cfg.AwsRegion)
	}
	return dto, nil
}

func (r *Runtime) describeInstanceFromAWS(instanceId string) (types.Instance, error) {
	ec2Client, err := r.EC2ForInstance(instanceId)
	if err != nil {
		return types.Instance{}, err
	}

	ctx := r.Context()
	resp, err := ec2Client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
		InstanceIds: []string{instanceId},
	})
	if err != nil {
		return types.Instance{}, awsclient.WrapError("describe instances", err)
	}

	for _, reservation := range resp.Reservations {
		for _, inst := range reservation.Instances {
			if aws.ToString(inst.InstanceId) == instanceId {
				return inst, nil
			}
		}
	}

	return types.Instance{}, fmt.Errorf("instance not found in AWS: %s", instanceId)
}

// fetches the instance from AWS, updates the local DB, and returns the instance.
func (r *Runtime) syncInstanceFromAWSByID(instanceID string) (*Instance, error) {
	inst, err := r.getInstanceFromAWS(instanceID)
	if err != nil {
		return nil, err
	}

	ip := inst.IPAddress
	if ip == "" {
		ip = inst.PrivateIPAddress
	}

	if err := r.DB().SyncInstanceFromAWS(
		inst.ID,
		inst.Status,
		ip,
		inst.InstanceType,
		inst.Name,
	); err != nil {
		return nil, err
	}
	return inst, nil
}

// RenameInstance updates a local box name. AWS is updated first so the Name tag
// remains the source of truth; DB and SSH config are best-effort follow-ups.
func (r *Runtime) RenameInstance(instanceID, userID, newName string) (*Instance, error) {
	newName = strings.TrimSpace(newName) // trim the name
	db := r.DB()

	record, err := requireOwnedInstance(db, instanceID, userID) // check if the instance is owned by the user
	if err != nil {
		return nil, err
	}
	if err := db.ValidateInstanceNameAvailableForRename(newName, userID, instanceID); err != nil { // check if the name is available
		return nil, err
	}

	ec2Client, err := r.EC2ForInstance(instanceID)
	if err != nil {
		return nil, err
	}

	ctx := r.Context()
	// ec2 update
	if err := updateInstanceNameTag(ctx, ec2Client, instanceID, newName); err != nil {
		return nil, err
	}

	// db update
	if err := db.UpdateInstanceName(instanceID, userID, newName); err != nil {
		return nil, fmt.Errorf("AWS name tag updated but failed to update local database; run outpost ls to resync: %w", err)
	}

	renamed := &Instance{
		ID:     instanceID,
		Name:   newName,
		Status: record.Status,
	}
	if record.InstanceType.Valid {
		renamed.InstanceType = record.InstanceType.String
	}
	if record.IPAddress.Valid {
		renamed.IPAddress = record.IPAddress.String
	}
	return renamed, nil
}

type ec2CreateTagsAPI interface {
	CreateTags(context.Context, *ec2.CreateTagsInput, ...func(*ec2.Options)) (*ec2.CreateTagsOutput, error)
}

func updateInstanceNameTag(ctx context.Context, ec2Client ec2CreateTagsAPI, instanceID, newName string) error {
	_, err := ec2Client.CreateTags(ctx, &ec2.CreateTagsInput{
		Resources: []string{instanceID},
		Tags: []types.Tag{
			{Key: aws.String("Name"), Value: aws.String(newName)},
		},
	})
	if err != nil {
		return awsclient.WrapError("update instance name tag", err)
	}
	return nil
}

// PortForwardResponse mirrors Lighthouse PortForwardResponse for POST /v1/boxes/{id}/ports.
type PortForwardResponse struct {
	Host       string `json:"host"`
	User       string `json:"user"`
	RemotePort string `json:"remotePort"`
}

// ForwardPort returns SSH connection details for port forwarding.
// Mirrors Lighthouse POST /v1/boxes/{id}/ports: BoxesController.forwardPort.
func (r *Runtime) ForwardPort(instanceID, port, userID string) (*PortForwardResponse, error) {
	port = strings.TrimSpace(port)
	if port == "" {
		return nil, fmt.Errorf("missing required field: port")
	}

	box, err := r.GetInstance(instanceID, userID)
	if err != nil {
		return nil, err
	}
	if box.Status != "running" {
		return nil, fmt.Errorf("box is %s, not running", box.Status)
	}

	host, err := box.SSHHost()
	if err != nil {
		return nil, err
	}

	return &PortForwardResponse{
		Host:       host,
		User:       "ec2-user",
		RemotePort: port,
	}, nil
}
