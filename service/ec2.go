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
	localDb "devbox-cli/service/localDb"
)

// EC2 defaults — mirrors Lighthouse application.properties.
const (
	defaultInstanceType    = "t4g.small"
	defaultAmiID           = "ami-03834b8550547b809"
	defaultStorageSizeGB   = 20
	defaultSecurityGroupID = "sg-053a61deae2e90570"
	defaultSubnetID        = ""

	isolatedSecurityGroupName = "devbox-isolated"

	readyMessage = "the user data script is completed"
	readyPath    = "/var/lib/devbox/ready"
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
}

// ListInstances returns instances for userID, using AWS as the source of truth.
func ListInstances(userID string) ([]*Instance, error) {
	db, err := localDb.Open()
	if err != nil {
		return nil, err
	}
	defer db.Close() // close in case of error or at the end of the function

	records, err := db.ListInstancesByUserID(userID)
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, nil
	}

	awsIDs := make([]string, len(records))
	for i, r := range records {
		awsIDs[i] = r.AwsInstanceID
	}

	ctx := context.Background()
	client, err := awsclient.NewClient(ctx)
	if err != nil {
		return nil, err
	}

	ec2Client := ec2.NewFromConfig(client.Config())
	resp, err := ec2Client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
		Filters: []types.Filter{
			{
				Name:   aws.String("instance-id"),
				Values: awsIDs,
			},
		},
	})
	//fmt.Println("Describe instances response:", resp)
	if err != nil {
		return nil, fmt.Errorf("describe instances: %w", err)
	}

	found := make(map[string]types.Instance)
	// map of aws instance id to instance details
	for _, reservation := range resp.Reservations {
		for _, inst := range reservation.Instances {
			found[aws.ToString(inst.InstanceId)] = inst
		}
	}
	// delete instances from database that are not in aws
	for _, record := range records {
		if _, ok := found[record.AwsInstanceID]; !ok {
			if err := db.DeleteInstanceByAwsInstanceID(record.AwsInstanceID); err != nil {
				return nil, err
			}
		}
	}

	// sync instances from aws to database
	// TODO: only sync when there is difference between local and aws
	instances := make([]*Instance, 0, len(found))
	for _, inst := range found {
		dto := instanceFromAWS(inst)
		ip := dto.IPAddress
		if ip == "" {
			ip = dto.PrivateIPAddress
		}
		if err := db.SyncInstanceFromAWS(
			dto.ID,
			dto.Status,
			ip,
			dto.InstanceType,
			dto.Name,
		); err != nil {
			return nil, err
		}
		instances = append(instances, dto)
	}

	return instances, nil
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

