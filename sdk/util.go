package sdk

import (
	"context"
	"errors"
	"fmt"
	"github.com/SneaksAndData/nexus-core/pkg/checkpoint/models"
	"github.com/SneaksAndData/nexus-sdk-go/pkg/generated/scheduler"
	"iter"
	"time"
)

func getRequestStub(result *api.ModelsRequestResult) *models.CheckpointedRequest {
	return &models.CheckpointedRequest{
		Id:             result.RequestId.Value,
		LifecycleStage: result.Status.Value,
	}
}

func awaitRun(client *api.Client, requestId string, algorithmName string, pollInterval *time.Duration) (*api.ModelsRequestResult, error) {
	for {
		response, err := client.ResultsAlgorithmNameRequestsRequestIdGet(context.TODO(), api.ResultsAlgorithmNameRequestsRequestIdGetParams{
			AlgorithmName: algorithmName,
			RequestId:     requestId,
		})

		if err != nil {
			return nil, err
		}

		switch result := response.(type) {
		case *api.ModelsRequestResult:
			if getRequestStub(result).IsFinished() {
				return result, nil
			}

			if pollInterval != nil {
				time.Sleep(*pollInterval)
			} else {
				time.Sleep(5 * time.Second)
			}
		case *api.ResultsAlgorithmNameRequestsRequestIdGetNotFoundApplicationJSON:
			return nil, fmt.Errorf("request %s for algorithm %s not found", requestId, algorithmName)
		case *api.ResultsAlgorithmNameRequestsRequestIdGetBadRequestApplicationJSON:
			return nil, fmt.Errorf("server returned BadRequest when looking up result for request %s for algorithm %s", requestId, algorithmName)
		default:
			return nil, fmt.Errorf("unexpected response type for request %s for algorithm %s", requestId, algorithmName)
		}
	}
}

func awaitRuns(client *api.Client, runs []*api.ModelsTaggedRequestResult, pollInterval *time.Duration) ([]*api.ModelsTaggedRequestResult, error) {
	resultChannel := make(chan *api.ModelsTaggedRequestResult, len(runs))
	for _, run := range runs {
		go func() {
			result, err := awaitRun(client, run.RequestId.Value, run.AlgorithmName.Value, pollInterval)
			if err != nil {
				close(resultChannel)
			}

			resultChannel <- &api.ModelsTaggedRequestResult{
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
			}
		}()
	}

	results := []*api.ModelsTaggedRequestResult{}
	for result := range resultChannel {
		results = append(results, result)
	}

	return results, nil
}

// AwaitTaggedRuns awaits results for submissions that use provided tags. In case algorithm name is not nil, only submission with a matching algorithm name will be awaited
func AwaitTaggedRuns(client *api.Client, tags []string, algorithmName *string, pollInterval *time.Duration) (iter.Seq[*api.ModelsTaggedRequestResult], error) {
	runsToAwait := []*api.ModelsTaggedRequestResult{}

	for _, tag := range tags {
		taggedRunsResponse, err := client.ResultsTagsTagGet(context.TODO(), api.ResultsTagsTagGetParams{Tag: tag})
		if err != nil {
			return nil, err
		}

		switch taggedRunResponseType := taggedRunsResponse.(type) {
		case *api.ResultsTagsTagGetOKApplicationJSON:
			for _, modelRequestResult := range *taggedRunResponseType {
				// include the run if algorithm name is not provided
				// if provided, only include those that have a matching name
				if algorithmName == nil || modelRequestResult.AlgorithmName.Value == *algorithmName {
					runsToAwait = append(runsToAwait, &modelRequestResult)
				}
			}
			if err != nil {
				return nil, err
			}
		case *api.ResultsTagsTagGetNotFoundApplicationJSON:
			return nil, errors.New("no submissions found for tag " + tag)
		case *api.ResultsTagsTagGetBadRequestApplicationJSON:
			return nil, errors.New("server returned BadRequest request for tag " + tag)
		default:
			return nil, errors.New("Unhandled response type for tag " + tag)
		}
	}

	runResults, err := awaitRuns(client, runsToAwait, pollInterval)
	if err != nil {
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
