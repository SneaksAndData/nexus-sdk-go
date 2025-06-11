package nexus_sdk_go

import "C"
import (
	"github.com/SneaksAndData/nexus-core/pkg/signals"
	"github.com/SneaksAndData/nexus-core/pkg/telemetry"
	api "github.com/SneaksAndData/nexus-sdk-go/pkg/generated/scheduler"
	"github.com/SneaksAndData/nexus-sdk-go/sdk"
	"k8s.io/klog/v2"
)

//export CreateSchedulerClient
func CreateSchedulerClient(url string, token string) *sdk.NexusSchedulerClient {
	ctx := signals.SetupSignalHandler()
	appLogger, err := telemetry.ConfigureLogger(ctx, map[string]string{}, "info")
	klog.SetSlogLogger(appLogger)

	logger := klog.FromContext(ctx)

	if err != nil {
		logger.Error(err, "one of the logging handlers cannot be configured")
	}

	if token == "" {
		return sdk.NewNexusSchedulerClient(url, &logger, nil)
	}

	return sdk.NewNexusSchedulerClient(url, &logger, &[]api.RequestOption{sdk.GetAuthOption(token)})
}
