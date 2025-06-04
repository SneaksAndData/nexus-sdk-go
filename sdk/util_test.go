package sdk

import (
	api "github.com/SneaksAndData/nexus-sdk-go/pkg/generated/scheduler"
	"strings"
	"testing"
)

func Test_GetRunResultsTagDoesNotExist(t *testing.T) {
	client, _ := api.NewClient("http://localhost:8080")
	for _, err := range GetRunResults(client, "aaa", nil) {
		if err == nil {
			t.Error("GetRunResults should have returned an error")
		}

		if err != nil && !strings.Contains(err.Error(), "no submissions found for tag") {
			t.Error("Incorrect error returned, should be: no submissions found for tag")
		}
	}
}

func Test_GetRunResultsTagExists(t *testing.T) {
	client, _ := api.NewClient("http://localhost:8080")
	expectedLength := 3
	actualLength := 0
	for _, err := range GetRunResults(client, "abc", nil) {
		if err != nil {
			t.Error(err)
		}
		actualLength++
	}

	if actualLength != expectedLength {
		t.Errorf("GetRunResults should have returned the expected %d submissions", expectedLength)
	}
}
