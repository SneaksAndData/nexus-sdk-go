package sdk

import (
	"context"
	"errors"
	"github.com/SneaksAndData/nexus-core/pkg/checkpoint/models"
	"github.com/SneaksAndData/nexus-sdk-go/pkg/generated/scheduler"
	"iter"
	"time"
)

func getRequestStub(result scheduler.ModelsRequestResult) *models.CheckpointedRequest {
	return &models.CheckpointedRequest{
		Id:             result.RequestId.Value,
		LifecycleStage: result.Status.Value,
	}
}

func awaitRun(client *scheduler.Client, requestId string, algorithmName string, pollInterval *time.Duration) (*scheduler.ModelsRequestResult, error) {
	for {
		result, err := client.ResultsAlgorithmNameRequestsRequestIdGet(context.TODO(), scheduler.ResultsAlgorithmNameRequestsRequestIdGetParams{
			AlgorithmName: algorithmName,
			RequestId:     requestId,
		})

		if err != nil {
			return nil, err
		}

		if getRequestStub(result).IsFinished() {
			return result, nil
		}

		if pollInterval != nil {
			time.Sleep(*pollInterval)
		} else {
			time.Sleep(5 * time.Second)
		}
	}
}

func awaitRuns(client *scheduler.Client, requestIds []string, algorithmName string, pollInterval *time.Duration) ([]*scheduler.ModelsRequestResult, error) {
	resultChannel := make(chan *scheduler.ModelsRequestResult, len(requestIds))
	for _, requestId := range requestIds {
		go func() {
			result, err := awaitRun(client, requestId, algorithmName, pollInterval)
			if err != nil {
				close(resultChannel)
			}

			resultChannel <- result
		}()
	}

	results := []*scheduler.ModelsRequestResult{}
	for result := range resultChannel {
		results = append(results, result)
	}

	return results, nil
}

func AwaitRunsByTag(client *scheduler.Client, tags []string) (iter.Seq[*scheduler.ModelsRequestResult], error) {
	runsToAwait := []scheduler.ModelsRequestResult{}

	for _, tag := range tags {
		taggedRunsResponse, err := client.ResultsTagsTagGet(context.TODO(), scheduler.ResultsTagsTagGetParams{Tag: tag})
		if err != nil {
			return nil, err
		}

		switch taggedRunResponseType := taggedRunsResponse.(type) {
		case *scheduler.ResultsTagsTagGetOKApplicationJSON:
			for _, modelRequestResult := range *taggedRunResponseType {
				runsToAwait = append(runsToAwait, modelRequestResult)
			}
			if err != nil {
				return nil, err
			}
		case *scheduler.ResultsTagsTagGetNotFoundApplicationJSON:
			return nil, errors.New("no submissions found for tag " + tag)
		case *scheduler.ResultsTagsTagGetBadRequestApplicationJSON:
			return nil, errors.New("server returned BadRequest request for tag " + tag)
		default:
			return nil, errors.New("Unhandled response type for tag " + tag)
		}
	}

	runResults, err := awaitRuns(client, []string{}, "", nil)
	if err != nil {
		return nil, err
	}

	return func(yield func(requestResult *scheduler.ModelsRequestResult) bool) {
		for _, result := range runResults {
			if !yield(result) {
				return
			}
		}
	}, nil
}
