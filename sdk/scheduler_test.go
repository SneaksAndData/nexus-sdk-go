package sdk

import (
	"context"
	"github.com/SneaksAndData/nexus-core/pkg/telemetry"
	"k8s.io/klog/v2"
	"strings"
	"testing"
)

type fixture struct {
	t      *testing.T
	url    string
	logger *klog.Logger
	client *NexusSchedulerClient
}

func newFixture(t *testing.T) *fixture {
	f := &fixture{}
	f.t = t
	appLogger, _ := telemetry.ConfigureLogger(context.TODO(), map[string]string{}, "info")
	klog.SetSlogLogger(appLogger)

	logger := klog.FromContext(context.TODO())

	f.logger = &logger
	f.client = NewNexusSchedulerClient(f.url, f.logger, nil)

	return f
}

func Test_GetRunResultsTagDoesNotExist(t *testing.T) {
	f := newFixture(t)
	for _, err := range f.client.GetRunResults("aaa", nil) {
		if err == nil {
			f.t.Error("GetRunResults should have returned an error")
		}

		if err != nil && !strings.Contains(err.Error(), "no submissions found for tag") {
			f.t.Error("Incorrect error returned, should be: no submissions found for tag")
		}
	}
}

func Test_GetRunResultsTagExists(t *testing.T) {
	f := newFixture(t)
	expectedLength := 3
	actualLength := 0
	for _, err := range f.client.GetRunResults("abc", nil) {
		if err != nil {
			f.t.Error(err)
		}
		actualLength++
	}

	if actualLength != expectedLength {
		f.t.Errorf("GetRunResults should have returned the expected %d submissions", expectedLength)
	}
}
