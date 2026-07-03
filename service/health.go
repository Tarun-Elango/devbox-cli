package service

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sts"

	awsclient "devbox-cli/service/aws"
)

// AWSCredsCheck is the result of validating AWS credentials via STS.
type AWSCredsCheck struct {
	Status string
	Detail string
}

// CheckAWSCredentials verifies configured credentials by calling STS GetCallerIdentity.
func CheckAWSCredentials(ctx context.Context) AWSCredsCheck {
	client, err := awsclient.NewClient(ctx)
	if err != nil {
		return AWSCredsCheck{Status: "error", Detail: err.Error()}
	}

	out, err := sts.NewFromConfig(client.Config()).GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return AWSCredsCheck{Status: "invalid", Detail: awsclient.ShortMessage(err)}
	}

	return AWSCredsCheck{Status: "ok", Detail: "account " + aws.ToString(out.Account)}
}
