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

func awaitRuns(client *api.Client, runs []*api.ModelsTaggedRequestResult, pollInterval *time.Duration) ([]*api.ModelsRequestResult, error) {
	resultChannel := make(chan *api.ModelsRequestResult, len(runs))
	for _, run := range runs {
		go func() {
			result, err := awaitRun(client, run.RequestId.Value, run.AlgorithmName.Value, pollInterval)
			if err != nil {
				close(resultChannel)
			}

			resultChannel <- result
		}()
	}

	results := []*api.ModelsRequestResult{}
	for result := range resultChannel {
		results = append(results, result)
	}

	return results, nil
}

func AwaitTaggedRuns(client *api.Client, tags []string) (iter.Seq[*api.ModelsRequestResult], error) {
	runsToAwait := []*api.ModelsTaggedRequestResult{}

	for _, tag := range tags {
		taggedRunsResponse, err := client.ResultsTagsTagGet(context.TODO(), api.ResultsTagsTagGetParams{Tag: tag})
		if err != nil {
			return nil, err
		}

		switch taggedRunResponseType := taggedRunsResponse.(type) {
		case *api.ResultsTagsTagGetOKApplicationJSON:
			for _, modelRequestResult := range *taggedRunResponseType {
				runsToAwait = append(runsToAwait, &modelRequestResult)
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

	runResults, err := awaitRuns(client, runsToAwait, nil)
	if err != nil {
		return nil, err
	}

	return func(yield func(requestResult *api.ModelsRequestResult) bool) {
		for _, result := range runResults {
			if !yield(result) {
				return
			}
		}
	}, nil
}
