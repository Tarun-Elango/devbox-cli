package service

import (
	"fmt"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	budgetstypes "github.com/aws/aws-sdk-go-v2/service/budgets/types"
)

func TestWrapBudgetAPIError(t *testing.T) {
	t.Run("delete not found", func(t *testing.T) {
		err := wrapBudgetAPIError("delete", "test2", &budgetstypes.NotFoundException{
			Message: aws.String("Unable to get budget: test2 - the budget doesn't exist."),
		})
		if !strings.Contains(err.Error(), "budget not found: test2") {
			t.Fatalf("got %q, want budget not found message", err)
		}
		if !strings.Contains(err.Error(), "outpost budget ls") {
			t.Fatalf("got %q, want list hint", err)
		}
	})

	t.Run("delete permission", func(t *testing.T) {
		err := wrapBudgetAPIError("delete", "test2", &budgetstypes.AccessDeniedException{
			Message: aws.String("not authorized"),
		})
		if !strings.Contains(err.Error(), "AWSBudgetsActionsWithAWSResourceControlAccess") {
			t.Fatalf("got %q, want permission hint", err)
		}
	})

	t.Run("delete generic", func(t *testing.T) {
		err := wrapBudgetAPIError("delete", "test2", fmt.Errorf("connection reset"))
		if !strings.Contains(err.Error(), "delete budget:") {
			t.Fatalf("got %q, want wrapped operation", err)
		}
	})

	t.Run("create duplicate", func(t *testing.T) {
		err := wrapBudgetAPIError("create", "monthly", &budgetstypes.DuplicateRecordException{
			Message: aws.String("The budget name already exists."),
		})
		if !strings.Contains(err.Error(), "budget already exists: monthly") {
			t.Fatalf("got %q, want duplicate message", err)
		}
	})

	t.Run("create creation limit", func(t *testing.T) {
		err := wrapBudgetAPIError("create", "monthly", &budgetstypes.CreationLimitExceededException{
			Message: aws.String("limit exceeded"),
		})
		if !strings.Contains(err.Error(), "limit reached") {
			t.Fatalf("got %q, want limit hint", err)
		}
	})

	t.Run("create invalid parameter", func(t *testing.T) {
		err := wrapBudgetAPIError("create", "monthly", &budgetstypes.InvalidParameterException{
			Message: aws.String("invalid email"),
		})
		if !strings.Contains(err.Error(), "check the budget name") {
			t.Fatalf("got %q, want validation hint", err)
		}
	})

	t.Run("create permission", func(t *testing.T) {
		err := wrapBudgetAPIError("create", "monthly", &budgetstypes.AccessDeniedException{
			Message: aws.String("not authorized"),
		})
		if !strings.Contains(err.Error(), "AWSBudgetsActionsWithAWSResourceControlAccess") {
			t.Fatalf("got %q, want permission hint", err)
		}
	})

	t.Run("get not found", func(t *testing.T) {
		err := wrapBudgetAPIError("get", "monthly", &budgetstypes.NotFoundException{
			Message: aws.String("Unable to get budget: monthly - the budget doesn't exist."),
		})
		if !strings.Contains(err.Error(), "budget not found: monthly") {
			t.Fatalf("got %q, want budget not found message", err)
		}
	})

	t.Run("get permission", func(t *testing.T) {
		err := wrapBudgetAPIError("get", "monthly", &budgetstypes.AccessDeniedException{
			Message: aws.String("not authorized"),
		})
		if !strings.Contains(err.Error(), "AWSBudgetsActionsWithAWSResourceControlAccess") {
			t.Fatalf("got %q, want permission hint", err)
		}
	})

	t.Run("update not found", func(t *testing.T) {
		err := wrapBudgetAPIError("update", "monthly", &budgetstypes.NotFoundException{
			Message: aws.String("Unable to get budget: monthly - the budget doesn't exist."),
		})
		if !strings.Contains(err.Error(), "budget not found: monthly") {
			t.Fatalf("got %q, want not found message", err)
		}
	})

	t.Run("update invalid parameter", func(t *testing.T) {
		err := wrapBudgetAPIError("update", "monthly", &budgetstypes.InvalidParameterException{
			Message: aws.String("invalid email"),
		})
		if !strings.Contains(err.Error(), "check the budget name") {
			t.Fatalf("got %q, want validation hint", err)
		}
	})

	t.Run("update duplicate", func(t *testing.T) {
		err := wrapBudgetAPIError("update", "monthly", &budgetstypes.DuplicateRecordException{
			Message: aws.String("The budget name already exists."),
		})
		if !strings.Contains(err.Error(), "budget already exists: monthly") {
			t.Fatalf("got %q, want duplicate message", err)
		}
	})

	t.Run("update permission", func(t *testing.T) {
		err := wrapBudgetAPIError("update", "monthly", &budgetstypes.AccessDeniedException{
			Message: aws.String("not authorized"),
		})
		if !strings.Contains(err.Error(), "AWSBudgetsActionsWithAWSResourceControlAccess") {
			t.Fatalf("got %q, want permission hint", err)
		}
	})

	t.Run("list permission", func(t *testing.T) {
		err := wrapBudgetAPIError("list", "", &budgetstypes.AccessDeniedException{
			Message: aws.String("not authorized"),
		})
		if !strings.Contains(err.Error(), "list budgets:") {
			t.Fatalf("got %q, want list budgets operation", err)
		}
		if !strings.Contains(err.Error(), "AWSBudgetsActionsWithAWSResourceControlAccess") {
			t.Fatalf("got %q, want permission hint", err)
		}
	})
}
