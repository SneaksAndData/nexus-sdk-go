package sdk

import (
	"context"
	"github.com/SneaksAndData/nexus-core/pkg/telemetry"
	api "github.com/SneaksAndData/nexus-sdk-go/pkg/generated/scheduler"
	"github.com/go-faster/jx"
	"github.com/ogen-go/ogen/json"
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

func verifyNonExistingRun(testFixture *fixture, params api.ModelsAlgorithmRequestAlgorithmParameters) {
	request := &api.ModelsAlgorithmRequest{
		AlgorithmParameters: params,
		CustomConfiguration: api.OptV1NexusAlgorithmSpec{
			Set: false,
		},
		ParentRequest: api.OptModelsAlgorithmRequestRef{
			Set: false,
		},
		PayloadValidFor: api.OptString{
			Set: false,
		},
		RequestApiVersion: api.OptString{
			Set: false,
		},
		Tag: api.OptString{
			Set: false,
		},
	}

	_, err := testFixture.client.CreateRun(request, "non-existing")

	if err == nil {
		testFixture.t.Error("CreateRun should have returned an error, since algorithm 'non-existing' is not deployed")
	}

	if err != nil && !strings.Contains(strings.ToLower(err.Error()), "no valid configuration found") {
		testFixture.t.Errorf("Incorrect error '%s' returned, should contain: `no valid configuration found`", err.Error())
	}
}

func verifyExistingRun(testFixture *fixture, params api.ModelsAlgorithmRequestAlgorithmParameters, runTag string) {
	request := &api.ModelsAlgorithmRequest{
		AlgorithmParameters: params,
		CustomConfiguration: api.OptV1NexusAlgorithmSpec{
			Set: false,
		},
		ParentRequest: api.OptModelsAlgorithmRequestRef{
			Set: false,
		},
		PayloadValidFor: api.OptString{
			Set: false,
		},
		RequestApiVersion: api.OptString{
			Set: false,
		},
		Tag: api.OptString{
			Value: runTag,
			Set:   true,
		},
	}

	_, err := testFixture.client.CreateRun(request, "hello-world")

	if err != nil {
		testFixture.t.Errorf("Error '%s' returned, no errors expected", err.Error())
	}
}

func Test_GetRunResultsTagDoesNotExist(t *testing.T) {
	f := newFixture(t)
	for _, err := range f.client.GetRunResults("aaa", nil) {
		if err != nil {
			f.t.Errorf("GetRunResults should have not returned an error %s", err.Error())
		}
	}
}
func Test_PostNonExistingAlgorithm_TextParams(t *testing.T) {
	f := newFixture(t)
	var params api.ModelsAlgorithmRequestAlgorithmParameters
	_ = json.Unmarshal([]byte("{\"algorithm\": \"non-existing\", \"settingA\": \"a\", \"settingB\": \"b\"}"), &params)

	verifyNonExistingRun(f, params)
}

func Test_PostNonExistingAlgorithm_CodeParams(t *testing.T) {
	f := newFixture(t)
	params := map[string]jx.Raw{
		"algorithm": jx.Raw("\"some-algorithm\""),
		"settingA":  jx.Raw("\"a\""),
		"settingB":  jx.Raw("\"b\""),
	}

	verifyNonExistingRun(f, params)
}

func Test_PostRun(t *testing.T) {
	f := newFixture(t)
	params := map[string]jx.Raw{
		"hello_text":   jx.Raw("\"hello from SDK Go!\""),
		"hello_author": jx.Raw("\"unit tests\""),
	}

	verifyExistingRun(f, params, "hello_test")
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
