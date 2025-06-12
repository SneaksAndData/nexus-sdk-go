package main

import "C"
import (
	"github.com/SneaksAndData/nexus-core/pkg/signals"
	"github.com/SneaksAndData/nexus-core/pkg/telemetry"
	api "github.com/SneaksAndData/nexus-sdk-go/pkg/generated/scheduler"
	"github.com/SneaksAndData/nexus-sdk-go/sdk"
	"k8s.io/klog/v2"
	"runtime"
	"strings"
)

var pinner = runtime.Pinner{}
var client *sdk.NexusSchedulerClient

//export CreateSchedulerClient
func CreateSchedulerClient(url *C.char, token *C.char) {
	ctx := signals.SetupSignalHandler()
	appLogger, err := telemetry.ConfigureLogger(ctx, map[string]string{}, "info")
	klog.SetSlogLogger(appLogger)

	logger := klog.FromContext(ctx)

	if err != nil {
		logger.Error(err, "one of the logging handlers cannot be configured")
	}

	if C.GoString(token) == "" {
		client = sdk.NewNexusSchedulerClient(C.GoString(url), &logger, nil, &pinner)
	}

	client = sdk.NewNexusSchedulerClient(C.GoString(url), &logger, &[]api.RequestOption{sdk.GetAuthOption(C.GoString(token))}, &pinner)
}

//export GetRunResults
func GetRunResults(tag *C.char) *C.char {
	results := []string{}
	for result, _ := range client.GetRunResults(C.GoString(tag), nil) {
		results = append(results, result.RequestId.Value)
	}

	return C.CString(strings.Join(results, ","))
}

func main() {

}
