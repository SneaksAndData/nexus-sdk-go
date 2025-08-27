package sdk

import (
	"context"
	"errors"
	"fmt"
	"github.com/SneaksAndData/nexus-core/pkg/checkpoint/models"
	"github.com/SneaksAndData/nexus-sdk-go/pkg/generated/scheduler"
	models2 "github.com/SneaksAndData/nexus-sdk-go/sdk/models"
	"io"
	"iter"
	"k8s.io/klog/v2"
	"runtime"
	"strconv"
	"sync"
	"time"
)

type AwaitTaggedResult struct {
	Result *api.ModelsTaggedRequestResult
	Error  error
}

type AwaitResult struct {
	Result *api.ModelsRequestResult
	Error  error
}

type NexusSchedulerClient struct {
	ApiClient      *api.Client
	RequestOptions *[]api.RequestOption
	Logger         *klog.Logger
}

func NewNexusSchedulerClient(schedulerUrl string, logger *klog.Logger, options *[]api.RequestOption, pinner *runtime.Pinner) *NexusSchedulerClient {
	client, err := api.NewClient(schedulerUrl)

	if err != nil { // coverage-ignore
		logger.Error(err, "unable to initialize Nexus scheduler client")
		klog.FlushAndExit(klog.ExitFlushTimeout, 1)
	}

	result := &NexusSchedulerClient{
		ApiClient:      client,
		RequestOptions: options,
		Logger:         logger,
	}

	if pinner != nil { // coverage-ignore
		pinner.Pin(client)
		pinner.Pin(logger)
		pinner.Pin(options)
		pinner.Pin(result)
	}

	return result
}

func (nc *NexusSchedulerClient) RefreshAuth(token string) { // coverage-ignore
	nc.RequestOptions = &[]api.RequestOption{GetSchedulerAuthOption(token)}
}

func (nc *NexusSchedulerClient) getRequestOptions() []api.RequestOption {
	if nc.RequestOptions == nil {
		return []api.RequestOption{}
	}

	return *nc.RequestOptions // coverage-ignore
}

func getRequestStub(result *api.ModelsRequestResult) *models.CheckpointedRequest {
	return &models.CheckpointedRequest{
		Id:             result.RequestId.Value,
		LifecycleStage: result.Status.Value,
	}
}

func (nc *NexusSchedulerClient) awaitRun(requestId string, algorithmName string, pollInterval *time.Duration) (*api.ModelsRequestResult, error) {
	invalidRequestResponseDuration := 0 * time.Second
	for {
		nc.Logger.V(0).Info(fmt.Sprintf("Checking status of a request %s/%s", algorithmName, requestId))
		response, err := nc.ApiClient.AlgorithmV1ResultsAlgorithmNameRequestsRequestIdGet(context.TODO(), api.AlgorithmV1ResultsAlgorithmNameRequestsRequestIdGetParams{
			AlgorithmName: algorithmName,
			RequestId:     requestId,
		}, nc.getRequestOptions()...)

		if err != nil { // coverage-ignore
			return nil, err
		}

		switch result := response.(type) {
		case *api.ModelsRequestResult:

			nc.Logger.V(0).Info(fmt.Sprintf("Request %s/%s status: %s", algorithmName, requestId, result.Status.Value))

			if getRequestStub(result).IsFinished() {
				nc.Logger.V(0).Info(fmt.Sprintf("Request %s/%s finished", algorithmName, requestId))
				return result, nil
			}

			if pollInterval != nil {
				time.Sleep(*pollInterval)
			} else {
				time.Sleep(5 * time.Second)
			}
		case *api.AlgorithmV1ResultsAlgorithmNameRequestsRequestIdGetBadRequestApplicationJSON, *api.AlgorithmV1ResultsAlgorithmNameRequestsRequestIdGetBadRequestTextPlain:
			if invalidRequestResponseDuration > 5*time.Minute {
				return nil, models2.NewBadRequestError(fmt.Errorf("invalid request parameters: algorithm '%s' or request id '%s'", algorithmName, requestId))
			}

			nc.Logger.V(0).Info("received bad request when trying to read a result - possible lag in submission accounting, will try again")

			if pollInterval != nil {
				invalidRequestResponseDuration += *pollInterval
				time.Sleep(*pollInterval)
			} else {
				invalidRequestResponseDuration += 5 * time.Second
				time.Sleep(5 * time.Second)
			}

		case *api.AlgorithmV1ResultsAlgorithmNameRequestsRequestIdGetUnauthorizedApplicationJSON, *api.AlgorithmV1ResultsAlgorithmNameRequestsRequestIdGetUnauthorizedTextPlain, *api.AlgorithmV1ResultsAlgorithmNameRequestsRequestIdGetUnauthorizedTextHTML: // coverage-ignore
			return nil, models2.NewUnauthorizedError(fmt.Errorf("client credentials not recognized or missing for algorithm/requestId '%s'/'%s'", algorithmName, requestId))
		case *api.AlgorithmV1ResultsAlgorithmNameRequestsRequestIdGetNotFoundApplicationJSON, *api.AlgorithmV1ResultsAlgorithmNameRequestsRequestIdGetNotFoundTextPlain:
			return nil, nil
		default: // coverage-ignore
			return nil, models2.NewSdkErr(fmt.Errorf("unhandled response type for algorithm/requestId '%s'/'%s'", algorithmName, requestId))
		}
	}
}

