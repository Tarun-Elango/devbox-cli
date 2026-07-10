package cmd

import (
	"fmt"
	"os"
	"time"

	"outpost-cli/helper"
	"outpost-cli/internal/config"
	"outpost-cli/service"
	"outpost-cli/service/localDb"
)

func Health(args []string) {
	helper.RejectExtraArgs(args, "usage: outpost health")
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
		check("region", "not set", "run: outpost setup")
	} else {
		check("region", "ok", cfg.AwsRegion)
	}

	if cfg.AwsAccessKey == "" && cfg.AwsSecret == "" {
		check("aws creds", "not configured", "run: outpost setup")
		check("aws updated", "n/a", "")
	} else {
		ctx, cancel := helper.CommandContext()
		defer cancel()
		creds := service.CheckAWSCredentials(ctx)
		check("aws creds", creds.Status, creds.Detail)

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
