package sdk

import (
	"fmt"
	rapi "github.com/SneaksAndData/nexus-sdk-go/pkg/generated/receiver"
	api "github.com/SneaksAndData/nexus-sdk-go/pkg/generated/scheduler"
	"net/http"
)

// GetSchedulerAuthOption provides a request modifier option that sets Auth header value
func GetSchedulerAuthOption(authHeaderValue string) api.RequestOption {
	return api.WithEditRequest(func(req *http.Request) error {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authHeaderValue))
		return nil
	})
}

// GetReceiverAuthOption provides a request modifier option that sets Auth header value
func GetReceiverAuthOption(authHeaderValue string) rapi.RequestOption {
	return rapi.WithEditRequest(func(req *http.Request) error {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authHeaderValue))
		return nil
	})
}