func (nc *NexusSchedulerClient) awaitRuns(runs iter.Seq2[*api.ModelsTaggedRequestResult, error], pollInterval *time.Duration, completed *chan int32) ([]*api.ModelsTaggedRequestResult, error) {
	resultChannel := make(chan *AwaitTaggedResult, 10)
	var wg sync.WaitGroup

	for run, runErr := range runs {
		wg.Add(1)
		go func() {
			defer func() {
				nc.Logger.V(0).Info(fmt.Sprintf("Received result for %s/%s", run.AlgorithmName.Value, run.RequestId.Value))
				if completed != nil {
					*completed <- 1
				}
				wg.Done()
			}()

			nc.Logger.V(0).Info(fmt.Sprintf("Starting await of a run %s/%s", run.AlgorithmName.Value, run.RequestId.Value))

			if runErr != nil {
				resultChannel <- &AwaitTaggedResult{
					Error:  runErr,
					Result: nil,
				}
				nc.Logger.V(0).Error(runErr, fmt.Sprintf("Await of the run %s/%s failed", run.AlgorithmName.Value, run.RequestId.Value))
				return
			}

			result, err := nc.awaitRun(run.RequestId.Value, run.AlgorithmName.Value, pollInterval)
			if err != nil {
				resultChannel <- &AwaitTaggedResult{
					Error:  err,
					Result: nil,
				}
				nc.Logger.V(0).Error(err, fmt.Sprintf("Await of the run %s/%s failed", run.AlgorithmName.Value, run.RequestId.Value))
				return
			}

			resultChannel <- &AwaitTaggedResult{
				Error: nil,
				Result: &api.ModelsTaggedRequestResult{
					AlgorithmName: api.OptString{
						Value: run.AlgorithmName.Value,
						Set:   true,
					},
					RequestId: api.OptString{
						Value: result.RequestId.Value,
						Set:   true,
					},
					ResultUri: api.OptString{
						Value: result.ResultUri.Value,
						Set:   true,
					},
					RunErrorMessage: api.OptString{
						Value: result.RunErrorMessage.Value,
						Set:   true,
					},
					Status: api.OptString{
						Value: result.Status.Value,
						Set:   true,
					},
				},
			}
		}()
	}

	go func() {
		wg.Wait()
		nc.Logger.V(0).Info("Successfully awaited all tagged runs")
		close(resultChannel)
	}()

	results := []*api.ModelsTaggedRequestResult{}
	for result := range resultChannel {
		if result.Error != nil {
			return nil, result.Error
		}
		results = append(results, result.Result)
	}

	return results, nil
}

func (nc *NexusSchedulerClient) getRuns(tags []string, algorithmName *string) iter.Seq2[*api.ModelsTaggedRequestResult, error] {
	return func(yield func(requestResult *api.ModelsTaggedRequestResult, err error) bool) {
		for _, tag := range tags {
			taggedRunsResponse, err := nc.ApiClient.AlgorithmV1ResultsTagsRequestTagGet(context.TODO(), api.AlgorithmV1ResultsTagsRequestTagGetParams{RequestTag: tag}, nc.getRequestOptions()...)
			if err != nil {
				yield(nil, models2.NewSdkErr(err))
				return
			}

			switch taggedRunResponseType := taggedRunsResponse.(type) {
			case *api.AlgorithmV1ResultsTagsRequestTagGetOKApplicationJSON:
				for _, modelRequestResult := range *taggedRunResponseType {
					// include the run if algorithm name is not provided
					// if provided, only include those that have a matching name
					if algorithmName == nil || modelRequestResult.AlgorithmName.Value == *algorithmName {
						if !yield(&modelRequestResult, nil) {
							return
						}
					}
				}
				if err != nil {
					if !yield(nil, models2.NewSdkErr(err)) {
						return
					}
				}
			case *api.AlgorithmV1ResultsTagsRequestTagGetBadRequestApplicationJSON, *api.AlgorithmV1ResultsTagsRequestTagGetBadRequestTextPlain:
				if !yield(nil, models2.NewBadRequestError(fmt.Errorf("invalid request for tag %s", tag))) {
					return
				}
			case *api.AlgorithmV1ResultsTagsRequestTagGetUnauthorizedApplicationJSON, *api.AlgorithmV1ResultsTagsRequestTagGetUnauthorizedTextPlain, *api.AlgorithmV1ResultsTagsRequestTagGetUnauthorizedTextHTML: // coverage-ignore
				if !yield(nil, models2.NewUnauthorizedError(fmt.Errorf("client credentials not accepted or missing for tag '%s'", tag))) {
					return
				}
			default: // coverage-ignore
				if !yield(nil, models2.NewSdkErr(fmt.Errorf("unhandled response type for tag '%s'", tag))) {
					return
				}
			}
		}
	}
}

