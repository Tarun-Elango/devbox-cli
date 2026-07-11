package service

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/google/uuid"

	awsclient "outpost-cli/service/aws"
	localDb "outpost-cli/service/localDb"
)

const (
	ImportKindBox      = "box"
	ImportKindSnapshot = "snapshot"
)

// ImportCandidate is an AWS resource in the configured region not yet in the local DB.
type ImportCandidate struct {
	Kind         string
	AWSID        string
	Name         string
	State        string
	InstanceType string
	IPAddress    string
	OSFamily     string
	// NeedsOSPrompt is true when the image looks Linux but the distro is unknown.
	NeedsOSPrompt bool
	// SkipReason explains why a resource was excluded from interactive import.
	SkipReason string
}

// ListUntrackedImportCandidates returns EC2 instances and self-owned AMIs in
// the configured region that are not already tracked for userID.
// Non-Linux images are omitted with SkipReason left empty (filtered out).
func (r *Runtime) ListUntrackedImportCandidates(userID string) ([]ImportCandidate, error) {
	cfg, err := awsclient.LoadConfig()
	if err != nil {
		return nil, err
	}
	region := cfg.AwsRegion
	if region == "" {
		return nil, fmt.Errorf("aws region is required; run: outpost setup")
	}

	ec2Client, err := r.EC2ForRegion(region)
	if err != nil {
		return nil, err
	}
	ctx := r.Context()
	db := r.DB()

	var out []ImportCandidate
	instanceNames := make(map[string]struct{})
	imageCache := make(map[string]types.Image)

	instancePages := ec2.NewDescribeInstancesPaginator(ec2Client, &ec2.DescribeInstancesInput{
		Filters: []types.Filter{{
			Name:   aws.String("instance-state-name"),
			Values: []string{"pending", "running", "stopping", "stopped"},
		}},
	})
	for instancePages.HasMorePages() {
		instResp, err := instancePages.NextPage(ctx)
		if err != nil {
			return nil, awsclient.WrapError("describe instances", err)
		}
		for _, reservation := range instResp.Reservations {
			for _, inst := range reservation.Instances {
				awsID := aws.ToString(inst.InstanceId)
				_, err := db.GetInstanceByAwsInstanceIDAndUserID(awsID, userID)
				if err == nil { // if the instance is already in the database, skip it
					continue
				}
				if err != sql.ErrNoRows {
					return nil, err
				}
				dto := instanceFromAWS(inst)
				name, err := uniqueImportName(db, userID, dto.Name, awsID, false, instanceNames)
				if err != nil {
					return nil, err
				}
				ip := dto.IPAddress
				if ip == "" {
					ip = dto.PrivateIPAddress
				}

				osFamily, needsPrompt, skip := r.classifyInstanceImport(ec2Client, imageCache, inst)
				if skip {
					continue
				}

				out = append(out, ImportCandidate{
					Kind:          ImportKindBox,
					AWSID:         awsID,
					Name:          name,
					State:         dto.Status,
					InstanceType:  dto.InstanceType,
					IPAddress:     ip,
					OSFamily:      osFamily,
					NeedsOSPrompt: needsPrompt,
				})
			}
		}
	}

	snapshotNames := make(map[string]struct{})
	imagePages := ec2.NewDescribeImagesPaginator(ec2Client, &ec2.DescribeImagesInput{
		Owners: []string{"self"},
	})
	for imagePages.HasMorePages() {
		imgResp, err := imagePages.NextPage(ctx)
		if err != nil {
			return nil, awsclient.WrapError("describe images", err)
		}
		for _, img := range imgResp.Images {
			amiID := aws.ToString(img.ImageId)
			_, err := db.GetSnapshotByAmiIDAndUserID(amiID, userID)
			if err == nil {
				continue
			}
			if err != sql.ErrNoRows {
				return nil, err
			}
			name, err := uniqueImportName(db, userID, aws.ToString(img.Name), amiID, true, snapshotNames)
			if err != nil {
				return nil, err
			}
			family, isLinux := ClassifyImageOSFamily(img)
			if !isLinux {
				continue
			}
			out = append(out, ImportCandidate{
				Kind:          ImportKindSnapshot,
				AWSID:         amiID,
				Name:          name,
				State:         string(img.State),
				OSFamily:      family,
				NeedsOSPrompt: family == "",
			})
		}
	}

	return out, nil
}

