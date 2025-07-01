package sdk

import (
	"context"
	"fmt"
	api "github.com/SneaksAndData/nexus-sdk-go/pkg/generated/receiver"
	models2 "github.com/SneaksAndData/nexus-sdk-go/sdk/models"
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

	if pinner != nil {
		pinner.Pin(client)
		pinner.Pin(logger)
		pinner.Pin(options)
		pinner.Pin(result)
	}

	return result
}

func (nc *NexusReceiverClient) RefreshAuth(token string) {
	nc.RequestOptions = &[]api.RequestOption{GetReceiverAuthOption(token)}
}

func (nc *NexusReceiverClient) getRequestOptions() []api.RequestOption {
	if nc.RequestOptions == nil {
		return []api.RequestOption{}
	}

	return *nc.RequestOptions
}

func (nc *NexusReceiverClient) CompleteRequest(result *api.ModelsAlgorithmResult, algorithm string, requestId string) error {
	completeResponse, err := nc.ApiClient.AlgorithmV12CompleteAlgorithmNameRequestsRequestIdPost(context.TODO(), result, api.AlgorithmV12CompleteAlgorithmNameRequestsRequestIdPostParams{
		AlgorithmName: algorithm,
		RequestId:     requestId,
	}, nc.getRequestOptions()...)

	if err != nil {
		return models2.NewSdkErr(err)
	}

	switch completeResponse.(type) {
	case *api.AlgorithmV12CompleteAlgorithmNameRequestsRequestIdPostAcceptedApplicationJSON:
		return nil
	case *api.AlgorithmV12CompleteAlgorithmNameRequestsRequestIdPostBadRequestApplicationJSON, *api.AlgorithmV12CompleteAlgorithmNameRequestsRequestIdPostBadRequestTextPlain:
		return models2.NewBadRequestError(fmt.Errorf("invalid request parameters: algorithm '%s' or request id '%s'", algorithm, requestId))
	case *api.AlgorithmV12CompleteAlgorithmNameRequestsRequestIdPostUnauthorizedApplicationJSON, *api.AlgorithmV12CompleteAlgorithmNameRequestsRequestIdPostUnauthorizedTextPlain, *api.AlgorithmV12CompleteAlgorithmNameRequestsRequestIdPostUnauthorizedTextHTML:
		return models2.NewUnauthorizedError(fmt.Errorf("client credentials not recognized or missing for algorithm/requestId '%s'/'%s'", algorithm, requestId))
	case *api.AlgorithmV12CompleteAlgorithmNameRequestsRequestIdPostNotFoundApplicationJSON, *api.AlgorithmV12CompleteAlgorithmNameRequestsRequestIdPostNotFoundTextPlain:
		return models2.NewNotFoundError(fmt.Errorf("unknown request: a combination of algorithm '%s'/'%s' does not exist", algorithm, requestId))
	default:
		return models2.NewSdkErr(fmt.Errorf("unhandled response type for algorithm/requestId '%s'/'%s'", algorithm, requestId))
	}
}