// GetRunResults retrieves run results for all runs with a matching tag, and optionally, an algorithm name
func (nc *NexusSchedulerClient) GetRunResults(tag string, algorithmName *string) iter.Seq2[*api.ModelsTaggedRequestResult, error] {
	return nc.getRuns([]string{tag}, algorithmName)
}

// AwaitRun awaits results for a submission identified by a request id and an algorithm name
func (nc *NexusSchedulerClient) AwaitRun(requestId string, algorithmName string, pollInterval *time.Duration) (*api.ModelsRequestResult, error) {
	resultChannel := make(chan *AwaitResult, 1)
	go func() {
		result, err := nc.awaitRun(requestId, algorithmName, pollInterval)
		if err != nil {
			resultChannel <- &AwaitResult{
				Error:  err,
				Result: nil,
			}
			close(resultChannel)
			return
		}

		resultChannel <- &AwaitResult{
			Error:  nil,
			Result: result,
		}
		close(resultChannel)
	}()

	runResult := <-resultChannel

	return runResult.Result, runResult.Error
}

// AwaitTaggedRuns awaits results for submissions that use provided tags. In case algorithm name is not nil, only submission with a matching algorithm name will be awaited
func (nc *NexusSchedulerClient) AwaitTaggedRuns(tags []string, algorithmName *string, pollInterval *time.Duration, completed *chan int32) (iter.Seq[*api.ModelsTaggedRequestResult], error) {
	runResults, err := nc.awaitRuns(nc.getRuns(tags, algorithmName), pollInterval, completed)
	if err != nil { // coverage-ignore
		return nil, err
	}

	return func(yield func(requestResult *api.ModelsTaggedRequestResult) bool) {
		for _, result := range runResults {
			if !yield(result) {
				return
			}
		}
	}, nil
}

func (nc *NexusSchedulerClient) CreateRun(request *api.ModelsAlgorithmRequest, algorithmName string, dryRun *bool) (string, error) {
	dryRunValue := ""
	if dryRun != nil {
		dryRunValue = strconv.FormatBool(*dryRun)
	}

	createdRunResponse, err := nc.ApiClient.AlgorithmV1RunAlgorithmNamePost(context.TODO(), request, api.AlgorithmV1RunAlgorithmNamePostParams{AlgorithmName: algorithmName, DryRun: api.OptString{
		Value: dryRunValue,
		Set:   dryRun != nil,
	}}, nc.getRequestOptions()...)

	if err != nil { // coverage-ignore
		return "", models2.NewSdkErr(err)
	}

	switch createdRunResponseType := createdRunResponse.(type) {
	case *api.AlgorithmV1RunAlgorithmNamePostBadRequestTextPlain:
		responseBytes, _ := io.ReadAll(createdRunResponseType.Data)
		return "", models2.NewBadRequestError(errors.New(string(responseBytes)))
	case *api.AlgorithmV1RunAlgorithmNamePostUnauthorizedApplicationJSON, *api.AlgorithmV1RunAlgorithmNamePostUnauthorizedTextPlain, *api.AlgorithmV1RunAlgorithmNamePostUnauthorizedTextHTML: // coverage-ignore
		return "", models2.NewUnauthorizedError(fmt.Errorf("client credentials not recognized or missing for algorithm '%s'", algorithmName))
	case *api.AlgorithmV1RunAlgorithmNamePostInternalServerErrorApplicationJSON, *api.AlgorithmV1RunAlgorithmNamePostInternalServerErrorTextPlain, *api.AlgorithmV1RunAlgorithmNamePostInternalServerErrorTextHTML: // coverage-ignore
		return "", models2.NewInternalServerError(fmt.Errorf("server error while creating a run request for algorithm '%s'", algorithmName))
	case *api.AlgorithmV1RunAlgorithmNamePostAcceptedApplicationJSON:
		return (*createdRunResponseType)["requestId"], nil
	case *api.AlgorithmV1RunAlgorithmNamePostBadRequestApplicationJSON:
		return "", models2.NewSdkErr(fmt.Errorf("unexpected response type '%s' for algorithm '%s'", *createdRunResponseType, algorithmName))
	default: // coverage-ignore
		return "", models2.NewSdkErr(fmt.Errorf("unhandled response type '%s' for algorithm '%s'", createdRunResponseType, algorithmName))
	}
}

