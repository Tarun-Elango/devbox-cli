package service

import (
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/ec2"

	awsclient "outpost-cli/service/aws"
)

// DeleteInstance terminates a box owned by userID and removes it from the local DB.
// Mirrors Lighthouse DELETE /v1/boxes/{id}: ec2Service.terminateInstance(id, userId).
func (r *Runtime) DeleteInstance(instanceID, userID string) error {
	db := r.DB()

	record, err := requireOwnedInstance(db, instanceID, userID)
	if err != nil {
		return err
	}

	ec2Client, err := r.EC2ForInstance(instanceID)
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

	ec2Client, err := r.EC2ForInstance(instanceID)
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

	ec2Client, err := r.EC2ForInstance(instanceID)
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

// RebootInstance reboots a running box owned by userID.
func (r *Runtime) RebootInstance(instanceID, userID string) error {
	db := r.DB()

	if _, err := requireOwnedInstance(db, instanceID, userID); err != nil {
		return err
	}

	instance, err := r.getInstanceFromAWS(instanceID)
	if err != nil {
		return err
	}
	if err := requireRebootableStatus(instance.Status); err != nil {
		return err
	}

	ec2Client, err := r.EC2ForInstance(instanceID)
	if err != nil {
		return err
	}

	ctx := r.Context()
	_, err = ec2Client.RebootInstances(ctx, &ec2.RebootInstancesInput{
		InstanceIds: []string{instanceID},
	})
	if err != nil {
		return awsclient.WrapError("reboot instance", err)
	}

	if _, err := r.syncInstanceFromAWSByID(instanceID); err != nil {
		return fmt.Errorf("instance rebooted but failed to refresh local state; run outpost ls to resync: %w", err)
	}
	return nil
}

func requireRebootableStatus(status string) error {
	if status != "running" {
		return fmt.Errorf("box is %s, not running", status)
	}
	return nil
}
