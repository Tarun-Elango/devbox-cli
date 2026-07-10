package service

import (
	"database/sql"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/google/uuid"

	awsclient "outpost-cli/service/aws"
	localDb "outpost-cli/service/localDb"
)

// Snapshot mirrors lighthouse SnapshotDto (amiId, name, state, boxAwsId), plus
// region/provider captured at creation time so the snapshot remains addressable
// even after its source box is deleted.
type Snapshot struct {
	AmiID    string `json:"amiId"`
	Name     string `json:"name"`
	State    string `json:"state"`
	BoxAwsID string `json:"boxAwsId"`
	Region   string `json:"region"`
	Provider string `json:"provider"`
}

// CreateSnapshot creates an AMI snapshot of the given box.
// Mirrors Lighthouse Ec2Service.createSnapshot.
func (r *Runtime) CreateSnapshot(boxID, name, userID string) (*Snapshot, error) {
	db := r.DB()

	box, err := requireOwnedInstance(db, boxID, userID)
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

	// Capture the box's region/provider on the snapshot itself so it stays
	// addressable even if the source box is later deleted.
	region := instance.Region
	provider := instance.Provider
	if provider == "" {
		provider = ProviderForRegion(region)
	}

	ec2Client, err := r.EC2ForRegion(region) // get ec2 client scoped to the box's region
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
		region,
		provider,
	); err != nil {
		return nil, err
	}

	return &Snapshot{ // return snapshot details
		AmiID:    imageID,
		Name:     name,
		State:    "pending",
		BoxAwsID: boxID,
		Region:   region,
		Provider: provider,
	}, nil
}

// ListSnapshots returns snapshots for userID, syncing state from AWS.
// Mirrors Lighthouse Ec2Service.listSnapshots.
func (r *Runtime) ListSnapshots(userID string) ([]*Snapshot, error) {
	db := r.DB()

	records, err := db.ListSnapshotsByUserIDWithBoxAwsID(userID)
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, nil
	}

	amiIDsByRegion := make(map[string][]string)
	for _, rec := range records {
		region, err := regionForSnapshotRecord(rec.SnapshotRecord)
		if err != nil {
			return nil, err
		}
		amiIDsByRegion[region] = append(amiIDsByRegion[region], rec.AmiID)
	}

	ctx := r.Context()
	stateByAmiID := make(map[string]string, len(records))
	for region, amiIDs := range amiIDsByRegion {
		ec2Client, err := r.EC2ForRegion(region)
		if err != nil {
			return nil, err
		}
		resp, err := ec2Client.DescribeImages(ctx, &ec2.DescribeImagesInput{
			Owners:   []string{"self"},
			ImageIds: amiIDs,
		})
		if err != nil {
			return nil, awsclient.WrapError("describe images", err)
		}
		for _, img := range resp.Images {
			stateByAmiID[aws.ToString(img.ImageId)] = string(img.State)
		}
	}

	snapshots := make([]*Snapshot, 0, len(records))
	err = reconcileLocalAgainstRemote(records,
		func(rec localDb.SnapshotWithBoxAwsID) string { return rec.AmiID },
		stateByAmiID,
		func(rec localDb.SnapshotWithBoxAwsID) error {
			return db.DeleteSnapshotByAmiID(rec.AmiID)
		},
		func(rec localDb.SnapshotWithBoxAwsID, awsState string) error {
			if localDb.StringValue(rec.State) != awsState {
				if err := db.UpdateSnapshotState(rec.AmiID, awsState); err != nil {
					return err
				}
			}
			region, err := regionForSnapshotRecord(rec.SnapshotRecord)
			if err != nil {
				return err
			}
			snapshots = append(snapshots, &Snapshot{
				AmiID:    rec.AmiID,
				Name:     rec.Name,
				State:    awsState,
				BoxAwsID: localDb.StringValue(rec.BoxAwsID),
				Region:   region,
				Provider: providerForSnapshotRecord(rec.SnapshotRecord, region),
			})
			return nil
		},
	)
	if err != nil {
		return nil, err
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

	ec2Client, err := r.ec2ForSnapshotRecord(*record)
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
	if localDb.StringValue(record.State) != awsState {
		if err := db.UpdateSnapshotState(amiID, awsState); err != nil {
			return nil, err
		}
	}

	boxAwsID := ""
	if boxID := localDb.StringValue(record.BoxID); boxID != "" {
		box, err := db.GetInstanceByID(boxID)
		if err == nil {
			boxAwsID = box.AwsInstanceID
		}
	}

	region, err := regionForSnapshotRecord(*record)
	if err != nil {
		return nil, err
	}

	return &Snapshot{
		AmiID:    record.AmiID,
		Name:     record.Name,
		State:    awsState,
		BoxAwsID: boxAwsID,
		Region:   region,
		Provider: providerForSnapshotRecord(*record, region),
	}, nil
}

// DeleteSnapshot removes a snapshot AMI and its backing EBS snapshots from AWS,
// then deletes the local DB record. Mirrors Lighthouse Ec2Service.deleteSnapshot.
func (r *Runtime) DeleteSnapshot(amiID, userID string) error {
	db := r.DB()

	record, err := db.GetSnapshotByAmiIDAndUserID(amiID, userID) // check if snapshot exists for user
	if err == sql.ErrNoRows {
		return fmt.Errorf("snapshot not found: %s", amiID)
	}
	if err != nil {
		return err
	}

	ec2Client, err := r.ec2ForSnapshotRecord(*record)
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

// regionForSnapshotRecord returns the snapshot's stored region. Region is
// captured at create time (and backfilled by migrateSnapshotsRegionProvider);
// there is no box/config fallback.
func regionForSnapshotRecord(record localDb.SnapshotRecord) (string, error) {
	if region := localDb.StringValue(record.Region); region != "" {
		return region, nil
	}
	return "", fmt.Errorf("snapshot %s has no stored region", record.AmiID)
}

// providerForSnapshotRecord returns the provider for the snapshot record.
func providerForSnapshotRecord(record localDb.SnapshotRecord, region string) string {
	if provider := localDb.StringValue(record.Provider); provider != "" {
		return provider
	}
	return ProviderForRegion(region)
}

func (r *Runtime) ec2ForSnapshotRecord(record localDb.SnapshotRecord) (*ec2.Client, error) {
	region, err := regionForSnapshotRecord(record)
	if err != nil {
		return nil, err
	}
	return r.EC2ForRegion(region)
}
