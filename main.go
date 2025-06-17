package main

//#cgo LDFLAGS:
//#include <stdio.h>
//#include <stdlib.h>
//#include <string.h>
//typedef struct RunResult {
//char* algorithm;
//char* request_id;
//char* result_uri;
//char* run_error_message;
//char* status;
//} RunResult;
import "C"
import (
	"github.com/SneaksAndData/nexus-core/pkg/signals"
	"github.com/SneaksAndData/nexus-core/pkg/telemetry"
	api "github.com/SneaksAndData/nexus-sdk-go/pkg/generated/scheduler"
	"github.com/SneaksAndData/nexus-sdk-go/sdk"
	"k8s.io/klog/v2"
	"runtime"
	"unsafe"
)

// for those in need: https://fluhus.github.io/snopher/

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
func GetRunResults(tag *C.char) **C.RunResult {
	results := []*api.ModelsTaggedRequestResult{}
	for result, err := range client.GetRunResults(C.GoString(tag), nil) {
		if result != nil {
			results = append(results, result)
		} else {
			client.Logger.Error(err, "error retrieving results")
		}
	}

	clangResults := C.malloc(C.size_t(len(results)) * C.size_t(unsafe.Sizeof(uintptr(0))))
	// this is just a type cast, assuming we are way below 10000 results anyway
	resultsPtrArray := (*[10000]*C.RunResult)(clangResults)

	for i, result := range results {
		resultsPtrArray[i] = &C.RunResult{
			algorithm:         C.CString(result.AlgorithmName.Value),
			request_id:        C.CString(result.RequestId.Value),
			result_uri:        C.CString(result.ResultUri.Value),
			run_error_message: C.CString(result.RunErrorMessage.Value),
			status:            C.CString(result.Status.Value),
		}
	}

	return (**C.RunResult)(clangResults)
}

//export UpdateToken
func UpdateToken(token *C.char) {
	client.RefreshAuth(C.GoString(token))
}

// TODO: memory release

func main() {

}
