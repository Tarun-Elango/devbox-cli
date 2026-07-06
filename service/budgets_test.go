package service

import (
	"fmt"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	budgetstypes "github.com/aws/aws-sdk-go-v2/service/budgets/types"
)

func TestWrapBudgetDeleteError(t *testing.T) {
	t.Run("not found", func(t *testing.T) {
		err := wrapBudgetDeleteError("test2", &budgetstypes.NotFoundException{
			Message: aws.String("Unable to get budget: test2 - the budget doesn't exist."),
		})
		if !strings.Contains(err.Error(), "budget not found: test2") {
			t.Fatalf("got %q, want budget not found message", err)
		}
		if !strings.Contains(err.Error(), "devbox budget ls") {
			t.Fatalf("got %q, want list hint", err)
		}
	})

	t.Run("permission", func(t *testing.T) {
		err := wrapBudgetDeleteError("test2", &budgetstypes.AccessDeniedException{
			Message: aws.String("not authorized"),
		})
		if !strings.Contains(err.Error(), "AWSBudgetsActionsWithAWSResourceControlAccess") {
			t.Fatalf("got %q, want permission hint", err)
		}
	})

	t.Run("generic", func(t *testing.T) {
		err := wrapBudgetDeleteError("test2", fmt.Errorf("connection reset"))
		if !strings.Contains(err.Error(), "delete budget:") {
			t.Fatalf("got %q, want wrapped operation", err)
		}
	})
}

func TestWrapBudgetCreateError(t *testing.T) {
	t.Run("duplicate", func(t *testing.T) {
		err := wrapBudgetCreateError("monthly", &budgetstypes.DuplicateRecordException{
			Message: aws.String("The budget name already exists."),
		})
		if !strings.Contains(err.Error(), "budget already exists: monthly") {
			t.Fatalf("got %q, want duplicate message", err)
		}
	})

	t.Run("creation limit", func(t *testing.T) {
		err := wrapBudgetCreateError("monthly", &budgetstypes.CreationLimitExceededException{
			Message: aws.String("limit exceeded"),
		})
		if !strings.Contains(err.Error(), "limit reached") {
			t.Fatalf("got %q, want limit hint", err)
		}
	})

	t.Run("invalid parameter", func(t *testing.T) {
		err := wrapBudgetCreateError("monthly", &budgetstypes.InvalidParameterException{
			Message: aws.String("invalid email"),
		})
		if !strings.Contains(err.Error(), "check the budget name") {
			t.Fatalf("got %q, want validation hint", err)
		}
	})

	t.Run("permission", func(t *testing.T) {
		err := wrapBudgetCreateError("monthly", &budgetstypes.AccessDeniedException{
			Message: aws.String("not authorized"),
		})
		if !strings.Contains(err.Error(), "AWSBudgetsActionsWithAWSResourceControlAccess") {
			t.Fatalf("got %q, want permission hint", err)
		}
	})
}

func TestWrapBudgetGetError(t *testing.T) {
	t.Run("not found", func(t *testing.T) {
		err := wrapBudgetGetError("monthly", &budgetstypes.NotFoundException{
			Message: aws.String("Unable to get budget: monthly - the budget doesn't exist."),
		})
		if !strings.Contains(err.Error(), "budget not found: monthly") {
			t.Fatalf("got %q, want budget not found message", err)
		}
	})

	t.Run("permission", func(t *testing.T) {
		err := wrapBudgetGetError("monthly", &budgetstypes.AccessDeniedException{
			Message: aws.String("not authorized"),
		})
		if !strings.Contains(err.Error(), "AWSBudgetsActionsWithAWSResourceControlAccess") {
			t.Fatalf("got %q, want permission hint", err)
		}
	})
}

func TestWrapBudgetUpdateError(t *testing.T) {
	t.Run("not found", func(t *testing.T) {
		err := wrapBudgetUpdateError("monthly", &budgetstypes.NotFoundException{
			Message: aws.String("Unable to get budget: monthly - the budget doesn't exist."),
		})
		if !strings.Contains(err.Error(), "budget not found: monthly") {
			t.Fatalf("got %q, want not found message", err)
		}
	})

	t.Run("invalid parameter", func(t *testing.T) {
		err := wrapBudgetUpdateError("monthly", &budgetstypes.InvalidParameterException{
			Message: aws.String("invalid email"),
		})
		if !strings.Contains(err.Error(), "check the budget name") {
			t.Fatalf("got %q, want validation hint", err)
		}
	})

	t.Run("duplicate", func(t *testing.T) {
		err := wrapBudgetUpdateError("monthly", &budgetstypes.DuplicateRecordException{
			Message: aws.String("The budget name already exists."),
		})
		if !strings.Contains(err.Error(), "budget already exists: monthly") {
			t.Fatalf("got %q, want duplicate message", err)
		}
	})

	t.Run("permission", func(t *testing.T) {
		err := wrapBudgetUpdateError("monthly", &budgetstypes.AccessDeniedException{
			Message: aws.String("not authorized"),
		})
		if !strings.Contains(err.Error(), "AWSBudgetsActionsWithAWSResourceControlAccess") {
			t.Fatalf("got %q, want permission hint", err)
		}
	})
}