func (nc *NexusSchedulerClient) GetRun(requestId string, algorithm string) (*api.ModelsRequestResult, error) {
	getRunResponse, err := nc.ApiClient.AlgorithmV1ResultsAlgorithmNameRequestsRequestIdGet(context.TODO(), api.AlgorithmV1ResultsAlgorithmNameRequestsRequestIdGetParams{AlgorithmName: algorithm, RequestId: requestId}, nc.getRequestOptions()...)

	if err != nil { // coverage-ignore
		return nil, models2.NewSdkErr(err)
	}

	switch getRunResponseType := getRunResponse.(type) {
	case *api.AlgorithmV1ResultsAlgorithmNameRequestsRequestIdGetBadRequestApplicationJSON, *api.AlgorithmV1ResultsAlgorithmNameRequestsRequestIdGetBadRequestTextPlain:
		return nil, models2.NewBadRequestError(fmt.Errorf("invalid request parameters: algorithm '%s' or request id '%s'", algorithm, requestId))
	case *api.AlgorithmV1ResultsAlgorithmNameRequestsRequestIdGetUnauthorizedApplicationJSON, *api.AlgorithmV1ResultsAlgorithmNameRequestsRequestIdGetUnauthorizedTextPlain, *api.AlgorithmV1ResultsAlgorithmNameRequestsRequestIdGetUnauthorizedTextHTML: // coverage-ignore
		return nil, models2.NewUnauthorizedError(fmt.Errorf("client credentials not recognized or missing for algorithm/requestId '%s'/'%s'", algorithm, requestId))
	case *api.AlgorithmV1ResultsAlgorithmNameRequestsRequestIdGetNotFoundApplicationJSON, *api.AlgorithmV1ResultsAlgorithmNameRequestsRequestIdGetNotFoundTextPlain:
		return nil, nil
	case *api.ModelsRequestResult:
		return getRunResponseType, nil
	default: // coverage-ignore
		return nil, models2.NewSdkErr(fmt.Errorf("unhandled response type for algorithm/requestId '%s'/'%s'", algorithm, requestId))
	}
}

func (nc *NexusSchedulerClient) GetMetadata(requestId string, algorithm string) (*api.ModelsCheckpointedRequest, error) {
	getMetadataResponse, err := nc.ApiClient.AlgorithmV1MetadataAlgorithmNameRequestsRequestIdGet(context.TODO(), api.AlgorithmV1MetadataAlgorithmNameRequestsRequestIdGetParams{AlgorithmName: algorithm, RequestId: requestId}, nc.getRequestOptions()...)

	if err != nil { // coverage-ignore
		return nil, models2.NewSdkErr(err)
	}

	switch getMetadataResponseType := getMetadataResponse.(type) {
	case *api.AlgorithmV1MetadataAlgorithmNameRequestsRequestIdGetBadRequestApplicationJSON, *api.AlgorithmV1MetadataAlgorithmNameRequestsRequestIdGetBadRequestTextPlain:
		return nil, models2.NewBadRequestError(fmt.Errorf("invalid request parameters: algorithm '%s' or request id '%s'", algorithm, requestId))
	case *api.AlgorithmV1MetadataAlgorithmNameRequestsRequestIdGetUnauthorizedApplicationJSON, *api.AlgorithmV1MetadataAlgorithmNameRequestsRequestIdGetUnauthorizedTextPlain, *api.AlgorithmV1MetadataAlgorithmNameRequestsRequestIdGetUnauthorizedTextHTML: // coverage-ignore
		return nil, models2.NewUnauthorizedError(fmt.Errorf("client credentials not recognized or missing for algorithm/requestId '%s'/'%s'", algorithm, requestId))
	case *api.AlgorithmV1MetadataAlgorithmNameRequestsRequestIdGetNotFoundApplicationJSON, *api.AlgorithmV1MetadataAlgorithmNameRequestsRequestIdGetNotFoundTextPlain:
		return nil, nil
	case *api.ModelsCheckpointedRequest:
		return getMetadataResponseType, nil
	default: // coverage-ignore
		return nil, models2.NewSdkErr(fmt.Errorf("unhandled response type for algorithm/requestId '%s'/'%s'", algorithm, requestId))
	}
}

