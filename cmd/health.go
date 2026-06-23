package cmd

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	_ "modernc.org/sqlite"

	"devbox-cli/internal/config"
	awsclient "devbox-cli/service/aws"
	"devbox-cli/service/localDb"
)

func Health(args []string) {
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
		client, err := awsclient.NewClient(context.Background())
		if err != nil {
			check("aws creds", "error", err.Error())
		} else {
			out, err := sts.NewFromConfig(client.Config()).GetCallerIdentity(context.Background(), &sts.GetCallerIdentityInput{})
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

	dbPath, err := localDb.DBPath()
	if err != nil {
		check("database", "error", err.Error())
	} else if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		check("database", "missing", dbPath)
	} else if err != nil {
		check("database", "error", err.Error())
	} else {
		conn, err := sql.Open("sqlite", dbPath)
		if err != nil {
			check("database", "unavailable", err.Error())
		} else {
			defer func() { _ = conn.Close() }()
			if err := conn.Ping(); err != nil {
				check("database", "unavailable", err.Error())
			} else {
				check("database", "ok", dbPath)
			}
		}
	}

	if failed {
		os.Exit(1)
	}
}