// CreateInstance creates a new box locally.
// Mirrors Lighthouse POST /v2/boxes: launchInstancev2(name, publicKey, snapshotAmiId, userId).
func CreateInstance(name, publicKey, fromSnapshot, userID string) (*Instance, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("box name is required")
	}
	if fromSnapshot != "" {
		return nil, fmt.Errorf("creating from snapshot is not supported in local mode yet")
	}

	userData, err := buildUserDataV2(publicKey, nil)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	client, err := awsclient.NewClient(ctx)
	if err != nil {
		return nil, err
	}
	ec2Client := ec2.NewFromConfig(client.Config())

	effectiveSgID := defaultSecurityGroupID
	if effectiveSgID == "" {
		effectiveSgID, err = ensureIsolatedSecurityGroup(ctx, ec2Client)
		if err != nil {
			return nil, err
		}
	}

	input := &ec2.RunInstancesInput{
		ImageId:      aws.String(defaultAmiID),
		InstanceType: types.InstanceType(defaultInstanceType),
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
		BlockDeviceMappings: []types.BlockDeviceMapping{
			{
				DeviceName: aws.String("/dev/xvda"),
				Ebs: &types.EbsBlockDevice{
					VolumeSize:          aws.Int32(defaultStorageSizeGB),
					VolumeType:          types.VolumeTypeGp3,
					DeleteOnTermination: aws.Bool(true),
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
	if defaultSubnetID != "" {
		input.SubnetId = aws.String(defaultSubnetID)
	}

	resp, err := ec2Client.RunInstances(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("run instances: %w", err)
	}
	if len(resp.Instances) == 0 {
		return nil, fmt.Errorf("run instances: no instances returned")
	}

	launched := resp.Instances[0]
	awsInstanceID := aws.ToString(launched.InstanceId)

	db, err := localDb.Open()
	if err != nil {
		return nil, err
	}
	defer db.Close()

	if err := db.InsertInstance(
		uuid.New().String(),
		awsInstanceID,
		name,
		userID,
		string(launched.State.Name),
		string(launched.InstanceType),
	); err != nil {
		return nil, err
	}

	return instanceFromAWS(launched), nil
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
			return "", fmt.Errorf("describe subnets: %w", err)
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
			return "", fmt.Errorf("describe vpcs: %w", err)
		}
		if len(resp.Vpcs) == 0 {
			return "", fmt.Errorf("no default vpc found")
		}
		vpcID = aws.ToString(resp.Vpcs[0].VpcId)
	}

	existing, err := ec2Client.DescribeSecurityGroups(ctx, &ec2.DescribeSecurityGroupsInput{
		Filters: []types.Filter{
			{Name: aws.String("group-name"), Values: []string{isolatedSecurityGroupName}},
			{Name: aws.String("vpc-id"), Values: []string{vpcID}},
		},
	})
	if err != nil {
		return "", fmt.Errorf("describe security groups: %w", err)
	}
	if len(existing.SecurityGroups) > 0 {
		return aws.ToString(existing.SecurityGroups[0].GroupId), nil
	}

	createResp, err := ec2Client.CreateSecurityGroup(ctx, &ec2.CreateSecurityGroupInput{
		GroupName:   aws.String(isolatedSecurityGroupName),
		Description: aws.String("Devbox isolated: SSH inbound only"),
		VpcId:       aws.String(vpcID),
	})
	if err != nil {
		return "", fmt.Errorf("create security group: %w", err)
	}
	sgID := aws.ToString(createResp.GroupId)

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
	if err != nil {
		return "", fmt.Errorf("authorize security group ingress: %w", err)
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

// GetInstance returns live instance details from AWS for a box owned by userID.
// Mirrors Lighthouse GET /v2/boxes/{id}: ec2Service.getInstance(id, userId).
func GetInstance(instanceId, userID string) (*Instance, error) {
	db, err := localDb.Open()
	if err != nil {
		return nil, err
	}
	defer db.Close()

	_, err = db.GetInstanceByAwsInstanceIDAndUserID(instanceId, userID) // get instance from local db
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("box not found: %s", instanceId)
	}
	if err != nil {
		return nil, err
	}

	return getInstanceFromAWS(instanceId)
}

func getInstanceFromAWS(instanceId string) (*Instance, error) {
	ctx := context.Background()
	client, err := awsclient.NewClient(ctx)
	if err != nil {
		return nil, err
	}

	ec2Client := ec2.NewFromConfig(client.Config())
	resp, err := ec2Client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
		InstanceIds: []string{instanceId},
	})
	if err != nil {
		return nil, fmt.Errorf("describe instances: %w", err)
	}

	for _, reservation := range resp.Reservations {
		for _, inst := range reservation.Instances {
			if aws.ToString(inst.InstanceId) == instanceId {
				return instanceFromAWS(inst), nil
			}
		}
	}

	return nil, fmt.Errorf("instance not found in AWS: %s", instanceId)
}

// // used to update an instance, like name, type
// func updateInstance(instanceId string, userId string, name string, status string) (*Instance, error) {
// 	return nil, fmt.Errorf("not implemented")
// }

// DeleteInstance terminates a box owned by userID and removes it from the local DB.
// Mirrors Lighthouse DELETE /v1/boxes/{id}: ec2Service.terminateInstance(id, userId).
func DeleteInstance(instanceID, userID string) (*ActionResult, error) {
	db, err := localDb.Open()
	if err != nil {
		return nil, err
	}
	defer db.Close()

	_, err = db.GetInstanceByAwsInstanceIDAndUserID(instanceID, userID)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("box not found: %s", instanceID)
	}
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	client, err := awsclient.NewClient(ctx)
	if err != nil {
		return nil, err
	}

	ec2Client := ec2.NewFromConfig(client.Config())
	_, err = ec2Client.TerminateInstances(ctx, &ec2.TerminateInstancesInput{
		InstanceIds: []string{instanceID},
	})
	if err != nil {
		return &ActionResult{
			Success: false,
			Message: fmt.Sprintf("Failed to terminate instance %s: %v", instanceID, err),
		}, nil
	}

	if err := db.DeleteInstanceByAwsInstanceID(instanceID); err != nil {
		return nil, err
	}

	return &ActionResult{
		Success: true,
		Message: fmt.Sprintf("Instance %s terminated successfully.", instanceID),
	}, nil
}

