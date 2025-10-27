package sdk

import (
	"context"
	"fmt"
	api "github.com/SneaksAndData/nexus-sdk-go/pkg/generated/receiver"
	models2 "github.com/SneaksAndData/nexus-sdk-go/sdk/models"
	"github.com/aws/smithy-go/ptr"
	"k8s.io/klog/v2"
	"runtime"
)

type NexusReceiverClient struct {
	ApiClient      *api.Client
	RequestOptions *[]api.RequestOption
	Logger         *klog.Logger
}

func NewNexusReceiverClient(receiverUrl string, logger *klog.Logger, options *[]api.RequestOption, pinner *runtime.Pinner) *NexusReceiverClient {
	client, err := api.NewClient(receiverUrl)

	if err != nil {
		logger.Error(err, "unable to initialize Nexus receiver client")
		klog.FlushAndExit(klog.ExitFlushTimeout, 1)
	}

	result := &NexusReceiverClient{
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

func (nc *NexusReceiverClient) RefreshAuth(token string) { // coverage-ignore
	nc.RequestOptions = &[]api.RequestOption{GetReceiverAuthOption(token)}
}

func (nc *NexusReceiverClient) getRequestOptions() []api.RequestOption {
	if nc.RequestOptions == nil {
		return []api.RequestOption{}
	}

	return *nc.RequestOptions
}

func (nc *NexusReceiverClient) CompleteRequest(result *api.ModelsAlgorithmResult, algorithm string, requestId string) error {
	completeResponse, err := nc.ApiClient.AlgorithmV1CompleteAlgorithmNameRequestsRequestIdPost(context.TODO(), result, api.AlgorithmV1CompleteAlgorithmNameRequestsRequestIdPostParams{
		AlgorithmName: algorithm,
		RequestId:     requestId,
	}, nc.getRequestOptions()...)

	if err != nil { // coverage-ignore
		return mapApiError(err)
	}

	switch completeResponse.(type) {
	case *api.AlgorithmV1CompleteAlgorithmNameRequestsRequestIdPostAcceptedApplicationJSON:
		return nil
	case *api.AlgorithmV1CompleteAlgorithmNameRequestsRequestIdPostBadRequestApplicationJSON, *api.AlgorithmV1CompleteAlgorithmNameRequestsRequestIdPostBadRequestTextPlain:
		return models2.NewBadRequestError(fmt.Errorf("invalid request parameters: algorithm '%s' or request id '%s'", algorithm, requestId))
	case *api.AlgorithmV1CompleteAlgorithmNameRequestsRequestIdPostUnauthorizedApplicationJSON, *api.AlgorithmV1CompleteAlgorithmNameRequestsRequestIdPostUnauthorizedTextPlain, *api.AlgorithmV1CompleteAlgorithmNameRequestsRequestIdPostUnauthorizedTextHTML: // coverage-ignore
		return models2.NewUnauthorizedError(fmt.Errorf("client credentials not recognized or missing for algorithm/requestId '%s'/'%s'", algorithm, requestId))
	case *api.AlgorithmV1CompleteAlgorithmNameRequestsRequestIdPostNotFoundApplicationJSON, *api.AlgorithmV1CompleteAlgorithmNameRequestsRequestIdPostNotFoundTextPlain: // coverage-ignore
		return models2.NewNotFoundError(fmt.Errorf("unknown request: a combination of algorithm '%s'/'%s' does not exist", algorithm, requestId))
	default: // coverage-ignore
		return models2.NewSdkErr(fmt.Errorf("unhandled response type for algorithm/requestId '%s'/'%s'", algorithm, requestId))
	}
}

func (nc *NexusReceiverClient) CheckRequest(algorithm string, requestId string) (*bool, error) {
	checkResponse, err := nc.ApiClient.AlgorithmV1CheckAlgorithmNameRequestsRequestIdGet(context.TODO(), api.AlgorithmV1CheckAlgorithmNameRequestsRequestIdGetParams{
		AlgorithmName: algorithm,
		RequestId:     requestId,
	}, nc.getRequestOptions()...)

	if err != nil { // coverage-ignore
		return nil, mapApiError(err)
	}

	switch checkResult := checkResponse.(type) {
	case *api.ModelsCheckRunResponse:
		return ptr.Bool(checkResult.IsProcessed.Value), nil
	case *api.AlgorithmV1CheckAlgorithmNameRequestsRequestIdGetUnauthorizedTextHTML, *api.AlgorithmV1CheckAlgorithmNameRequestsRequestIdGetUnauthorizedApplicationJSON: // coverage-ignore
		return nil, models2.NewUnauthorizedError(fmt.Errorf("client credentials not recognized or missing for algorithm/requestId '%s'/'%s'", algorithm, requestId))
	case *api.AlgorithmV1CheckAlgorithmNameRequestsRequestIdGetBadRequestTextHTML, *api.AlgorithmV1CheckAlgorithmNameRequestsRequestIdGetBadRequestApplicationJSON: // coverage-ignore
		return nil, models2.NewBadRequestError(fmt.Errorf("invalid request parameters: algorithm '%s' or request id '%s'", algorithm, requestId))
	case *api.AlgorithmV1CheckAlgorithmNameRequestsRequestIdGetNotFoundTextHTML, *api.AlgorithmV1CheckAlgorithmNameRequestsRequestIdGetNotFoundApplicationJSON: // coverage-ignore
		return nil, models2.NewNotFoundError(fmt.Errorf("unknown request: a combination of algorithm '%s'/'%s' does not exist", algorithm, requestId))
	default: // coverage-ignore
		return nil, models2.NewSdkErr(fmt.Errorf("unhandled response type for algorithm/requestId '%s'/'%s'", algorithm, requestId))
	}
}
