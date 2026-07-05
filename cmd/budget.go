package cmd

import (
	"fmt"
	"os"
	"strings"

	"devbox-cli/helper"
	"devbox-cli/service"
	awsclient "devbox-cli/service/aws"
)

const budgetUsage = "usage: devbox budget [ls] [--refresh]"

// Budget dispatches budget sub-commands.
//
//	devbox budget            → list all account budgets
//	devbox budget ls         → same as above
//	devbox budget [ls] --refresh → bypass the local cache and refetch
func Budget(args []string) {
	if len(args) == 0 {
		budgetList(args)
		return
	}

	switch args[0] {
	case "ls":
		budgetList(args[1:])
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
		// if the error is a permission error, print a hint
		if awsclient.IsPermissionError(err) {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			fmt.Fprintln(os.Stderr, "hint: IAM action budgets:ViewBudget is required to list budgets")
		} else {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
		}
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

func orDash(s string) string {
	if s == "" {
		return "—"
	}
	return s
}
