package sdk

import (
	"github.com/SneaksAndData/nexus-core/pkg/checkpoint/models"
	"testing"
)

type ExpectedResultCheck struct {
	IsFinished bool
	IsSuccess  int
}

func getFakeResults() map[string]ExpectedResultCheck {
	return map[string]ExpectedResultCheck{
		models.LifecycleStageCompleted: {
			IsFinished: true,
			IsSuccess:  1,
		},
		models.LifecycleStageRunning: {
			IsFinished: false,
			IsSuccess:  -1,
		},
		models.LifecycleStageCancelled: {
			IsFinished: true,
			IsSuccess:  0,
		},
		models.LifecycleStageNew: {
			IsFinished: false,
			IsSuccess:  -1,
		},
		models.LifecycleStageFailed: {
			IsFinished: true,
			IsSuccess:  0,
		},
		models.LifecycleStageBuffered: {
			IsFinished: false,
			IsSuccess:  -1,
		},
		models.LifecycleStageSchedulingFailed: {
			IsFinished: true,
			IsSuccess:  0,
		},
		models.LifecycleStageDeadlineExceeded: {
			IsFinished: true,
			IsSuccess:  0,
		},
	}
}

func TestResultChecks(t *testing.T) {
	for result, expectedCheck := range getFakeResults() {
		finished := IsFinished(result)
		success := IsSuccess(result)

		if expectedCheck.IsFinished != finished {
			t.Fatalf("for status %s expected: %v, got: %v", result, expectedCheck.IsFinished, finished)
		}

		if expectedCheck.IsSuccess != success {
			t.Fatalf("for status %s expected: %v, got: %v", result, expectedCheck.IsSuccess, success)
		}
	}
}
