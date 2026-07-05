// list AWS Budgets for the configured account (read-only; show only, no create/edit)

package service

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/budgets"
	budgetstypes "github.com/aws/aws-sdk-go-v2/service/budgets/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"

	awsclient "devbox-cli/service/aws"
)

// budgetsRegion is where the Budgets API lives; billing is global but the SDK
// endpoint is pinned to us-east-1 regardless of the box region.
const budgetsRegion = "us-east-1"

const budgetCacheFile = "budgets-cache.json" // no need to back up this file

// budgetCacheTTL bounds how long a cached budgets response is reused.
const budgetCacheTTL = 12 * time.Hour

// BudgetSummary is a simplified view of an AWS Budget for display.
type BudgetSummary struct {
	Name            string
	Type            string
	Period          string
	Limit           string
	ActualSpend     string
	ForecastedSpend string
	LastUpdated     time.Time
	PctOfBudget     float64 // -1 when it can't be computed (e.g. zero/missing limit)
}

// ListBudgetsOptions configures ListBudgets.
type ListBudgetsOptions struct {
	// Refresh forces a live API call, bypassing any cached response.
	Refresh bool
}

// ListBudgetsResult reports the budgets along with whether cached data was used.
type ListBudgetsResult struct {
	Budgets   []BudgetSummary
	Cached    bool
	FetchedAt time.Time
}

type budgetCachePayload struct {
	FetchedAt time.Time       `json:"fetchedAt"`
	AccountID string          `json:"accountId"`
	Budgets   []BudgetSummary `json:"budgets"`
}

// ListBudgets returns all budgets for the account, using a local cache when
// fresh (see budgetCacheTTL) unless opts.Refresh is set.
func ListBudgets(ctx context.Context, opts ListBudgetsOptions) (ListBudgetsResult, error) {
	accountID, err := accountIDForBudgets(ctx)
	if err != nil {
		return ListBudgetsResult{}, err
	}

	if !opts.Refresh {
		// if the data is cached, return the cached data
		if cached, ok := readBudgetCache(accountID); ok {
			return ListBudgetsResult{Budgets: cached.Budgets, Cached: true, FetchedAt: cached.FetchedAt}, nil
		}
	}

	client, err := awsclient.NewClientForRegion(ctx, budgetsRegion) // get the client for the budgets region
	if err != nil {
		return ListBudgetsResult{}, err
	}
	budgetsClient := budgets.NewFromConfig(client.Config()) // create the budgets client

	summaries, err := describeAllBudgets(ctx, budgetsClient, accountID) // describe all the budgets
	if err != nil {
		return ListBudgetsResult{}, awsclient.WrapError("list budgets", err)
	}

	fetchedAt := time.Now()
	writeBudgetCache(budgetCachePayload{FetchedAt: fetchedAt, AccountID: accountID, Budgets: summaries}) // write the cache

	return ListBudgetsResult{Budgets: summaries, Cached: false, FetchedAt: fetchedAt}, nil
}

// accountIDForBudgets resolves the AWS account ID via STS, using the box
// region client (STS is regional-agnostic enough for this call).
// returns the account ID
func accountIDForBudgets(ctx context.Context) (string, error) {
	client, err := awsclient.NewClient(ctx)
	if err != nil {
		return "", err
	}
	out, err := sts.NewFromConfig(client.Config()).GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return "", awsclient.WrapError("get caller identity", err)
	}
	return aws.ToString(out.Account), nil
}

func describeAllBudgets(ctx context.Context, client *budgets.Client, accountID string) ([]BudgetSummary, error) {
	var summaries []BudgetSummary

	paginator := budgets.NewDescribeBudgetsPaginator(client, &budgets.DescribeBudgetsInput{
		AccountId: aws.String(accountID),
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, b := range page.Budgets {
			summaries = append(summaries, toBudgetSummary(b))
		}
	}

	return summaries, nil
}

// toBudgetSummary converts a AWS Budget to a BudgetSummary object
func toBudgetSummary(b budgetstypes.Budget) BudgetSummary {
	s := BudgetSummary{
		Name:        aws.ToString(b.BudgetName),
		Type:        string(b.BudgetType),
		Period:      string(b.TimeUnit),
		PctOfBudget: -1,
	}

	if b.BudgetLimit != nil {
		s.Limit = formatSpend(b.BudgetLimit.Amount, b.BudgetLimit.Unit)
	}
	if b.CalculatedSpend != nil {
		if b.CalculatedSpend.ActualSpend != nil {
			s.ActualSpend = formatSpend(b.CalculatedSpend.ActualSpend.Amount, b.CalculatedSpend.ActualSpend.Unit)
		}
		if b.CalculatedSpend.ForecastedSpend != nil {
			s.ForecastedSpend = formatSpend(b.CalculatedSpend.ForecastedSpend.Amount, b.CalculatedSpend.ForecastedSpend.Unit)
		}
	}
	if b.LastUpdatedTime != nil {
		s.LastUpdated = *b.LastUpdatedTime
	}

	if b.BudgetLimit != nil && b.CalculatedSpend != nil && b.CalculatedSpend.ActualSpend != nil {
		limitAmt, limitErr := strconv.ParseFloat(aws.ToString(b.BudgetLimit.Amount), 64)
		actualAmt, actualErr := strconv.ParseFloat(aws.ToString(b.CalculatedSpend.ActualSpend.Amount), 64)
		if limitErr == nil && actualErr == nil && limitAmt > 0 {
			s.PctOfBudget = actualAmt / limitAmt * 100
		}
	}

	return s
}

func formatSpend(amount, unit *string) string {
	amt := aws.ToString(amount)
	u := aws.ToString(unit)
	if amt == "" {
		return ""
	}
	if u == "" {
		return amt
	}
	return fmt.Sprintf("%s %s", amt, u)
}

func budgetCachePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home dir: %w", err)
	}
	return filepath.Join(home, ".devbox", budgetCacheFile), nil
}

func readBudgetCache(accountID string) (budgetCachePayload, bool) {
	path, err := budgetCachePath()
	if err != nil {
		return budgetCachePayload{}, false
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return budgetCachePayload{}, false
	}
	var payload budgetCachePayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return budgetCachePayload{}, false
	}
	if payload.AccountID != accountID {
		return budgetCachePayload{}, false
	}
	if time.Since(payload.FetchedAt) > budgetCacheTTL { // if the data is older than the cache TTL, return false
		return budgetCachePayload{}, false
	}
	return payload, true
}

func writeBudgetCache(payload budgetCachePayload) {
	path, err := budgetCachePath()
	if err != nil {
		return
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return
	}
	tmpFile, err := os.CreateTemp(dir, budgetCacheFile+".tmp-*")
	if err != nil {
		return
	}
	tmpPath := tmpFile.Name()
	if _, err := tmpFile.Write(data); err != nil {
		_ = tmpFile.Close()
		_ = os.Remove(tmpPath)
		return
	}
	if err := tmpFile.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return
	}
	if err := os.Chmod(tmpPath, 0600); err != nil {
		_ = os.Remove(tmpPath)
		return
	}
	_ = os.Rename(tmpPath, path)
}