// ActionResult mirrors lighthouse ActionResult for stop/start/delete responses.
type ActionResult struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// StopInstance stops a running box owned by userID.
// Mirrors Lighthouse POST /v1/boxes/{id}/stop: ec2Service.stopInstance(id, userId).
func StopInstance(instanceID, userID string) (*ActionResult, error) {
	db, err := localDb.Open()
	if err != nil {
		return nil, err
	}
	defer db.Close()

	_, err = db.GetInstanceByAwsInstanceIDAndUserID(instanceID, userID)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("box not found: %s", instanceID)
	}
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	client, err := awsclient.NewClient(ctx)
	if err != nil {
		return nil, err
	}

	ec2Client := ec2.NewFromConfig(client.Config())
	_, err = ec2Client.StopInstances(ctx, &ec2.StopInstancesInput{
		InstanceIds: []string{instanceID},
	})
	if err != nil {
		return &ActionResult{
			Success: false,
			Message: fmt.Sprintf("Failed to stop instance %s: %v", instanceID, err),
		}, nil
	}

	return &ActionResult{
		Success: true,
		Message: fmt.Sprintf("Instance %s stopped successfully.", instanceID),
	}, nil
}

// StartInstance starts a stopped box owned by userID.
// Mirrors Lighthouse POST /v1/boxes/{id}/start: ec2Service.startInstance(id, userId).
func StartInstance(instanceID, userID string) (*ActionResult, error) {
	db, err := localDb.Open()
	if err != nil {
		return nil, err
	}
	defer db.Close()

	_, err = db.GetInstanceByAwsInstanceIDAndUserID(instanceID, userID)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("box not found: %s", instanceID)
	}
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	client, err := awsclient.NewClient(ctx)
	if err != nil {
		return nil, err
	}

	ec2Client := ec2.NewFromConfig(client.Config())
	_, err = ec2Client.StartInstances(ctx, &ec2.StartInstancesInput{
		InstanceIds: []string{instanceID},
	})
	if err != nil {
		return &ActionResult{
			Success: false,
			Message: fmt.Sprintf("Failed to start instance %s: %v", instanceID, err),
		}, nil
	}

	return &ActionResult{
		Success: true,
		Message: fmt.Sprintf("Instance %s started successfully.", instanceID),
	}, nil
}

// SshStatus mirrors lighthouse SshStatusResponse for local mode.
type SshStatus struct {
	Ready    bool
	Instance *Instance
}

// GetSshStatus checks EC2 instance/system status before SSH.
// Mirrors Lighthouse GET /v2/boxes/{id}/ssh-status: ec2Service.getSshStatus(id, userId).
func GetSshStatus(instanceID, userID string) (*SshStatus, error) {
	db, err := localDb.Open()
	if err != nil {
		return nil, err
	}
	defer db.Close()

	_, err = db.GetInstanceByAwsInstanceIDAndUserID(instanceID, userID)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("box not found: %s", instanceID)
	}
	if err != nil {
		return nil, err
	}

	return checkSshStatusFromAWS(instanceID), nil
}

func checkSshStatusFromAWS(instanceID string) *SshStatus {
	notReady := &SshStatus{Ready: false}

	ctx := context.Background()
	client, err := awsclient.NewClient(ctx)
	if err != nil {
		return notReady
	}

	ec2Client := ec2.NewFromConfig(client.Config())
	resp, err := ec2Client.DescribeInstanceStatus(ctx, &ec2.DescribeInstanceStatusInput{
		InstanceIds:         []string{instanceID},
		IncludeAllInstances: aws.Bool(true),
	})
	if err != nil {
		return notReady
	}
	if len(resp.InstanceStatuses) == 0 {
		return notReady
	}

	st := resp.InstanceStatuses[0]
	instanceOk := st.InstanceStatus != nil && st.InstanceStatus.Status == types.SummaryStatusOk
	systemOk := st.SystemStatus != nil && st.SystemStatus.Status == types.SummaryStatusOk
	if !instanceOk || !systemOk {
		return notReady
	}

	inst, err := getInstanceFromAWS(instanceID)
	if err != nil {
		return notReady
	}

	return &SshStatus{Ready: true, Instance: inst}
}

// PortForwardResponse mirrors Lighthouse PortForwardResponse for POST /v1/boxes/{id}/ports.
type PortForwardResponse struct {
	Host       string `json:"host"`
	User       string `json:"user"`
	RemotePort string `json:"remotePort"`
}

// ForwardPort returns SSH connection details for port forwarding.
// Mirrors Lighthouse POST /v1/boxes/{id}/ports: BoxesController.forwardPort.
func ForwardPort(instanceID, port, userID string) (*PortForwardResponse, error) {
	port = strings.TrimSpace(port)
	if port == "" {
		return nil, fmt.Errorf("missing required field: port")
	}

	box, err := GetInstance(instanceID, userID)
	if err != nil {
		return nil, err
	}
	if box.Status != "running" {
		return nil, fmt.Errorf("box is %s, not running", box.Status)
	}

	host := box.IPAddress
	if host == "" {
		host = box.PrivateIPAddress
	}
	if host == "" {
		return nil, fmt.Errorf("no IP address available for instance: %s", instanceID)
	}

	return &PortForwardResponse{
		Host:       host,
		User:       "ec2-user",
		RemotePort: port,
	}, nil
}
