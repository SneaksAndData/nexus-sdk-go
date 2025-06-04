package sdk

import (
	"context"
	"fmt"
	"github.com/SneaksAndData/nexus-core/pkg/checkpoint/models"
	"github.com/SneaksAndData/nexus-sdk-go/pkg/generated/scheduler"
	"iter"
	"time"
)

type AwaitResult struct {
	Result *api.ModelsTaggedRequestResult
	Error  error
}

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

func awaitRuns(client *api.Client, runs iter.Seq2[*api.ModelsTaggedRequestResult, error], pollInterval *time.Duration) ([]*api.ModelsTaggedRequestResult, error) {
	resultChannel := make(chan *AwaitResult)
	for run, runErr := range runs {
		go func() {
			if runErr != nil {
				resultChannel <- &AwaitResult{
					Error:  runErr,
					Result: nil,
				}
				close(resultChannel)
			}

			result, err := awaitRun(client, run.RequestId.Value, run.AlgorithmName.Value, pollInterval)
			if err != nil {
				resultChannel <- &AwaitResult{
					Error:  err,
					Result: nil,
				}
				close(resultChannel)
			}

			resultChannel <- &AwaitResult{
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

	results := []*api.ModelsTaggedRequestResult{}
	for result := range resultChannel {
		if result.Error != nil {
			return nil, result.Error
		}
		results = append(results, result.Result)
	}

	return results, nil
}

func getRuns(client *api.Client, tags []string, algorithmName *string) iter.Seq2[*api.ModelsTaggedRequestResult, error] {
	return func(yield func(requestResult *api.ModelsTaggedRequestResult, err error) bool) {
		for _, tag := range tags {
			taggedRunsResponse, err := client.ResultsTagsTagGet(context.TODO(), api.ResultsTagsTagGetParams{Tag: tag})
			if err != nil {
				if !yield(nil, err) {
					return
				}
			}

			switch taggedRunResponseType := taggedRunsResponse.(type) {
			case *api.ResultsTagsTagGetOKApplicationJSON:
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
					if !yield(nil, err) {
						return
					}
				}
			case *api.ResultsTagsTagGetNotFoundApplicationJSON:
				if !yield(nil, fmt.Errorf("no submissions found for tag %s", tag)) {
					return
				}
			case *api.ResultsTagsTagGetBadRequestApplicationJSON:
				if !yield(nil, fmt.Errorf("server returned BadRequest request for tag %s", tag)) {
					return
				}
			default:
				if !yield(nil, fmt.Errorf("unhandled response type for tag %s", tag)) {
					return
				}
			}
		}
	}
}

// GetRunResults retrieves run results for all runs with a matching tag, and optionally, an algorithm name
func GetRunResults(client *api.Client, tag string, algorithmName *string) iter.Seq2[*api.ModelsTaggedRequestResult, error] {
	return getRuns(client, []string{tag}, algorithmName)
}

// AwaitRun awaits results for a submission identified by a request id and an algorithm name
func AwaitRun(client *api.Client, requestId string, algorithmName string, pollInterval *time.Duration) (*api.ModelsRequestResult, error) {
	resultChannel := make(chan *api.ModelsRequestResult, 1)
	go func() {
		result, err := awaitRun(client, requestId, algorithmName, pollInterval)
		if err != nil {
			close(resultChannel)
		}

		resultChannel <- result
	}()

	runResult := <-resultChannel

	return runResult, nil
}

// AwaitTaggedRuns awaits results for submissions that use provided tags. In case algorithm name is not nil, only submission with a matching algorithm name will be awaited
func AwaitTaggedRuns(client *api.Client, tags []string, algorithmName *string, pollInterval *time.Duration) (iter.Seq[*api.ModelsTaggedRequestResult], error) {
	runResults, err := awaitRuns(client, getRuns(client, tags, algorithmName), pollInterval)
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