func (nc *NexusSchedulerClient) GetBufferedRequest(requestId string, algorithm string) (string, error) {
	getBufferedResponse, err := nc.ApiClient.AlgorithmV1BufferAlgorithmNameRequestsRequestIdGet(context.TODO(), api.AlgorithmV1BufferAlgorithmNameRequestsRequestIdGetParams{AlgorithmName: algorithm, RequestId: requestId}, nc.getRequestOptions()...)

	if err != nil { // coverage-ignore
		return "", models2.NewSdkErr(err)
	}

	switch getBufferedResponseType := getBufferedResponse.(type) {
	case *api.AlgorithmV1BufferAlgorithmNameRequestsRequestIdGetBadRequestTextPlain, *api.AlgorithmV1BufferAlgorithmNameRequestsRequestIdGetBadRequestTextHTML:
		return "", models2.NewBadRequestError(fmt.Errorf("invalid request parameters: algorithm '%s' or request id '%s'", algorithm, requestId))
	case *api.AlgorithmV1BufferAlgorithmNameRequestsRequestIdGetUnauthorizedApplicationJSON, *api.AlgorithmV1BufferAlgorithmNameRequestsRequestIdGetUnauthorizedTextPlain, *api.AlgorithmV1BufferAlgorithmNameRequestsRequestIdGetUnauthorizedTextHTML:
		return "", models2.NewUnauthorizedError(fmt.Errorf("client credentials not recognized or missing for algorithm/requestId '%s'/'%s'", algorithm, requestId))
	case *api.AlgorithmV1BufferAlgorithmNameRequestsRequestIdGetOKTextPlain:
		responseBytes, _ := io.ReadAll(getBufferedResponseType.Data)
		return string(responseBytes), nil
	default: // coverage-ignore
		return "", models2.NewSdkErr(fmt.Errorf("unhandled response type for algorithm/requestId '%s'/'%s'", algorithm, requestId))
	}
}

func (nc *NexusSchedulerClient) CancelRun(cancellation *api.ModelsCancellationRequest, requestId string, algorithm string) error {
	cancelledResponse, err := nc.ApiClient.AlgorithmV1CancelAlgorithmNameRequestsRequestIdPost(context.TODO(), cancellation, api.AlgorithmV1CancelAlgorithmNameRequestsRequestIdPostParams{
		AlgorithmName: algorithm,
		RequestId:     requestId,
	}, nc.getRequestOptions()...)

	if err != nil { // coverage-ignore
		return models2.NewSdkErr(err)
	}

	switch cancelledResponse.(type) {
	case *api.AlgorithmV1CancelAlgorithmNameRequestsRequestIdPostBadRequestApplicationJSON, *api.AlgorithmV1CancelAlgorithmNameRequestsRequestIdPostBadRequestTextPlain, *api.AlgorithmV1CancelAlgorithmNameRequestsRequestIdPostBadRequestTextHTML:
		return models2.NewBadRequestError(fmt.Errorf("invalid request parameters: algorithm '%s' or request id '%s'", algorithm, requestId))
	case *api.AlgorithmV1CancelAlgorithmNameRequestsRequestIdPostUnauthorizedApplicationJSON, *api.AlgorithmV1CancelAlgorithmNameRequestsRequestIdPostUnauthorizedTextPlain, *api.AlgorithmV1CancelAlgorithmNameRequestsRequestIdPostUnauthorizedTextHTML:
		return models2.NewUnauthorizedError(fmt.Errorf("client credentials not recognized or missing for algorithm/requestId '%s'/'%s'", algorithm, requestId))
	case *api.AlgorithmV1CancelAlgorithmNameRequestsRequestIdPostOKApplicationJSON, *api.AlgorithmV1CancelAlgorithmNameRequestsRequestIdPostOKTextPlain, *api.AlgorithmV1CancelAlgorithmNameRequestsRequestIdPostOKTextHTML:
		return nil
	default: // coverage-ignore
		return models2.NewSdkErr(fmt.Errorf("unhandled response type for algorithm/requestId '%s'/'%s'", algorithm, requestId))

	}
}
