package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"

	awsclient "outpost-cli/service/aws"
)

// ResolveAMIForOS looks up the latest AMI for family and arch in region via AWS
// public SSM parameters.
func (r *Runtime) ResolveAMIForOS(ctx context.Context, region, osFamily, arch string) (string, error) {
	profile, ok := OSProfileFor(osFamily)
	if !ok {
		return "", fmt.Errorf("unsupported os family %q", osFamily)
	}
	if region == "" {
		return "", fmt.Errorf("aws region is required; run: outpost setup")
	}

	param, err := profile.SSMParameterForArch(arch)
	if err != nil {
		return "", err
	}

	client, err := r.SSMForRegion(region) // get the ssm client for the region
	if err != nil {
		return "", err
	}

	resp, err := client.GetParameter(ctx, &ssm.GetParameterInput{
		Name: aws.String(param),
	})
	if err != nil {
		return "", awsclient.WrapError(
			fmt.Sprintf("resolve AMI for %s (%s) via SSM parameter %s", profile.DisplayName, arch, param),
			err,
		)
	}
	if resp.Parameter == nil {
		return "", fmt.Errorf("SSM parameter %s returned no value for %s in %s", param, profile.DisplayName, region)
	}
	amiID := strings.TrimSpace(aws.ToString(resp.Parameter.Value))
	if amiID == "" || !strings.HasPrefix(amiID, "ami-") {
		return "", fmt.Errorf("SSM parameter %s returned invalid AMI id %q for %s in %s", param, amiID, profile.DisplayName, region)
	}
	return amiID, nil
}
