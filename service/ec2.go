package service

import (
	"context"
	"database/sql"
	"encoding/base64"
	"fmt"
	"os"
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
	defaultAmiID           = "ami-096f34d377a72cea5" // amazon linux 2023 ami
	defaultStorageSizeGB   = 16
	defaultSecurityGroupID = "" // we dont have one, so we will default to creating in the code
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

	awsIDs := make([]string, len(records))
	for i, record := range records {
		awsIDs[i] = record.AwsInstanceID
	}

	ec2Client, err := r.EC2()
	if err != nil {
		return nil, err
	}

	ctx := r.Context()
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
		return nil, awsclient.WrapError("describe instances", err)
	}
	//part 1 - delete instances from database that are not in aws
	// aws instance id -> instance object from aws response
	found := make(map[string]types.Instance)
	for _, reservation := range resp.Reservations {
		for _, inst := range reservation.Instances {
			found[aws.ToString(inst.InstanceId)] = inst // map of aws instance id to instance object
		}
	}
	// delete instances from database that are not in aws
	for _, record := range records {
		if _, ok := found[record.AwsInstanceID]; !ok {
			// delete the instance from the database
			if err := db.DeleteInstanceByAwsInstanceID(record.AwsInstanceID); err != nil {
				return nil, err
			}
			// delete the host from the ssh config
			_ = DeleteHost(record.Name)
		}
	}

	// part 2 - compare aws response with local db ( dont care if we look at deleted instances)
	// cause we only consider found/instances in aws response
	// instanceID -> cached DB record from records/localDb from records/localDb
	recordByInstanceID := make(map[string]localDb.InstanceRecord, len(records))
	for _, record := range records {
		recordByInstanceID[record.AwsInstanceID] = record
	}

	instances := make([]*Instance, 0, len(found))

	// loop through the instances in aws response ( only use records for local db values)
	for _, inst := range found {
		dto := instanceFromAWS(inst)
		ip := dto.IPAddress
		if ip == "" {
			ip = dto.PrivateIPAddress
		}
		record := recordByInstanceID[dto.ID] // record has the corresponding record from local db

		// for each instance, if any of the fields are different, sync the instance from aws
		if record.NeedsAWSSync(dto.Status, ip, dto.InstanceType, dto.Name) {
			if err := db.SyncInstanceFromAWS( // input is new values from aws response
				dto.ID,
				dto.Status,
				ip,
				dto.InstanceType,
				dto.Name,
			); err != nil {
				return nil, err
			}

			// if we are syncing the the aws instance to db, then update the ssh config in case the ip address has changed
			if ip != "" {
				if err := syncSSHHostIP(record.Name, ip); err != nil {
					return nil, fmt.Errorf("update SSH config for %q: %w", record.Name, err)
				}
			}
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
func (r *Runtime) CreateInstance(name, publicKey, snapshotAmiID, userID, instanceType string) (*Instance, error) {
	return r.createInstanceWithStartupScripts(name, publicKey, snapshotAmiID, userID, instanceType, nil)
}

func (r *Runtime) createInstanceWithStartupScripts(name, publicKey, snapshotAmiID, userID, instanceType string, startupScripts []string) (*Instance, error) {
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

	snapshotAmiID = strings.TrimSpace(snapshotAmiID) // snapshotAmiId will only have the id
	fromSnapshot := snapshotAmiID != ""
	effectiveAmiID := defaultAmiID

	ctx := r.Context()
	db := r.DB()

	if fromSnapshot {
		_, err := db.GetSnapshotByAmiIDAndUserID(snapshotAmiID, userID)
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("snapshot not found: %s", snapshotAmiID)
		}
		if err != nil {
			return nil, err
		}

		ec2Client, err := r.EC2()
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
	}

	userData, err := buildUserDataV2(publicKey, startupScripts)
	if err != nil {
		return nil, err
	}

	ec2Client, err := r.EC2()
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
					VolumeSize:          aws.Int32(defaultStorageSizeGB),
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

	if err := db.InsertInstance(
		uuid.New().String(),
		awsInstanceID,
		name,
		userID,
		string(launched.State.Name),
		string(launched.InstanceType),
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

	// upon successful creation, add the host to the ssh config
	dto := instanceFromAWS(launched) // convert the aws instance to a custom Instance struct
	if ip, err := dto.SSHHost(); err == nil {
		if err := AddHost(name, ip); err != nil {
			fmt.Fprintf(os.Stderr, "warning: box created but failed to update SSH config: %v\n", err)
		}
	}

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
	if len(existing.SecurityGroups) > 0 { // if security group exists, return it
		return aws.ToString(existing.SecurityGroups[0].GroupId), nil
	}

	// create security group
	createResp, err := ec2Client.CreateSecurityGroup(ctx, &ec2.CreateSecurityGroupInput{
		GroupName:   aws.String(isolatedSecurityGroupName),
		Description: aws.String("Devbox isolated: SSH inbound only"),
		VpcId:       aws.String(vpcID),
	})
	if err != nil {
		return "", awsclient.WrapError("create security group", err)
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

// GetInstance returns live instance details from AWS for a box owned by userID.
// Mirrors Lighthouse GET /v2/boxes/{id}: ec2Service.getInstance(id, userId).
func (r *Runtime) GetInstance(instanceId, userID string) (*Instance, error) {
	db := r.DB()

	if _, err := requireOwnedInstance(db, instanceId, userID); err != nil {
		return nil, err
	}

	return r.getInstanceFromAWS(instanceId)
}

func (r *Runtime) getInstanceFromAWS(instanceId string) (*Instance, error) {
	ec2Client, err := r.EC2()
	if err != nil {
		return nil, err
	}

	ctx := r.Context()
	resp, err := ec2Client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
		InstanceIds: []string{instanceId},
	})
	if err != nil {
		return nil, awsclient.WrapError("describe instances", err)
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

// // used to update an instance, like name, type
// func updateInstance(instanceId string, userId string, name string, status string) (*Instance, error) {
// 	return nil, fmt.Errorf("not implemented")
// }

// DeleteInstance terminates a box owned by userID and removes it from the local DB.
// Mirrors Lighthouse DELETE /v1/boxes/{id}: ec2Service.terminateInstance(id, userId).
func (r *Runtime) DeleteInstance(instanceID, userID string) error {
	db := r.DB()

	record, err := requireOwnedInstance(db, instanceID, userID)
	if err != nil {
		return err
	}

	ec2Client, err := r.EC2()
	if err != nil {
		return err
	}

	// Terminate in AWS first so we never drop the local record while the instance
	// is still running (avoids orphan EC2). Local DB is removed only after terminate succeeds.
	ctx := r.Context()
	_, err = ec2Client.TerminateInstances(ctx, &ec2.TerminateInstancesInput{
		InstanceIds: []string{instanceID},
	})
	if err != nil {
		return awsclient.WrapError("terminate instance", err)
	}

	if err := DeleteHost(record.Name); err != nil {
		return fmt.Errorf("instance terminated but failed to remove SSH config entry: %w", err)
	}

	if err := db.DeleteInstanceByAwsInstanceID(instanceID); err != nil {
		return fmt.Errorf("instance terminated in AWS but failed to remove local record: %w", err)
	}

	return nil
}

// StopInstance stops a running box owned by userID.
// Mirrors Lighthouse POST /v1/boxes/{id}/stop: ec2Service.stopInstance(id, userId).
func (r *Runtime) StopInstance(instanceID, userID string) error {
	db := r.DB()

	if _, err := requireOwnedInstance(db, instanceID, userID); err != nil {
		return err
	}

	ec2Client, err := r.EC2()
	if err != nil {
		return err
	}

	ctx := r.Context()
	_, err = ec2Client.StopInstances(ctx, &ec2.StopInstancesInput{
		InstanceIds: []string{instanceID},
	})
	if err != nil {
		return awsclient.WrapError("stop instance", err)
	}

	if _, err := r.syncInstanceFromAWSByID(instanceID); err != nil {
		return err
	}
	return nil
}

// StartInstance starts a stopped box owned by userID.
// Mirrors Lighthouse POST /v1/boxes/{id}/start: ec2Service.startInstance(id, userId).
func (r *Runtime) StartInstance(instanceID, userID string) error {
	db := r.DB()

	record, err := requireOwnedInstance(db, instanceID, userID)
	if err != nil {
		return err
	}

	instance, err := r.getInstanceFromAWS(instanceID)
	if err != nil {
		return err
	}
	if instance.Status != "stopped" {
		return fmt.Errorf("box is %s, not stopped, or still stopping. ", instance.Status)
	}

	ec2Client, err := r.EC2()
	if err != nil {
		return err
	}

	ctx := r.Context()
	_, err = ec2Client.StartInstances(ctx, &ec2.StartInstancesInput{
		InstanceIds: []string{instanceID},
	})
	if err != nil {
		return awsclient.WrapError("start instance", err)
	}

	inst, err := r.syncInstanceFromAWSByID(instanceID)
	if err != nil {
		return err
	}
	if ip, err := inst.SSHHost(); err == nil {
		if err := syncSSHHostIP(record.Name, ip); err != nil {
			return fmt.Errorf("box started but failed to update SSH config: %w", err)
		}
	}

	return nil
}

// SshStatus mirrors lighthouse SshStatusResponse for local mode.
type SshStatus struct {
	Ready    bool
	Instance *Instance
}

// GetSshStatus returns instance details for SSH. Readiness is determined by the
// user-data probe in cmd, not EC2 instance/system status checks.
// Mirrors Lighthouse GET /v2/boxes/{id}/ssh-status: ec2Service.getSshStatus(id, userId).
func (r *Runtime) GetSshStatus(instanceID, userID string) (*SshStatus, error) {
	db := r.DB()

	if _, err := requireOwnedInstance(db, instanceID, userID); err != nil {
		return nil, err
	}

	return r.checkSshStatusFromAWS(instanceID)
}

func (r *Runtime) checkSshStatusFromAWS(instanceID string) (*SshStatus, error) {
	// notReady := &SshStatus{Ready: false}

	// ec2Client, err := r.EC2() // create the ec2 client from the aws config
	// if err != nil {
	// 	return nil, err
	// }

	// ctx := r.Context()
	// resp, err := ec2Client.DescribeInstanceStatus(ctx, &ec2.DescribeInstanceStatusInput{
	// 	InstanceIds:         []string{instanceID},
	// 	IncludeAllInstances: aws.Bool(true),
	// })
	// if err != nil {
	// 	return nil, awsclient.WrapError("describe instance status", err)
	// }
	// if len(resp.InstanceStatuses) == 0 {
	// 	return notReady, nil
	// }

	// st := resp.InstanceStatuses[0]
	// instanceOk := st.InstanceStatus != nil && st.InstanceStatus.Status == types.SummaryStatusOk
	// systemOk := st.SystemStatus != nil && st.SystemStatus.Status == types.SummaryStatusOk
	// if !instanceOk || !systemOk {
	// 	return notReady, nil
	// }

	inst, err := r.getInstanceFromAWS(instanceID)
	if err != nil {
		return nil, err
	}

	return &SshStatus{Ready: true, Instance: inst}, nil
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

	sshStatus, err := r.GetSshStatus(instanceID, userID)
	if err != nil {
		return nil, err
	}
	if !sshStatus.Ready {
		return nil, fmt.Errorf("box is not ready yet (EC2 status checks still pending)")
	}
	if sshStatus.Instance == nil {
		return nil, fmt.Errorf("box is ready but instance details are unavailable, try again in a few minutes")
	}

	box := sshStatus.Instance
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
