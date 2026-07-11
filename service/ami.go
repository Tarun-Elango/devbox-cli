package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"

	awsclient "outpost-cli/service/aws"
)

// ResolveAMIForOS looks up the latest ARM64 AMI for family in region via AWS
// public SSM parameters.
func (r *Runtime) ResolveAMIForOS(ctx context.Context, region, osFamily string) (string, error) {
	profile, ok := OSProfileFor(osFamily)
	if !ok {
		return "", fmt.Errorf("unsupported os family %q", osFamily)
	}
	if region == "" {
		return "", fmt.Errorf("aws region is required; run: outpost setup")
	}

	client, err := r.SSMForRegion(region) // get the ssm client for the region
	if err != nil {
		return "", err
	}

	resp, err := client.GetParameter(ctx, &ssm.GetParameterInput{
		Name: aws.String(profile.SSMParameter),
	})
	if err != nil {
		return "", awsclient.WrapError(
			fmt.Sprintf("resolve AMI for %s via SSM parameter %s", profile.DisplayName, profile.SSMParameter),
			err,
		)
	}
	if resp.Parameter == nil {
		return "", fmt.Errorf("SSM parameter %s returned no value for %s in %s", profile.SSMParameter, profile.DisplayName, region)
	}
	amiID := strings.TrimSpace(aws.ToString(resp.Parameter.Value))
	if amiID == "" || !strings.HasPrefix(amiID, "ami-") {
		return "", fmt.Errorf("SSM parameter %s returned invalid AMI id %q for %s in %s", profile.SSMParameter, amiID, profile.DisplayName, region)
	}
	return amiID, nil
}
