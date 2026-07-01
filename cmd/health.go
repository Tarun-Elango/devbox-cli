package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sts"

	"devbox-cli/helper"
	"devbox-cli/internal/config"
	awsclient "devbox-cli/service/aws"
	"devbox-cli/service/localDb"
)

func Health(args []string) {
	helper.RejectExtraArgs(args, "usage: devbox health")
	var failed bool

	// helper closure to check the health of the system
	check := func(name, status, detail string) {
		if detail != "" {
			fmt.Printf("%-14s %s (%s)\n", name+":", status, detail)
		} else {
			fmt.Printf("%-14s %s\n", name+":", status)
		}
		if status != "ok" && status != "unknown" {
			failed = true
		}
	}

	cfgPath, err := config.ConfigPath()
	var cfg *config.Config
	if err != nil {
		check("config", "error", err.Error())
		cfg = &config.Config{}
	} else if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		check("config", "missing", cfgPath)
		cfg = &config.Config{}
	} else if err != nil {
		check("config", "error", err.Error())
		cfg = &config.Config{}
	} else {
		cfg, err = config.Load()
		if err != nil {
			check("config", "invalid", err.Error())
			cfg = &config.Config{}
		} else {
			check("config", "ok", cfgPath)
		}
	}

	if cfg.AwsRegion == "" {
		check("region", "not set", "run: devbox setup")
	} else {
		check("region", "ok", cfg.AwsRegion)
	}

	if cfg.AwsAccessKey == "" && cfg.AwsSecret == "" {
		check("aws creds", "not configured", "run: devbox setup")
		check("aws updated", "n/a", "")
	} else {
		ctx, cancel := helper.CommandContext()
		defer cancel() // cancel the context if the aws call fails
		client, err := awsclient.NewClient(ctx)
		if err != nil {
			check("aws creds", "error", err.Error())
		} else {
			// pass ctx, so there is a timeout for the aws call
			out, err := sts.NewFromConfig(client.Config()).GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
			if err != nil {
				check("aws creds", "invalid", awsclient.ShortMessage(err))
			} else {
				check("aws creds", "ok", "account "+aws.ToString(out.Account))
			}
		}

		if cfg.AwsCredsUpdatedAt.IsZero() {
			check("aws updated", "unknown", "")
		} else {
			check("aws updated", "ok", cfg.AwsCredsUpdatedAt.Local().Format(time.RFC3339))
		}
	}

	db, err := localDb.Open()
	if err != nil {
		check("database", "unavailable", err.Error())
	} else {
		defer func() { _ = db.Close() }()
		dbPath, _ := localDb.DBPath()
		check("database", "ok", dbPath)
	}

	if failed {
		os.Exit(1)
	}
}
