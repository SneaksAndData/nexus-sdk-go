package sdk

import (
	"github.com/SneaksAndData/nexus-core/pkg/checkpoint/models"
	sapi "github.com/SneaksAndData/nexus-sdk-go/pkg/generated/scheduler"
	"github.com/aws/smithy-go/ptr"
	"testing"
)

type ExpectedResultCheck struct {
	IsFinished bool
	IsSuccess  *bool
}

func getFakeResults() map[sapi.ModelsRequestResult]ExpectedResultCheck {
	return map[sapi.ModelsRequestResult]ExpectedResultCheck{
		{
			sapi.NewOptString("test-id"),
			sapi.NewOptString("http://localhost/result"),
			sapi.OptString{Set: false},
			sapi.NewOptString(models.LifecycleStageCompleted),
		}: {
			IsFinished: true,
			IsSuccess:  ptr.Bool(true),
		},
		{
			sapi.NewOptString("test-id"),
			sapi.NewOptString("http://localhost/result"),
			sapi.OptString{Set: false},
			sapi.NewOptString(models.LifecycleStageRunning),
		}: {
			IsFinished: false,
			IsSuccess:  nil,
		},
		{
			sapi.NewOptString("test-id"),
			sapi.NewOptString("http://localhost/result"),
			sapi.OptString{Set: false},
			sapi.NewOptString(models.LifecycleStageCancelled),
		}: {
			IsFinished: true,
			IsSuccess:  ptr.Bool(false),
		},
		{
			sapi.NewOptString("test-id"),
			sapi.NewOptString("http://localhost/result"),
			sapi.OptString{Set: false},
			sapi.NewOptString(models.LifecycleStageNew),
		}: {
			IsFinished: false,
			IsSuccess:  nil,
		},
		{
			sapi.NewOptString("test-id"),
			sapi.NewOptString("http://localhost/result"),
			sapi.OptString{Set: false},
			sapi.NewOptString(models.LifecycleStageFailed),
		}: {
			IsFinished: true,
			IsSuccess:  ptr.Bool(false),
		},
		{
			sapi.NewOptString("test-id"),
			sapi.NewOptString("http://localhost/result"),
			sapi.OptString{Set: false},
			sapi.NewOptString(models.LifecycleStageBuffered),
		}: {
			IsFinished: false,
			IsSuccess:  nil,
		},
		{
			sapi.NewOptString("test-id"),
			sapi.NewOptString("http://localhost/result"),
			sapi.OptString{Set: false},
			sapi.NewOptString(models.LifecycleStageSchedulingFailed),
		}: {
			IsFinished: true,
			IsSuccess:  ptr.Bool(false),
		},
		{
			sapi.NewOptString("test-id"),
			sapi.NewOptString("http://localhost/result"),
			sapi.OptString{Set: false},
			sapi.NewOptString(models.LifecycleStageDeadlineExceeded),
		}: {
			IsFinished: true,
			IsSuccess:  ptr.Bool(false),
		},
	}
}

func TestResultChecks(t *testing.T) {
	for result, expectedCheck := range getFakeResults() {
		finished, _ := IsFinished(result)
		success, _ := IsSuccess(result)

		if expectedCheck.IsFinished != finished {
			t.Fatalf("for status %s expected: %v, got: %v", result.Status.Value, expectedCheck.IsFinished, finished)
		}

		if expectedCheck.IsSuccess != nil && success != nil && *expectedCheck.IsSuccess != *success {
			t.Fatalf("for status %s expected: %v, got: %v", result.Status.Value, *expectedCheck.IsSuccess, *success)
		}

		if expectedCheck.IsSuccess == nil && success != nil {
			t.Fatalf("expected nil for IsSuccess for status %s", result.Status.Value)
		}

		if expectedCheck.IsSuccess != nil && success == nil {
			t.Fatalf("expected a value nil for IsSuccess for status %s, but got nil", result.Status.Value)
		}
	}
}
