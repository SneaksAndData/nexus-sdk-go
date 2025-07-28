package sdk

import (
	"context"
	"github.com/SneaksAndData/nexus-core/pkg/telemetry"
	api "github.com/SneaksAndData/nexus-sdk-go/pkg/generated/scheduler"
	"github.com/go-faster/jx"
	"k8s.io/klog/v2"
	"os"
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
	f := &fixture{
		url: os.Getenv("NEXUS_TEST_SCHEDULER_URL"),
		t:   t,
	}
	appLogger, _ := telemetry.ConfigureLogger(context.TODO(), map[string]string{}, "info")
	klog.SetSlogLogger(appLogger)

	logger := klog.FromContext(context.TODO())

	f.logger = &logger
	f.client = NewNexusSchedulerClient(f.url, f.logger, nil, nil)

	return f
}

func Test_GetRunResultsTagDoesNotExist(t *testing.T) {
	f := newFixture(t)
	for _, err := range f.client.GetRunResults("aaa", nil) {
		if err != nil {
			f.t.Errorf("GetRunResults should have not returned an error %s", err.Error())
		}
	}
}
func Test_PostNonExistingAlgorithm(t *testing.T) {
	f := newFixture(t)
	request := &api.ModelsAlgorithmRequest{
		AlgorithmParameters: api.ModelsAlgorithmRequestAlgorithmParameters{
			"algorithm": jx.Raw("smth"),
			"settingA":  jx.Raw("a"),
			"settingB":  jx.Raw("b"),
		},
	}

	_, err := f.client.CreateRun(request, "non-existing")
	if err == nil {
		f.t.Error("CreateRun should have returned an error, since algorithm 'non-existing' is not deployed")
	}

	if err != nil && !strings.Contains(strings.ToLower(err.Error()), "no valid configuration found") {
		f.t.Errorf("Incorrect error %s returned, should contain: `no valid configuration found`", err.Error())
	}
}

//
//func Test_GetRunResultsTagExists(t *testing.T) {
//	f := newFixture(t)
//	expectedLength := 3
//	actualLength := 0
//	for _, err := range f.client.GetRunResults("abc", nil) {
//		if err != nil {
//			f.t.Error(err)
//		}
//		actualLength++
//	}
//
//	if actualLength != expectedLength {
//		f.t.Errorf("GetRunResults should have returned the expected %d submissions", expectedLength)
//	}
//}
//
//func Test_AwaitRun(t *testing.T) {
//	f := newFixture(t)
//	runId, err := f.client.CreateRun(&api.ModelsAlgorithmRequest{
//		AlgorithmParameters: nil,
//		CustomConfiguration: api.OptV1NexusAlgorithmSpec{
//			Set: false,
//		},
//		ParentRequest: api.OptModelsAlgorithmRequestRef{
//			Set: false,
//		},
//		PayloadValidFor: api.OptString{
//			Set: false,
//		},
//		RequestApiVersion: api.OptString{
//			Set: false,
//		},
//		Tag: api.OptString{
//			Value: "abc",
//			Set:   true,
//		},
//	}, "omni-channel-solver")
//
//	if err != nil {
//		f.t.Error(err)
//	}
//
//	_, err = f.client.AwaitRun(runId, "omni-channel-solver", nil)
//
//	if err != nil {
//		f.t.Error(err)
//	}
//}
