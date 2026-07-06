// AWS Budgets for the configured account (list, create, and delete).

package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/budgets"
	budgetstypes "github.com/aws/aws-sdk-go-v2/service/budgets/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	smithy "github.com/aws/smithy-go"

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

// CreateBudgetOptions configures CreateBudget.
type CreateBudgetOptions struct {
	Name     string
	LimitUSD float64
	Email    string
}

// BudgetDetails holds the editable fields of an existing budget.
type BudgetDetails struct {
	Name     string
	LimitUSD float64
	Email    string
}

// UpdateBudgetOptions configures UpdateBudget. Empty NewName or NewEmail, or nil
// NewLimitUSD, means keep the current value.
type UpdateBudgetOptions struct {
	CurrentName string
	NewName     string
	NewLimitUSD *float64
	NewEmail    string
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
		return ListBudgetsResult{}, wrapBudgetError("list budgets", err)
	}

	fetchedAt := time.Now()
	writeBudgetCache(budgetCachePayload{FetchedAt: fetchedAt, AccountID: accountID, Budgets: summaries}) // write the cache

	return ListBudgetsResult{Budgets: summaries, Cached: false, FetchedAt: fetchedAt}, nil
}

// CreateBudget creates a monthly cost budget for all AWS services with the standard
// alert thresholds: 85% actual, 100% actual, and 100% forecasted spend.
func CreateBudget(ctx context.Context, opts CreateBudgetOptions) error {
	name := strings.TrimSpace(opts.Name)
	email := strings.TrimSpace(opts.Email)
	if name == "" {
		return fmt.Errorf("budget name is required")
	}
	if opts.LimitUSD <= 0 {
		return fmt.Errorf("budget limit must be greater than zero")
	}
	if email == "" || !strings.Contains(email, "@") {
		return fmt.Errorf("a valid email address is required for budget alerts")
	}

	accountID, err := accountIDForBudgets(ctx)
	if err != nil {
		return err
	}

	client, err := awsclient.NewClientForRegion(ctx, budgetsRegion)
	if err != nil {
		return err
	}
	budgetsClient := budgets.NewFromConfig(client.Config())

	if err := createBudgetWithClient(ctx, budgetsClient, accountID, opts); err != nil {
		return err
	}

	clearBudgetCache()
	return nil
}

// DeleteBudget removes a budget by exact name.
func DeleteBudget(ctx context.Context, name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("budget name is required")
	}

	accountID, err := accountIDForBudgets(ctx)
	if err != nil {
		return err
	}

	client, err := awsclient.NewClientForRegion(ctx, budgetsRegion)
	if err != nil {
		return err
	}
	budgetsClient := budgets.NewFromConfig(client.Config())

	if err := deleteBudgetWithClient(ctx, budgetsClient, accountID, name); err != nil {
		return err
	}

	clearBudgetCache()
	return nil
}

// GetBudgetDetails returns the name, monthly limit, and alert email for a budget.
func GetBudgetDetails(ctx context.Context, name string) (BudgetDetails, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return BudgetDetails{}, fmt.Errorf("budget name is required")
	}

	accountID, err := accountIDForBudgets(ctx)
	if err != nil {
		return BudgetDetails{}, err
	}

	client, err := awsclient.NewClientForRegion(ctx, budgetsRegion)
	if err != nil {
		return BudgetDetails{}, err
	}
	budgetsClient := budgets.NewFromConfig(client.Config())

	out, err := budgetsClient.DescribeBudget(ctx, &budgets.DescribeBudgetInput{
		AccountId:  aws.String(accountID),
		BudgetName: aws.String(name),
	})
	if err != nil {
		return BudgetDetails{}, wrapBudgetGetError(name, err)
	}

	limitUSD, err := budgetLimitUSD(out.Budget)
	if err != nil {
		return BudgetDetails{}, fmt.Errorf("get budget %q: %w", name, err)
	}

	email, err := budgetAlertEmail(ctx, budgetsClient, accountID, name)
	if err != nil {
		return BudgetDetails{}, err
	}

	return BudgetDetails{Name: name, LimitUSD: limitUSD, Email: email}, nil
}

