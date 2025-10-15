package sdk

import (
	coremodels "github.com/SneaksAndData/nexus-core/pkg/checkpoint/models"
)

func IsFinished(resultStatus string) bool {
	switch resultStatus {
	case coremodels.LifecycleStageFailed, coremodels.LifecycleStageCompleted, coremodels.LifecycleStageDeadlineExceeded, coremodels.LifecycleStageSchedulingFailed, coremodels.LifecycleStageCancelled:
		return true
	case coremodels.LifecycleStageRunning:
		return false
	default:
		return false
	}
}

func IsSuccess(resultStatus string) int {
	finished := IsFinished(resultStatus)

	if !finished {
		return -1
	}

	if resultStatus == coremodels.LifecycleStageCompleted {
		return 1
	}

	return 0
}
