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
//char* request_id;
//char* client_error_type;
//char* client_error_message;
//} AlgorithmRun;
//typedef struct CustomRunConfiguration {
//char* version;
//char* workgroup_name;
//char* workgroup_group;
//char* workgroup_kind;
//char* cpu_limit;
//char* memory_limit;
//} CustomRunConfiguration;
//typedef struct ParentRequest {
//char* algorithm_name;
//char* request_id;
//} ParentRequest;
//typedef struct RequestMetadata {
//char* algorithm;
//char* id;
//char* algorithm_failure_cause;
//char* algorithm_failure_details;
//char* api_version;
//char* applied_configuration;
//char* configuration_overrides;
//char* content_hash;
//char* job_uid;
//char* last_modified;
//char* lifecycle_stage;
//char* parent_job;
//char* payload_uri;
//char* payload_valid_for;
//char* received_at;
//char* received_by_host;
//char* result_uri;
//char* sent_at;
//char* tag;
//char* client_error_type;
//char* client_error_message;
//} RequestMetadata;
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
	"time"
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
func GetRunResults(tag *C.char, algorithm *C.char) *C.RunResult {
	results := []*api.ModelsTaggedRequestResult{}
	var algName *string
	if C.GoString(algorithm) != "" {
		value := C.GoString(algorithm)
		algName = &value
	}

	for result, err := range client.GetRunResults(C.GoString(tag), algName) {
		if result != nil {
			results = append(results, result)
		} else if err != nil {
			client.Logger.V(1).Error(err, "error retrieving results")
			errorResults := C.malloc(C.size_t(1) * C.size_t(unsafe.Sizeof(C.RunResult{})))
			goCResults := (*[10]C.RunResult)(errorResults)
			goCResults[0] = C.RunResult{
				algorithm:            nil,
				request_id:           nil,
				result_uri:           nil,
				run_error_message:    nil,
				client_error_type:    C.CString(reflect.TypeOf(err).String()),
				client_error_message: C.CString(err.Error()),
				status:               nil,
			}
			return (*C.RunResult)(errorResults)
		}
	}

	cResults := C.calloc(C.size_t(len(results)+1), C.size_t(unsafe.Sizeof(C.RunResult{})))
	goCResults := (*[10000]C.RunResult)(unsafe.Pointer(cResults))

	for i, result := range results {
		goCResults[i] = C.RunResult{
			algorithm:            C.CString(result.AlgorithmName.Value),
			request_id:           C.CString(result.RequestId.Value),
			result_uri:           C.CString(result.ResultUri.Value),
			run_error_message:    C.CString(result.RunErrorMessage.Value),
			client_error_type:    nil,
			client_error_message: nil,
			status:               C.CString(result.Status.Value),
		}
	}

	return (*C.RunResult)(cResults)
}

