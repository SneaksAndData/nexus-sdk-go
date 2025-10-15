package sdk

import (
	"fmt"
	coremodels "github.com/SneaksAndData/nexus-core/pkg/checkpoint/models"
	sapi "github.com/SneaksAndData/nexus-sdk-go/pkg/generated/scheduler"
	"github.com/SneaksAndData/nexus-sdk-go/sdk/models"
	"github.com/aws/smithy-go/ptr"
)

func IsFinished(runResult sapi.ModelsRequestResult) (bool, error) {
	if !runResult.Status.Set {
		return false, models.NewSdkErr(fmt.Errorf("status not set for the run %s", runResult.RequestId.Value))
	}

	switch runResult.Status.Value {
	case coremodels.LifecycleStageFailed, coremodels.LifecycleStageCompleted, coremodels.LifecycleStageDeadlineExceeded, coremodels.LifecycleStageSchedulingFailed, coremodels.LifecycleStageCancelled:
		return true, nil
	case coremodels.LifecycleStageRunning:
		return false, nil
	default:
		return false, nil
	}
}

func IsSuccess(runResult sapi.ModelsRequestResult) (*bool, error) {
	if !runResult.Status.Set {
		return ptr.Bool(false), models.NewSdkErr(fmt.Errorf("status not set for the run %s", runResult.RequestId.Value))
	}

	finished, err := IsFinished(runResult)
	if err != nil {
		return ptr.Bool(false), err
	}

	if !finished {
		return nil, nil
	}

	result := runResult.Status.Value == coremodels.LifecycleStageCompleted

	return ptr.Bool(result), nil
}
