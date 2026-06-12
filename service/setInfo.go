package service

import "devbox-cli/internal/config"

// Region is an AWS region id with a human-readable label.
type Region struct {
	ID   string
	Name string
}

// AllRegions returns every standard AWS commercial region for the setup picker.
func AllRegions() []Region {
	return []Region{
		{ID: "us-east-1", Name: "US East (N. Virginia)"},
		{ID: "us-east-2", Name: "US East (Ohio)"},
		{ID: "us-west-1", Name: "US West (N. California)"},
		{ID: "us-west-2", Name: "US West (Oregon)"},
		{ID: "af-south-1", Name: "Africa (Cape Town)"},
		{ID: "ap-east-1", Name: "Asia Pacific (Hong Kong)"},
		{ID: "ap-south-1", Name: "Asia Pacific (Mumbai)"},
		{ID: "ap-south-2", Name: "Asia Pacific (Hyderabad)"},
		{ID: "ap-southeast-1", Name: "Asia Pacific (Singapore)"},
		{ID: "ap-southeast-2", Name: "Asia Pacific (Sydney)"},
		{ID: "ap-southeast-3", Name: "Asia Pacific (Jakarta)"},
		{ID: "ap-southeast-4", Name: "Asia Pacific (Melbourne)"},
		{ID: "ap-northeast-1", Name: "Asia Pacific (Tokyo)"},
		{ID: "ap-northeast-2", Name: "Asia Pacific (Seoul)"},
		{ID: "ap-northeast-3", Name: "Asia Pacific (Osaka)"},
		{ID: "ca-central-1", Name: "Canada (Central)"},
		{ID: "ca-west-1", Name: "Canada West (Calgary)"},
		{ID: "eu-central-1", Name: "Europe (Frankfurt)"},
		{ID: "eu-central-2", Name: "Europe (Zurich)"},
		{ID: "eu-west-1", Name: "Europe (Ireland)"},
		{ID: "eu-west-2", Name: "Europe (London)"},
		{ID: "eu-west-3", Name: "Europe (Paris)"},
		{ID: "eu-north-1", Name: "Europe (Stockholm)"},
		{ID: "eu-south-1", Name: "Europe (Milan)"},
		{ID: "eu-south-2", Name: "Europe (Spain)"},
		{ID: "il-central-1", Name: "Israel (Tel Aviv)"},
		{ID: "me-central-1", Name: "Middle East (UAE)"},
		{ID: "me-south-1", Name: "Middle East (Bahrain)"},
		{ID: "sa-east-1", Name: "South America (São Paulo)"},
	}
}

// SaveAWSCredentials stores AWS credentials and region in ~/.devbox/config.json.
func SaveAWSCredentials(secret, accessKey, region string) error {
	cfg, err := config.Load() // load
	if err != nil {
		return err
	}
	cfg.AwsSecret = secret
	cfg.AwsAccessKey = accessKey
	cfg.AwsRegion = region
	cfg.Mode = "local"
	return config.Save(cfg)
}


// EnsureLocalModeAndGetCurrMode returns the configured mode. If unset, it
// persists "local" and returns "local".
func EnsureLocalModeAndGetCurrMode() (string, error) {
	cfg, err := config.Load()
	if err != nil {
		return "", err
	}
	if cfg.Mode != "" {
		return cfg.Mode, nil
	}
	cfg.Mode = "local"
	if err := config.Save(cfg); err != nil {
		return "", err
	}
	return "local", nil
}