//export CreateRun
func CreateRun(algorithmName *C.char, algorithmParameters *C.char, customConfiguration *C.CustomRunConfiguration, parentRequest *C.ParentRequest, payloadValidFor *C.char, tag *C.char) C.AlgorithmRun {
	var algParams api.ModelsAlgorithmRequestAlgorithmParameters
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
		Value: api.V1NexusAlgorithmSpec{},
		Set:   false,
	}

	reportDecodeErr := func(err error) C.AlgorithmRun {
		decodeErr = models2.NewInputDecodeError(err)
		return C.AlgorithmRun{
			request_id:           C.CString(""),
			client_error_type:    C.CString(reflect.TypeOf(decodeErr).String()),
			client_error_message: C.CString(decodeErr.Error()),
		}
	}

	if err := json.Unmarshal([]byte(C.GoString(algorithmParameters)), &algParams); err != nil {
		return reportDecodeErr(err)
	}

	if parentRequest != nil {
		parentRequestParam = api.OptModelsAlgorithmRequestRef{
			Value: api.ModelsAlgorithmRequestRef{
				AlgorithmName: C.GoString(parentRequest.algorithm_name),
				RequestId:     C.GoString(parentRequest.request_id),
			},
			Set: true,
		}
	}

	if customConfiguration != nil {
		var cpuLimit api.OptString
		var memoryLimit api.OptString
		var container api.OptV1NexusAlgorithmContainer
		var workgroup api.OptV1NexusAlgorithmWorkgroupRef

		if customConfiguration.cpu_limit != nil {
			cpuLimit = api.OptString{
				Value: C.GoString(customConfiguration.cpu_limit),
				Set:   true,
			}
		}

		if customConfiguration.memory_limit != nil {
			memoryLimit = api.OptString{
				Value: C.GoString(customConfiguration.memory_limit),
				Set:   true,
			}
		}

		if customConfiguration.version != nil {
			container = api.OptV1NexusAlgorithmContainer{
				Value: api.V1NexusAlgorithmContainer{
					VersionTag: api.OptString{
						Value: C.GoString(customConfiguration.version),
						Set:   true,
					},
				},
				Set: true,
			}
		}

		if customConfiguration.workgroup_name != nil {
			workgroup = api.OptV1NexusAlgorithmWorkgroupRef{
				Value: api.V1NexusAlgorithmWorkgroupRef{
					Name: api.OptString{
						Value: C.GoString(customConfiguration.workgroup_name),
						Set:   true,
					},
					Group: api.OptString{
						Value: C.GoString(customConfiguration.workgroup_group),
						Set:   true,
					},
					Kind: api.OptString{
						Value: C.GoString(customConfiguration.workgroup_kind),
						Set:   true,
					},
				},
				Set: true,
			}
		}

		customSpec = api.OptV1NexusAlgorithmSpec{
			Value: api.V1NexusAlgorithmSpec{
				Args: nil,
				Command: api.OptString{
					Set: false,
				},
				ComputeResources: api.OptV1NexusAlgorithmResources{
					Value: api.V1NexusAlgorithmResources{
						CpuLimit:    cpuLimit,
						MemoryLimit: memoryLimit,
						CustomResources: api.OptV1NexusAlgorithmResourcesCustomResources{
							Set: false,
						},
					},
				},
				Container: container,
				DatadogIntegrationSettings: api.OptV1NexusDatadogIntegrationSettings{
					Set: false,
				},
				ErrorHandlingBehaviour: api.OptV1NexusErrorHandlingBehaviour{
					Set: false,
				},
				RuntimeEnvironment: api.OptV1NexusAlgorithmRuntimeEnvironment{
					Set: false,
				},
				WorkgroupRef: workgroup,
			},
			Set: true,
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
			Value: C.GoString(payloadValidFor),
			Set:   true,
		},
		RequestApiVersion: api.OptString{
			Value: "1.2",
			Set:   true,
		},
		Tag: requestTag,
	}, C.GoString(algorithmName))

	if err != nil {
		return C.AlgorithmRun{
			request_id:           C.CString(""),
			client_error_type:    C.CString(reflect.TypeOf(err).String()),
			client_error_message: C.CString(err.Error()),
		}
	}

	return C.AlgorithmRun{
		request_id:           C.CString(result),
		client_error_type:    nil,
		client_error_message: nil,
	}
}

//export GetRun
func GetRun(requestId *C.char, algorithmName *C.char) C.RunResult {
	result, err := client.GetRun(C.GoString(requestId), C.GoString(algorithmName))

	if err != nil {
		return C.RunResult{
			algorithm:            C.CString(C.GoString(algorithmName)),
			request_id:           C.CString(C.GoString(requestId)),
			result_uri:           nil,
			run_error_message:    nil,
			client_error_type:    C.CString(reflect.TypeOf(err).String()),
			client_error_message: C.CString(err.Error()),
			status:               nil,
		}
	}

	return C.RunResult{
		algorithm:            C.CString(C.GoString(algorithmName)),
		request_id:           C.CString(C.GoString(requestId)),
		result_uri:           C.CString(result.ResultUri.Value),
		run_error_message:    C.CString(result.RunErrorMessage.Value),
		client_error_type:    nil,
		client_error_message: nil,
		status:               C.CString(result.Status.Value),
	}
}