// UpdateBudget changes a budget's name, limit, and/or alert email. AWS does not
// allow renaming in place; a name change creates the new budget then deletes the old one.
func UpdateBudget(ctx context.Context, opts UpdateBudgetOptions) error {
	currentName := strings.TrimSpace(opts.CurrentName)
	if currentName == "" {
		return fmt.Errorf("budget name is required")
	}

	details, err := GetBudgetDetails(ctx, currentName)
	if err != nil {
		return err
	}

	newName := strings.TrimSpace(opts.NewName)
	if newName == "" {
		newName = details.Name
	}
	newLimit := details.LimitUSD
	if opts.NewLimitUSD != nil {
		newLimit = *opts.NewLimitUSD
	}
	newEmail := strings.TrimSpace(opts.NewEmail)
	if newEmail == "" {
		newEmail = details.Email
	}

	if newName == details.Name && newLimit == details.LimitUSD && newEmail == details.Email {
		return nil
	}
	if newLimit <= 0 {
		return fmt.Errorf("budget limit must be greater than zero")
	}
	if newEmail == "" || !strings.Contains(newEmail, "@") {
		return fmt.Errorf("a valid email address is required for budget alerts")
	}

	accountID, err := accountIDForBudgets(ctx)
	if err != nil {
		return err
	}

	client, err := awsclient.NewClientForRegion(ctx, budgetsRegion)
	if err != nil {
		return err
	}
	budgetsClient := budgets.NewFromConfig(client.Config())

	if newName != details.Name {
		// if the name is different, create a new budget and delete the old one
		if err := createBudgetWithClient(ctx, budgetsClient, accountID, CreateBudgetOptions{
			Name:     newName,
			LimitUSD: newLimit,
			Email:    newEmail,
		}); err != nil {
			return err
		}
		// delete the old budget
		if err := deleteBudgetWithClient(ctx, budgetsClient, accountID, details.Name); err != nil {
			return fmt.Errorf("renamed budget to %q but failed to delete %q: %w\nhint: delete the old budget manually with `devbox budget delete %q`", newName, details.Name, err, details.Name)
		}
		clearBudgetCache()
		return nil
	}
	// if the limit is different, update the limit
	if newLimit != details.LimitUSD {
		if err := updateBudgetLimit(ctx, budgetsClient, accountID, details.Name, newLimit); err != nil {
			return err
		}
	}
	// if the email is different, update the email
	if newEmail != details.Email {
		if err := updateBudgetEmail(ctx, budgetsClient, accountID, details.Name, details.Email, newEmail); err != nil {
			return err
		}
	}

	clearBudgetCache()
	return nil
}

func createBudgetWithClient(ctx context.Context, budgetsClient *budgets.Client, accountID string, opts CreateBudgetOptions) error {
	name := strings.TrimSpace(opts.Name)
	email := strings.TrimSpace(opts.Email)
	limitAmount := strconv.FormatFloat(opts.LimitUSD, 'f', -1, 64)
	emailSubscriber := budgetstypes.Subscriber{
		Address:          aws.String(email),
		SubscriptionType: budgetstypes.SubscriptionTypeEmail,
	}

	_, err := budgetsClient.CreateBudget(ctx, &budgets.CreateBudgetInput{
		AccountId: aws.String(accountID),
		Budget: &budgetstypes.Budget{
			BudgetName: aws.String(name),
			BudgetType: budgetstypes.BudgetTypeCost,
			TimeUnit:   budgetstypes.TimeUnitMonthly,
			BudgetLimit: &budgetstypes.Spend{
				Amount: aws.String(limitAmount),
				Unit:   aws.String("USD"),
			},
		},
		NotificationsWithSubscribers: defaultBudgetNotifications(emailSubscriber),
	})
	if err != nil {
		return wrapBudgetCreateError(name, err)
	}
	return nil
}

func deleteBudgetWithClient(ctx context.Context, budgetsClient *budgets.Client, accountID, name string) error {
	_, err := budgetsClient.DeleteBudget(ctx, &budgets.DeleteBudgetInput{
		AccountId:  aws.String(accountID),
		BudgetName: aws.String(name),
	})
	if err != nil {
		return wrapBudgetDeleteError(name, err)
	}
	return nil
}

func updateBudgetLimit(ctx context.Context, budgetsClient *budgets.Client, accountID, name string, limitUSD float64) error {
	out, err := budgetsClient.DescribeBudget(ctx, &budgets.DescribeBudgetInput{
		AccountId:  aws.String(accountID),
		BudgetName: aws.String(name),
	})
	if err != nil {
		return wrapBudgetGetError(name, err)
	}

	b := out.Budget
	limitAmount := strconv.FormatFloat(limitUSD, 'f', -1, 64)
	b.BudgetLimit = &budgetstypes.Spend{
		Amount: aws.String(limitAmount),
		Unit:   aws.String("USD"),
	}

	_, err = budgetsClient.UpdateBudget(ctx, &budgets.UpdateBudgetInput{
		AccountId: aws.String(accountID),
		NewBudget: b,
	})
	if err != nil {
		return wrapBudgetUpdateError(name, err)
	}
	return nil
}

