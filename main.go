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
//char* client_error_type;
//char* client_error_message;
//char* status;
//} RunResult;
//typedef struct AlgorithmRun {
//char* client_error_type;
//char* client_error_message;
//char* request_id;
//} AlgorithmRun;
import "C"
import (
	"github.com/SneaksAndData/nexus-core/pkg/signals"
	"github.com/SneaksAndData/nexus-core/pkg/telemetry"
	api "github.com/SneaksAndData/nexus-sdk-go/pkg/generated/scheduler"
	"github.com/SneaksAndData/nexus-sdk-go/sdk"
	models2 "github.com/SneaksAndData/nexus-sdk-go/sdk/models"
	"github.com/ogen-go/ogen/json"
	"k8s.io/klog/v2"
	"os"
	"reflect"
	"runtime"
	"unsafe"
)

// for those in need: https://fluhus.github.io/snopher/

var pinner = runtime.Pinner{}
var client *sdk.NexusSchedulerClient

//export CreateSchedulerClient
func CreateSchedulerClient(url *C.char, token *C.char) {
	logLevelString := os.Getenv("NEXUS__SDK_LOG_LEVEL")
	if logLevelString == "" {
		logLevelString = telemetry.LoggingDisabled
	}
	ctx := signals.SetupSignalHandler()
	appLogger, err := telemetry.ConfigureLogger(ctx, map[string]string{}, logLevelString)
	klog.SetSlogLogger(appLogger)

	logger := klog.FromContext(ctx)

	if err != nil {
		logger.V(1).Error(err, "one of the logging handlers cannot be configured")
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
		} else if err != nil {
			client.Logger.V(1).Error(err, "error retrieving results")
			errorResults := C.malloc(C.size_t(1) * C.size_t(unsafe.Sizeof(uintptr(0))))
			goCResults := (*[10]*C.RunResult)(errorResults)
			goCResults[0] = &C.RunResult{
				algorithm:            nil,
				request_id:           nil,
				result_uri:           nil,
				run_error_message:    nil,
				client_error_type:    C.CString(reflect.TypeOf(err).String()),
				client_error_message: C.CString(err.Error()),
				status:               nil,
			}
			return (**C.RunResult)(errorResults)
		}
	}

	cResults := C.malloc(C.size_t(len(results)) * C.size_t(unsafe.Sizeof(uintptr(0))))
	goCResults := (*[10000]*C.RunResult)(unsafe.Pointer(cResults))

	for i, result := range results {
		goCResults[i] = &C.RunResult{
			algorithm:            C.CString(result.AlgorithmName.Value),
			request_id:           C.CString(result.RequestId.Value),
			result_uri:           C.CString(result.ResultUri.Value),
			run_error_message:    C.CString(result.RunErrorMessage.Value),
			client_error_type:    nil,
			client_error_message: nil,
			status:               C.CString(result.Status.Value),
		}
	}

	return (**C.RunResult)(cResults)
}

//export CreateRun
func CreateRun(algorithmName *C.char, algorithmParameters *C.char, customConfiguration *C.char, parentRequest *C.char, payloadValidFrom *C.char, tag *C.char) *C.AlgorithmRun {
	var algParams api.ModelsAlgorithmRequestAlgorithmParameters
	var algSpecOverride api.V1NexusAlgorithmSpec
	var parent api.ModelsAlgorithmRequestRef
	var decodeErr *models2.InputDecodeError
	parentRequestParam := api.OptModelsAlgorithmRequestRef{
		Value: api.ModelsAlgorithmRequestRef{},
		Set:   false,
	}
	requestTag := api.OptString{
		Value: "",
		Set:   false,
	}
	customSpec := api.OptV1NexusAlgorithmSpec{
		Value: &api.V1NexusAlgorithmSpec{},
		Set:   false,
	}

	reportDecodeErr := func(err error) C.AlgorithmRun {
		decodeErr = models2.NewInputDecodeError(err)
		return &C.AlgorithmRun{
			request_id:           C.CString(""),
			client_error_type:    C.CString(reflect.TypeOf(decodeErr).String()),
			client_error_message: C.CString(decodeErr.Error()),
		}
	}

	if err := json.Unmarshal([]byte(C.GoString(algorithmParameters)), &algParams); err != nil {
		return reportDecodeErr(err)
	}

	if parentRequest != nil {
		if err := json.Unmarshal([]byte(C.GoString(parentRequest)), &parent); err != nil {
			return reportDecodeErr(err)
		}

		parentRequestParam = api.OptModelsAlgorithmRequestRef{
			Value: parent,
			Set:   true,
		}
	}

	if customConfiguration != nil {
		if err := json.Unmarshal([]byte(C.GoString(algSpecOverride)), &algSpecOverride); err != nil {
			return reportDecodeErr(err)
		}

		customSpec = api.OptV1NexusAlgorithmSpec{
			Value: algSpecOverride,
			Set:   true,
		}
	}

	if tag != nil {
		requestTag = api.OptString{
			Value: C.GoString(tag),
			Set:   true,
		}
	}

	result, err := client.CreateRun(&api.ModelsAlgorithmRequest{
		AlgorithmParameters: algParams,
		CustomConfiguration: customSpec,
		ParentRequest:       parentRequestParam,
		PayloadValidFor: api.OptString{
			Value: C.GoString(payloadValidFrom),
			Set:   true,
		},
		RequestApiVersion: api.OptString{
			Value: "1.2",
			Set:   true,
		},
		Tag: requestTag,
	}, C.GoString(algorithmName))

	if err != nil {
		return &C.AlgorithmRun{
			request_id:           C.CString(""),
			client_error_type:    C.CString(reflect.TypeOf(err).String()),
			client_error_message: C.CString(err.Error()),
		}
	}

	return &C.AlgorithmRun{
		request_id:           C.CString(result),
		client_error_type:    nil,
		client_error_message: nil,
	}
}

//export UpdateToken
func UpdateToken(token *C.char) {
	client.RefreshAuth(C.GoString(token))
}

//export FreeRunResult
func FreeRunResult(result C.RunResult) {
	C.free(unsafe.Pointer(result.algorithm))
	C.free(unsafe.Pointer(result.request_id))
	C.free(unsafe.Pointer(result.result_uri))
	C.free(unsafe.Pointer(result.run_error_message))
	C.free(unsafe.Pointer(result.client_error_type))
	C.free(unsafe.Pointer(result.client_error_message))
	C.free(unsafe.Pointer(result.status))
}

//export FreeAlgorithmRun
func FreeAlgorithmRun(algRun C.AlgorithmRun) {
	C.free(unsafe.Pointer(algRun.client_error_type))
	C.free(unsafe.Pointer(algRun.client_error_message))
	C.free(unsafe.Pointer(algRun.request_id))
}

//export FreeClient
func FreeClient() {
	pinner.Unpin()
}

func main() {

}