//export GetRequestMetadata
func GetRequestMetadata(requestId *C.char, algorithmName *C.char) C.RequestMetadata {
	metadata, err := client.GetMetadata(C.GoString(requestId), C.GoString(algorithmName))

	if err != nil {
		return C.RequestMetadata{
			algorithm:                 C.CString(C.GoString(algorithmName)),
			id:                        C.CString(C.GoString(requestId)),
			algorithm_failure_cause:   nil,
			algorithm_failure_details: nil,
			api_version:               nil,
			applied_configuration:     nil,
			configuration_overrides:   nil,
			content_hash:              nil,
			job_uid:                   nil,
			last_modified:             nil,
			lifecycle_stage:           nil,
			parent_job:                nil,
			payload_uri:               nil,
			payload_valid_for:         nil,
			received_at:               nil,
			received_by_host:          nil,
			result_uri:                nil,
			sent_at:                   nil,
			tag:                       nil,
			client_error_type:         C.CString(reflect.TypeOf(err).String()),
			client_error_message:      C.CString(err.Error()),
		}
	}

	var appliedConfig string
	var configOverride string

	if metadata.AppliedConfiguration.Set {
		configBytes, _ := json.Marshal(metadata.AppliedConfiguration.Value)
		appliedConfig = string(configBytes)
	}

	if metadata.ConfigurationOverrides.Set {
		configBytes, _ := json.Marshal(metadata.ConfigurationOverrides.Value)
		configOverride = string(configBytes)
	}

	return C.RequestMetadata{
		algorithm:                 C.CString(C.GoString(algorithmName)),
		id:                        C.CString(C.GoString(requestId)),
		algorithm_failure_cause:   C.CString(metadata.AlgorithmFailureCause.Value),
		algorithm_failure_details: C.CString(metadata.AlgorithmFailureDetails.Value),
		api_version:               C.CString(metadata.APIVersion.Value),
		applied_configuration:     C.CString(appliedConfig),
		configuration_overrides:   C.CString(configOverride),
		content_hash:              C.CString(metadata.ContentHash.Value),
		job_uid:                   C.CString(metadata.JobUID.Value),
		last_modified:             C.CString(metadata.LastModified.Value),
		lifecycle_stage:           C.CString(metadata.LifecycleStage.Value),
		parent_job:                nil,
		payload_uri:               C.CString(metadata.PayloadURI.Value),
		payload_valid_for:         C.CString(metadata.PayloadValidFor.Value),
		received_at:               C.CString(metadata.ReceivedAt.Value),
		received_by_host:          C.CString(metadata.ReceivedByHost.Value),
		result_uri:                C.CString(metadata.ResultURI.Value),
		sent_at:                   C.CString(metadata.SentAt.Value),
		tag:                       C.CString(metadata.Tag.Value),
		client_error_type:         nil,
		client_error_message:      nil,
	}
}

//export AwaitRun
func AwaitRun(requestId *C.char, algorithmName *C.char, pollIntervalSeconds int32) C.RunResult {
	pollInterval := time.Duration(pollIntervalSeconds) * time.Second

	result, err := client.AwaitRun(C.GoString(requestId), C.GoString(algorithmName), &pollInterval)

	if err != nil {
		return C.RunResult{
			algorithm:            C.CString(C.GoString(algorithmName)),
			request_id:           C.CString(C.GoString(requestId)),
			result_uri:           nil,
			run_error_message:    nil,
			client_error_type:    C.CString(reflect.TypeOf(err).String()),
			client_error_message: C.CString(err.Error()),
			status:               nil,
		}
	}

	return C.RunResult{
		algorithm:            C.CString(C.GoString(algorithmName)),
		request_id:           C.CString(C.GoString(requestId)),
		result_uri:           C.CString(result.ResultUri.Value),
		run_error_message:    C.CString(result.RunErrorMessage.Value),
		client_error_type:    nil,
		client_error_message: nil,
		status:               C.CString(result.Status.Value),
	}
}