func updateBudgetEmail(ctx context.Context, budgetsClient *budgets.Client, accountID, budgetName, oldEmail, newEmail string) error {
	notifications, err := describeBudgetNotifications(ctx, budgetsClient, accountID, budgetName)
	if err != nil {
		return err
	}

	oldSubscriber := budgetstypes.Subscriber{
		Address:          aws.String(oldEmail),
		SubscriptionType: budgetstypes.SubscriptionTypeEmail,
	}
	newSubscriber := budgetstypes.Subscriber{
		Address:          aws.String(newEmail),
		SubscriptionType: budgetstypes.SubscriptionTypeEmail,
	}

	for _, notification := range notifications {
		// for each alert, describe the subscribers
		subscribers, err := describeNotificationSubscribers(ctx, budgetsClient, accountID, budgetName, notification)
		if err != nil {
			return err
		}
		// for each subscriber, update the subscriber
		for _, sub := range subscribers {
			if sub.SubscriptionType != budgetstypes.SubscriptionTypeEmail {
				continue
			}
			if aws.ToString(sub.Address) != oldEmail {
				continue
			}
			_, err := budgetsClient.UpdateSubscriber(ctx, &budgets.UpdateSubscriberInput{
				AccountId:     aws.String(accountID),
				BudgetName:    aws.String(budgetName),
				Notification:  notification,
				OldSubscriber: &oldSubscriber,
				NewSubscriber: &newSubscriber,
			})
			if err != nil {
				return wrapBudgetUpdateError(budgetName, err)
			}
		}
	}
	return nil
}

// budgetLimitUSD converts the budget limit to a float64
func budgetLimitUSD(b *budgetstypes.Budget) (float64, error) {
	if b == nil || b.BudgetLimit == nil {
		return 0, fmt.Errorf("budget has no limit")
	}
	limit, err := strconv.ParseFloat(aws.ToString(b.BudgetLimit.Amount), 64) // parse the limit to a float64
	if err != nil {
		return 0, fmt.Errorf("parse budget limit: %w", err)
	}
	return limit, nil
}

func budgetAlertEmail(ctx context.Context, budgetsClient *budgets.Client, accountID, budgetName string) (string, error) {
	notifications, err := describeBudgetNotifications(ctx, budgetsClient, accountID, budgetName)
	if err != nil {
		return "", err
	}

	for _, notification := range notifications {
		subscribers, err := describeNotificationSubscribers(ctx, budgetsClient, accountID, budgetName, notification)
		if err != nil {
			return "", err
		}
		for _, sub := range subscribers {
			if sub.SubscriptionType == budgetstypes.SubscriptionTypeEmail {
				if addr := strings.TrimSpace(aws.ToString(sub.Address)); addr != "" {
					return addr, nil
				}
			}
		}
	}
	return "", fmt.Errorf("budget %q has no email alert subscriber", budgetName)
}

func describeBudgetNotifications(ctx context.Context, budgetsClient *budgets.Client, accountID, budgetName string) ([]*budgetstypes.Notification, error) {
	var notifications []*budgetstypes.Notification
	paginator := budgets.NewDescribeNotificationsForBudgetPaginator(budgetsClient, &budgets.DescribeNotificationsForBudgetInput{
		AccountId:  aws.String(accountID),
		BudgetName: aws.String(budgetName),
	})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, wrapBudgetGetError(budgetName, err)
		}
		for i := range page.Notifications {
			notifications = append(notifications, &page.Notifications[i])
		}
	}
	return notifications, nil
}

func describeNotificationSubscribers(ctx context.Context, budgetsClient *budgets.Client, accountID, budgetName string, notification *budgetstypes.Notification) ([]budgetstypes.Subscriber, error) {
	var subscribers []budgetstypes.Subscriber
	paginator := budgets.NewDescribeSubscribersForNotificationPaginator(budgetsClient, &budgets.DescribeSubscribersForNotificationInput{
		AccountId:    aws.String(accountID),
		BudgetName:   aws.String(budgetName),
		Notification: notification,
	})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, wrapBudgetGetError(budgetName, err)
		}
		subscribers = append(subscribers, page.Subscribers...)
	}
	return subscribers, nil
}

