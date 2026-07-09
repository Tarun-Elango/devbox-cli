package cmd

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"devbox-cli/helper"
	"devbox-cli/service"
)

const budgetUsage = "usage: devbox budget [ls] [--refresh] | create <name> <limit> <email> | update <name> | delete <name>"

// Budget dispatches budget sub-commands.
//
//	devbox budget            → list all account budgets
//	devbox budget ls         → same as above
//	devbox budget [ls] --refresh → bypass the local cache and refetch
//	devbox budget create <name> <limit> <email> → create a monthly cost budget
//	devbox budget update <name> → interactively update name, limit, or alert email
//	devbox budget delete <name> → delete a budget by exact name
func Budget(args []string) {
	if len(args) == 0 {
		budgetList(args)
		return
	}

	switch args[0] {
	case "ls":
		budgetList(args[1:])
	case "create":
		budgetCreate(args[1:])
	case "update":
		budgetUpdate(args[1:])
	case "delete":
		budgetDelete(args[1:])
	case "--refresh":
		budgetList(args)
	default:
		fmt.Fprintf(os.Stderr, "budget: unknown sub-command %q\n", args[0])
		fmt.Fprintln(os.Stderr, budgetUsage)
		os.Exit(1)
	}
}

func budgetList(args []string) {
	refresh := false
	for _, a := range args { // check for the refresh flag
		if a == "--refresh" {
			refresh = true
			continue
		}
		fmt.Fprintf(os.Stderr, "budget: unexpected argument %q\n", a)
		fmt.Fprintln(os.Stderr, budgetUsage)
		os.Exit(1)
	}

	fmt.Println("Listing account budgets")
	ctx, cancel := helper.CommandContext()
	defer cancel()

	result, err := service.ListBudgets(ctx, service.ListBudgetsOptions{Refresh: refresh})
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	if len(result.Budgets) == 0 { // if no budgets found, print a message
		fmt.Println("No budgets found.")
		return
	}

	if result.Cached { // if the data is cached, print a message
		fmt.Printf("(showing cached data from %s; use --refresh to refetch)\n", result.FetchedAt.Local().Format("2006-01-02 15:04:05"))
	}

	printBudgetTable(result.Budgets)
}

func budgetCreate(args []string) {
	const usage = "usage: devbox budget create <name> <limit> <email>"
	if len(args) != 3 {
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(1)
	}

	name := strings.TrimSpace(args[0])
	limit, err := strconv.ParseFloat(strings.TrimSpace(args[1]), 64)
	if err != nil || limit <= 0 {
		fmt.Fprintln(os.Stderr, "error: limit must be a positive number (USD)")
		os.Exit(1)
	}
	email := strings.TrimSpace(args[2])

	fmt.Printf("Creating monthly cost budget %q (limit: %.2f USD, alerts to %s)\n", name, limit, email)
	ctx, cancel := helper.CommandContext()
	defer cancel()

	if err := service.CreateBudget(ctx, service.CreateBudgetOptions{
		Name:     name,
		LimitUSD: limit,
		Email:    email,
	}); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Budget created.")
	fmt.Println("Alerts: 85% actual spend, 100% actual spend, 100% forecasted spend.")
	fmt.Println("Scope: all AWS services.")
	fmt.Println()
	fmt.Println("Listing account budgets")

	result, err := service.ListBudgets(ctx, service.ListBudgetsOptions{Refresh: true})
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	if len(result.Budgets) == 0 {
		fmt.Println("No budgets found.")
		return
	}
	printBudgetTable(result.Budgets)
}

func budgetDelete(args []string) {
	const usage = "usage: devbox budget delete <name>"
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(1)
	}

	name := strings.TrimSpace(strings.Join(args, " "))
	if name == "" {
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(1)
	}

	fmt.Printf("Deleting budget %q\n", name)
	ctx, cancel := helper.CommandContext() // get the command context
	defer cancel()

	if err := service.DeleteBudget(ctx, name); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Budget deleted.")
}

func budgetUpdate(args []string) {
	const usage = "usage: devbox budget update <name>"
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(1)
	}

	name := strings.TrimSpace(strings.Join(args, " "))
	if name == "" {
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(1)
	}

	ctx, cancel := helper.CommandContext()
	defer cancel()

	details, err := service.GetBudgetDetails(ctx, name)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Updating budget %q. Press Enter to keep current values.\n\n", details.Name)

	fmt.Printf("Budget name [%s]: ", details.Name)
	newName, err := helper.ReadStdinLine()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading name: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Monthly limit USD [%.2f]: ", details.LimitUSD)
	limitLine, err := helper.ReadStdinLine()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading limit: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Alert email [%s]: ", details.Email)
	newEmail, err := helper.ReadStdinLine()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading email: %v\n", err)
		os.Exit(1)
	}

	// create the update budget options
	opts := service.UpdateBudgetOptions{
		CurrentName: details.Name,
		NewName:     strings.TrimSpace(newName),
		NewEmail:    strings.TrimSpace(newEmail),
	}
	if strings.TrimSpace(limitLine) != "" {
		limit, err := strconv.ParseFloat(strings.TrimSpace(limitLine), 64)
		if err != nil || limit <= 0 {
			fmt.Fprintln(os.Stderr, "error: limit must be a positive number (USD)")
			os.Exit(1)
		}
		opts.NewLimitUSD = &limit
	}

	changed := (opts.NewName != "" && opts.NewName != details.Name) ||
		(opts.NewLimitUSD != nil && *opts.NewLimitUSD != details.LimitUSD) ||
		(opts.NewEmail != "" && opts.NewEmail != details.Email)
	if !changed {
		fmt.Println("No changes.")
		return
	}

	if err := service.UpdateBudget(ctx, opts); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	finalName := details.Name
	if opts.NewName != "" {
		finalName = opts.NewName
	}
	fmt.Printf("Budget %q updated.\n", finalName)

	fmt.Println()
	fmt.Println("Listing account budgets")

	result, err := service.ListBudgets(ctx, service.ListBudgetsOptions{Refresh: true})
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	if len(result.Budgets) == 0 {
		fmt.Println("No budgets found.")
		return
	}
	printBudgetTable(result.Budgets)
}

// helper: printBudgetTable prints a table of budget summaries.
func printBudgetTable(budgets []service.BudgetSummary) {
	fmt.Printf("%-28s  %-8s  %-10s  %-14s  %-14s  %-14s  %s\n",
		"NAME", "TYPE", "PERIOD", "LIMIT", "SPENT", "FORECAST", "% BUDGET")
	fmt.Println(strings.Repeat("-", 120))
	for _, b := range budgets {
		pct := "—"
		if b.PctOfBudget >= 0 {
			pct = fmt.Sprintf("%.0f%%", b.PctOfBudget)
		}
		fmt.Printf("%-28s  %-8s  %-10s  %-14s  %-14s  %-14s  %s\n",
			b.Name, b.Type, b.Period, orDash(b.Limit), orDash(b.ActualSpend), orDash(b.ForecastedSpend), pct)
	}
}

// helper: orDash returns a dash if the string is empty, otherwise the string itself.
func orDash(s string) string {
	if s == "" {
		return "—"
	}
	return s
}
