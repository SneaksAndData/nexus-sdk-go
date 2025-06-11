package sdk

import (
	api "github.com/SneaksAndData/nexus-sdk-go/pkg/generated/scheduler"
	"net/http"
)

// GetAuthOption provides a request modifier option that sets Auth header value
func GetAuthOption(authHeaderValue string) api.RequestOption {
	return api.WithEditRequest(func(req *http.Request) error {
		req.Header.Set("Authorization", authHeaderValue)
		return nil
	})
}