// defaultBudgetNotifications creates the default budget notifications for the budget
func defaultBudgetNotifications(email budgetstypes.Subscriber) []budgetstypes.NotificationWithSubscribers {
	thresholds := []struct {
		notificationType budgetstypes.NotificationType
		threshold        float64
	}{
		{budgetstypes.NotificationTypeActual, 85},
		{budgetstypes.NotificationTypeActual, 100},
		{budgetstypes.NotificationTypeForecasted, 100},
	}

	notifications := make([]budgetstypes.NotificationWithSubscribers, 0, len(thresholds))
	for _, t := range thresholds {
		notifications = append(notifications, budgetstypes.NotificationWithSubscribers{
			Notification: &budgetstypes.Notification{
				ComparisonOperator: budgetstypes.ComparisonOperatorGreaterThan,
				NotificationType:   t.notificationType,
				Threshold:          t.threshold,
				ThresholdType:      budgetstypes.ThresholdTypePercentage,
			},
			Subscribers: []budgetstypes.Subscriber{email},
		})
	}
	return notifications
}

func wrapBudgetError(operation string, err error) error {
	if awsclient.IsPermissionError(err) {
		return fmt.Errorf("%s: %w\nhint: add the AWSBudgetsActionsWithAWSResourceControlAccess permission policy to the IAM user", operation, err)
	}
	return awsclient.WrapError(operation, err)
}

func wrapBudgetCreateError(name string, err error) error {
	if awsclient.IsPermissionError(err) {
		return fmt.Errorf("create budget: %w\nhint: add AWSBudgetsActionsWithAWSResourceControlAccess permission to the IAM user", err)
	}
	switch {
	case budgetHasErrorCode(err, "DuplicateRecordException"):
		return fmt.Errorf("budget already exists: %s\nhint: choose a different name or delete the existing budget first", name)
	case budgetHasErrorCode(err, "CreationLimitExceededException"):
		return fmt.Errorf("create budget: %w\nhint: AWS budget or notification limit reached — delete unused budgets or request a limit increase in the AWS console", err)
	case budgetHasErrorCode(err, "InvalidParameterException"):
		return fmt.Errorf("create budget: %w\nhint: check the budget name, limit, and email address", err)
	case budgetHasErrorCode(err, "ServiceQuotaExceededException"):
		return fmt.Errorf("create budget: %w\nhint: AWS service quota reached — request a quota increase in the AWS console", err)
	case budgetHasErrorCode(err, "ResourceLockedException"):
		return fmt.Errorf("create budget: %w\nhint: the budget is locked — wait a moment and retry", err)
	default:
		return awsclient.WrapError("create budget", err)
	}
}

func wrapBudgetGetError(name string, err error) error {
	if awsclient.IsPermissionError(err) {
		return fmt.Errorf("get budget: %w\nhint: add AWSBudgetsActionsWithAWSResourceControlAccess permission to the IAM user", err)
	}
	if budgetHasErrorCode(err, "NotFoundException") {
		return fmt.Errorf("budget not found: %s\nhint: run `devbox budget ls` to see existing budgets", name)
	}
	return awsclient.WrapError("get budget", err)
}

func wrapBudgetUpdateError(name string, err error) error {
	if awsclient.IsPermissionError(err) {
		return fmt.Errorf("update budget: %w\nhint: add AWSBudgetsActionsWithAWSResourceControlAccess permission to the IAM user", err)
	}
	switch {
	case budgetHasErrorCode(err, "NotFoundException"):
		return fmt.Errorf("budget not found: %s\nhint: run `devbox budget ls` to see existing budgets", name)
	case budgetHasErrorCode(err, "InvalidParameterException"):
		return fmt.Errorf("update budget: %w\nhint: check the budget name, limit, and email address", err)
	case budgetHasErrorCode(err, "DuplicateRecordException"):
		return fmt.Errorf("budget already exists: %s\nhint: choose a different name", name)
	default:
		return awsclient.WrapError("update budget", err)
	}
}

func wrapBudgetDeleteError(name string, err error) error {
	if awsclient.IsPermissionError(err) {
		return fmt.Errorf("delete budget: %w\nhint: add AWSBudgetsActionsWithAWSResourceControlAccess permission to the IAM user", err)
	}
	if budgetHasErrorCode(err, "NotFoundException") {
		return fmt.Errorf("budget not found: %s\nhint: run `devbox budget ls` to see existing budgets", name)
	}
	return awsclient.WrapError("delete budget", err)
}

func budgetHasErrorCode(err error, code string) bool {
	for err != nil {
		var apiErr smithy.APIError
		if errors.As(err, &apiErr) && apiErr.ErrorCode() == code {
			return true
		}
		err = errors.Unwrap(err)
	}
	return false
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

func clearBudgetCache() {
	path, err := budgetCachePath()
	if err != nil {
		return
	}
	_ = os.Remove(path)
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