//export AwaitRuns
func AwaitRuns(tags **C.char, algorithm *C.char, pollIntervalSeconds int32, completed *int32) *C.RunResult {
	pollInterval := time.Duration(pollIntervalSeconds) * time.Second
	var algName *string
	if C.GoString(algorithm) != "" {
		value := C.GoString(algorithm)
		algName = &value
	}
	var goTags []string
	cTags := unsafe.Slice(tags, 1<<30)
	// copy tags into a Go slice
	for i := 0; cTags[i] != nil; i++ {
		goTags = append(goTags, C.GoString(cTags[i]))
	}

	var counterRef *chan int32
	// activate progress counter
	if unsafe.Pointer(completed) != nil {
		counter := make(chan int32, 10)
		counterRef = &counter
		go func() {
			for range counter {
				*(*C.int)(unsafe.Pointer(completed))++
			}
		}()
	}

	resultsIter, err := client.AwaitTaggedRuns(goTags, algName, &pollInterval, counterRef)

	if counterRef != nil {
		close(*counterRef)
	}

	results := []*api.ModelsTaggedRequestResult{}

	if err != nil {
		client.Logger.V(1).Error(err, "error retrieving results")
		errorResults := C.malloc(C.size_t(1) * C.size_t(unsafe.Sizeof(C.RunResult{})))
		goCResults := (*[10]C.RunResult)(errorResults)
		goCResults[0] = C.RunResult{
			algorithm:            nil,
			request_id:           nil,
			result_uri:           nil,
			run_error_message:    nil,
			client_error_type:    C.CString(reflect.TypeOf(err).String()),
			client_error_message: C.CString(err.Error()),
			status:               nil,
		}
		return (*C.RunResult)(errorResults)
	}

	for result := range resultsIter {
		results = append(results, result)
	}

	cResults := C.calloc(C.size_t(len(results)+1), C.size_t(unsafe.Sizeof(C.RunResult{})))
	goCResults := (*[10000]C.RunResult)(unsafe.Pointer(cResults))

	for i, result := range results {
		goCResults[i] = C.RunResult{
			algorithm:            C.CString(result.AlgorithmName.Value),
			request_id:           C.CString(result.RequestId.Value),
			result_uri:           C.CString(result.ResultUri.Value),
			run_error_message:    C.CString(result.RunErrorMessage.Value),
			client_error_type:    nil,
			client_error_message: nil,
			status:               C.CString(result.Status.Value),
		}
	}

	return (*C.RunResult)(cResults)
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

//export FreeRequestMetadata
func FreeRequestMetadata(result C.RequestMetadata) {
	C.free(unsafe.Pointer(result.algorithm))
	C.free(unsafe.Pointer(result.id))
	C.free(unsafe.Pointer(result.algorithm_failure_cause))
	C.free(unsafe.Pointer(result.algorithm_failure_details))
	C.free(unsafe.Pointer(result.api_version))
	C.free(unsafe.Pointer(result.applied_configuration))
	C.free(unsafe.Pointer(result.configuration_overrides))
	C.free(unsafe.Pointer(result.content_hash))
	C.free(unsafe.Pointer(result.job_uid))
	C.free(unsafe.Pointer(result.last_modified))
	C.free(unsafe.Pointer(result.lifecycle_stage))
	C.free(unsafe.Pointer(result.parent_job))
	C.free(unsafe.Pointer(result.payload_uri))
	C.free(unsafe.Pointer(result.payload_valid_for))
	C.free(unsafe.Pointer(result.received_at))
	C.free(unsafe.Pointer(result.received_by_host))
	C.free(unsafe.Pointer(result.result_uri))
	C.free(unsafe.Pointer(result.sent_at))
	C.free(unsafe.Pointer(result.tag))
	C.free(unsafe.Pointer(result.client_error_type))
	C.free(unsafe.Pointer(result.client_error_message))
}

//export FreeAlgorithmRun
func FreeAlgorithmRun(algRun C.AlgorithmRun) {
	C.free(unsafe.Pointer(algRun.client_error_type))
	C.free(unsafe.Pointer(algRun.client_error_message))
	C.free(unsafe.Pointer(algRun.request_id))
}

//export FreeRunResultsPointer
func FreeRunResultsPointer(results *C.RunResult) {
	C.free(unsafe.Pointer(results))
}

//export FreeClient
func FreeClient() {
	pinner.Unpin()
}

func main() {

}
