package service

import (
	"database/sql"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/google/uuid"

	awsclient "devbox-cli/service/aws"
)

// Snapshot mirrors lighthouse SnapshotDto (amiId, name, state, boxAwsId).
type Snapshot struct {
	AmiID    string `json:"amiId"`
	Name     string `json:"name"`
	State    string `json:"state"`
	BoxAwsID string `json:"boxAwsId"`
}

// CreateSnapshot creates an AMI snapshot of the given box.
// Mirrors Lighthouse Ec2Service.createSnapshot.
func (r *Runtime) CreateSnapshot(boxID, name, userID string) (*Snapshot, error) {
	db := r.DB()

	box, err := db.GetInstanceByAwsInstanceIDAndUserID(boxID, userID) // check if box exists
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("box not found: %s", boxID)
	}
	if err != nil {
		return nil, err
	}

	instance, err := r.getInstanceFromAWS(boxID) // get instance from AWS
	if err != nil {
		return nil, err
	}
	if instance.Status == "terminated" || instance.Status == "stopping" {
		return nil, fmt.Errorf("cannot snapshot a %s instance: %s", instance.Status, boxID)
	}

	ec2Client, err := r.EC2()
	if err != nil {
		return nil, err
	}
	ctx := r.Context()
	// create image in AWS
	createResp, err := ec2Client.CreateImage(ctx, &ec2.CreateImageInput{
		InstanceId: aws.String(boxID),
		Name:       aws.String(name),
		NoReboot:   aws.Bool(true),
	})
	if err != nil {
		return nil, awsclient.WrapError("create image", err)
	}

	imageID := aws.ToString(createResp.ImageId)
	// insert snapshot into local db
	if err := db.InsertSnapshot(
		uuid.New().String(),
		imageID,
		name,
		userID,
		box.ID,
		"pending",
	); err != nil {
		return nil, err
	}

	return &Snapshot{ // return snapshot details
		AmiID:    imageID,
		Name:     name,
		State:    "pending",
		BoxAwsID: boxID,
	}, nil
}

// ListSnapshots returns snapshots for userID, syncing state from AWS.
// Mirrors Lighthouse Ec2Service.listSnapshots.
func (r *Runtime) ListSnapshots(userID string) ([]*Snapshot, error) {
	db := r.DB()

	records, err := db.ListSnapshotsByUserID(userID)
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, nil
	}

	amiIDs := make([]string, len(records))
	for i, rec := range records {
		amiIDs[i] = rec.AmiID
	}

	ec2Client, err := r.EC2()
	if err != nil {
		return nil, err
	}

	ctx := r.Context()
	resp, err := ec2Client.DescribeImages(ctx, &ec2.DescribeImagesInput{
		Owners:   []string{"self"},
		ImageIds: amiIDs,
	})
	if err != nil {
		return nil, awsclient.WrapError("describe images", err)
	}

	stateByAmiID := make(map[string]string, len(resp.Images))
	for _, img := range resp.Images {
		stateByAmiID[aws.ToString(img.ImageId)] = string(img.State)
	}

	for _, record := range records {
		awsState, ok := stateByAmiID[record.AmiID]
		if !ok {
			if err := db.DeleteSnapshotByAmiID(record.AmiID); err != nil {
				return nil, err
			}
			continue
		}
		dbState := ""
		if record.State.Valid {
			dbState = record.State.String
		}
		if dbState != awsState {
			if err := db.UpdateSnapshotState(record.AmiID, awsState); err != nil {
				return nil, err
			}
		}
	}

	records, err = db.ListSnapshotsByUserID(userID)
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, nil
	}

	boxAwsIDByBoxID := make(map[string]string)
	for _, rec := range records {
		if !rec.BoxID.Valid || rec.BoxID.String == "" {
			continue
		}
		if _, ok := boxAwsIDByBoxID[rec.BoxID.String]; ok {
			continue
		}
		box, err := db.GetInstanceByID(rec.BoxID.String)
		if err != nil {
			continue
		}
		boxAwsIDByBoxID[rec.BoxID.String] = box.AwsInstanceID
	}

	snapshots := make([]*Snapshot, 0, len(records))
	for _, rec := range records {
		state := "unknown"
		if awsState, ok := stateByAmiID[rec.AmiID]; ok {
			state = awsState
		} else if rec.State.Valid {
			state = rec.State.String
		}
		boxAwsID := ""
		if rec.BoxID.Valid {
			boxAwsID = boxAwsIDByBoxID[rec.BoxID.String]
		}
		snapshots = append(snapshots, &Snapshot{
			AmiID:    rec.AmiID,
			Name:     rec.Name,
			State:    state,
			BoxAwsID: boxAwsID,
		})
	}

	return snapshots, nil
}

