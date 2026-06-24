// configure the aws sdk and create a client

package aws

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"

	appconfig "devbox-cli/internal/config"
)

const awsRequestTimeout = 60 * time.Second

// aws key and secret are in .devbox/config.json
// we need to load the config and create a client

// Client wraps a configured AWS SDK v2 runtime.
type Client struct {
	cfg aws.Config
}

// LoadConfig reads ~/.devbox/config.json.
func LoadConfig() (*appconfig.Config, error) {
	return appconfig.Load() // return config struct, and error
}

// NewClient loads app credentials and builds an AWS SDK config.
func NewClient(ctx context.Context) (*Client, error) {
	appCfg, err := LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}
	if appCfg.AwsSecret == "" || appCfg.AwsAccessKey == "" {
		return nil, fmt.Errorf("aws secret and access key are required")
	}
	if appCfg.AwsRegion == "" {
		return nil, fmt.Errorf("aws region is required; run: devbox setup")
	}

	opts := []func(*awsconfig.LoadOptions) error{
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			appCfg.AwsAccessKey,
			appCfg.AwsSecret,
			"",
		)),
		awsconfig.WithRegion(appCfg.AwsRegion),
		awsconfig.WithRetryMode(aws.RetryModeAdaptive),
		awsconfig.WithHTTPClient(&http.Client{Timeout: awsRequestTimeout}),
	}
	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}

	return &Client{cfg: awsCfg}, nil
}

// Config returns the underlying AWS SDK config for creating service clients.
func (c *Client) Config() aws.Config {
	return c.cfg
}