func (r *Runtime) classifyInstanceImport(ec2Client *ec2.Client, cache map[string]types.Image, inst types.Instance) (osFamily string, needsPrompt bool, skip bool) {
	imageID := aws.ToString(inst.ImageId)
	if imageID == "" {
		return "", true, false
	}
	img, ok := cache[imageID]
	if !ok {
		resp, err := ec2Client.DescribeImages(r.Context(), &ec2.DescribeImagesInput{
			ImageIds: []string{imageID},
		})
		if err != nil || len(resp.Images) == 0 {
			return "", true, false
		}
		img = resp.Images[0]
		cache[imageID] = img
	}
	family, isLinux := ClassifyImageOSFamily(img)
	if !isLinux {
		return "", false, true
	}
	if family == "" {
		return "", true, false
	}
	return family, false, false
}

// ImportCandidate inserts a listed candidate into the local DB.
func (r *Runtime) ImportCandidate(c ImportCandidate, userID string) error {
	cfg, err := awsclient.LoadConfig()
	if err != nil {
		return err
	}
	region := cfg.AwsRegion
	provider := ProviderForRegion(region)
	db := r.DB()

	osFamily := NormalizeOSFamily(c.OSFamily)
	if osFamily == "" {
		osFamily = DefaultOSFamily
	}
	if err := ValidateOSFamily(osFamily); err != nil {
		return err
	}

	switch c.Kind {
	case ImportKindBox:
		if err := db.InsertInstance(
			uuid.New().String(), c.AWSID, c.Name, userID,
			c.State, c.InstanceType, region, provider, osFamily,
		); err != nil {
			return err
		}
		// ponytail: InsertInstance has no IP column arg; sync only when we have one
		if c.IPAddress != "" {
			return db.SyncInstanceFromAWS(c.AWSID, c.State, c.IPAddress, c.InstanceType, "")
		}
		return nil
	case ImportKindSnapshot:
		return db.InsertSnapshot(
			uuid.New().String(), c.AWSID, c.Name, userID,
			"", c.State, region, provider, osFamily,
		)
	default:
		return fmt.Errorf("unknown import kind: %s", c.Kind)
	}
}

// uniqueImportName: AWS name/tag, else imported-<id>, else -2/-3 on collision.
// ponytail: linear scan; fine for interactive import
func uniqueImportName(db *localDb.DB, userID, preferred, awsID string, snapshot bool, batchNames map[string]struct{}) (string, error) {
	base := strings.TrimSpace(preferred)
	if base == "" || looksLikeAWSResourceID(base) {
		base = "imported-" + shortAWSID(awsID)
	}
	for i := 1; i <= 100; i++ {
		name := base
		if i > 1 {
			name = fmt.Sprintf("%s-%d", base, i)
		}
		if _, used := batchNames[name]; used {
			continue
		}
		var err error
		if snapshot {
			err = db.ValidateSnapshotNameAvailable(name, userID)
		} else {
			err = db.ValidateInstanceNameAvailable(name, userID)
		}
		if err == nil {
			batchNames[name] = struct{}{}
			return name, nil
		}
		if !strings.Contains(err.Error(), "already exists") {
			return "", err
		}
	}
	return "", fmt.Errorf("could not find free name for %s", awsID)
}

func looksLikeAWSResourceID(s string) bool {
	s = strings.ToLower(strings.TrimSpace(s))
	return strings.HasPrefix(s, "i-") || strings.HasPrefix(s, "ami-")
}

func shortAWSID(awsID string) string {
	if i := strings.LastIndex(awsID, "-"); i >= 0 {
		awsID = awsID[i+1:]
	}
	if len(awsID) > 8 {
		return awsID[len(awsID)-8:]
	}
	return awsID
}