// GetSnapshot returns a snapshot by amiID owned by userID, syncing state from AWS.
func (r *Runtime) GetSnapshot(amiID, userID string) (*Snapshot, error) {
	db := r.DB()

	record, err := db.GetSnapshotByAmiIDAndUserID(amiID, userID)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("snapshot not found: %s", amiID)
	}
	if err != nil {
		return nil, err
	}

	ec2Client, err := r.EC2()
	if err != nil {
		return nil, err
	}

	// check snapshot state from AWS
	ctx := r.Context()
	resp, err := ec2Client.DescribeImages(ctx, &ec2.DescribeImagesInput{
		Owners:   []string{"self"},
		ImageIds: []string{amiID}, // detail snapshot by amiID
	})
	if err != nil {
		return nil, awsclient.WrapError("describe images", err)
	}

	if len(resp.Images) == 0 {
		if err := db.DeleteSnapshotByAmiID(amiID); err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("snapshot not found: %s", amiID)
	}

	awsState := string(resp.Images[0].State)
	dbState := ""
	if record.State.Valid {
		dbState = record.State.String
	}
	if dbState != awsState {
		if err := db.UpdateSnapshotState(amiID, awsState); err != nil {
			return nil, err
		}
	}

	boxAwsID := ""
	if record.BoxID.Valid && record.BoxID.String != "" {
		box, err := db.GetInstanceByID(record.BoxID.String)
		if err == nil {
			boxAwsID = box.AwsInstanceID
		}
	}

	return &Snapshot{
		AmiID:    record.AmiID,
		Name:     record.Name,
		State:    awsState,
		BoxAwsID: boxAwsID,
	}, nil
}

// DeleteSnapshot removes a snapshot AMI and its backing EBS snapshots from AWS,
// then deletes the local DB record. Mirrors Lighthouse Ec2Service.deleteSnapshot.
func (r *Runtime) DeleteSnapshot(amiID, userID string) error {
	db := r.DB()

	_, err := db.GetSnapshotByAmiIDAndUserID(amiID, userID) // check if snapshot exists for user
	if err == sql.ErrNoRows {
		return fmt.Errorf("snapshot not found: %s", amiID)
	}
	if err != nil {
		return err
	}

	ec2Client, err := r.EC2()
	if err != nil {
		return err
	}

	ctx := r.Context()
	resp, err := ec2Client.DescribeImages(ctx, &ec2.DescribeImagesInput{
		Owners:   []string{"self"},
		ImageIds: []string{amiID},
	})
	if err != nil {
		return awsclient.WrapError("describe images", err)
	}

	if len(resp.Images) > 0 { // if snapshot exists in AWS
		image := resp.Images[0]
		_, err = ec2Client.DeregisterImage(ctx, &ec2.DeregisterImageInput{ // deregister image in AWS
			ImageId: aws.String(amiID),
		})
		if err != nil {
			return awsclient.WrapError("deregister image", err)
		}

		for _, bdm := range image.BlockDeviceMappings {
			if bdm.Ebs != nil && bdm.Ebs.SnapshotId != nil {
				_, err = ec2Client.DeleteSnapshot(ctx, &ec2.DeleteSnapshotInput{ // delete ebs snapshot in AWS
					SnapshotId: bdm.Ebs.SnapshotId,
				})
				if err != nil {
					return awsclient.WrapError(fmt.Sprintf("delete ebs snapshot %s", aws.ToString(bdm.Ebs.SnapshotId)), err)
				}
			}
		}
	}

	return db.DeleteSnapshotByAmiID(amiID) // delete snapshot from local db
}
