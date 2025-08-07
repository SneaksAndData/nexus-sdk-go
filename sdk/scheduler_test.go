package sdk

import (
	"context"
	"errors"
	"fmt"
	"github.com/SneaksAndData/nexus-core/pkg/telemetry"
	receiverapi "github.com/SneaksAndData/nexus-sdk-go/pkg/generated/receiver"
	schedulerapi "github.com/SneaksAndData/nexus-sdk-go/pkg/generated/scheduler"
	"github.com/go-faster/jx"
	"github.com/google/uuid"
	"github.com/ogen-go/ogen/json"
	"k8s.io/klog/v2"
	"os"
	"strings"
	"testing"
	"time"
)

var helloParams = map[string]jx.Raw{
	"hello_text":   jx.Raw("\"hello from SDK Go!\""),
	"hello_author": jx.Raw("\"unit tests\""),
}

type fixture struct {
	t              *testing.T
	url            string
	receiverUrl    string
	logger         *klog.Logger
	client         *NexusSchedulerClient
	receiverClient *NexusReceiverClient
}

func newFixture(t *testing.T) *fixture {
	f := &fixture{
		url:         os.Getenv("NEXUS_TEST_SCHEDULER_URL"),
		receiverUrl: os.Getenv("NEXUS_TEST_RECEIVER_URL"),
		t:           t,
	}
	appLogger, _ := telemetry.ConfigureLogger(context.TODO(), map[string]string{}, "info")
	klog.SetSlogLogger(appLogger)

	logger := klog.FromContext(context.TODO())

	f.logger = &logger
	f.client = NewNexusSchedulerClient(f.url, f.logger, nil, nil)
	f.receiverClient = NewNexusReceiverClient(f.receiverUrl, f.logger, nil, nil)

	return f
}

func verifyNonExistingRun(testFixture *fixture, params schedulerapi.ModelsAlgorithmRequestAlgorithmParameters) {
	request := &schedulerapi.ModelsAlgorithmRequest{
		AlgorithmParameters: params,
		CustomConfiguration: schedulerapi.OptV1NexusAlgorithmSpec{
			Set: false,
		},
		ParentRequest: schedulerapi.OptModelsAlgorithmRequestRef{
			Set: false,
		},
		PayloadValidFor: schedulerapi.OptString{
			Set: false,
		},
		RequestApiVersion: schedulerapi.OptString{
			Set: false,
		},
		Tag: schedulerapi.OptString{
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

func verifyExistingRun(testFixture *fixture, params schedulerapi.ModelsAlgorithmRequestAlgorithmParameters, runTag string) {
	request := &schedulerapi.ModelsAlgorithmRequest{
		AlgorithmParameters: params,
		CustomConfiguration: schedulerapi.OptV1NexusAlgorithmSpec{
			Set: false,
		},
		ParentRequest: schedulerapi.OptModelsAlgorithmRequestRef{
			Set: false,
		},
		PayloadValidFor: schedulerapi.OptString{
			Set: false,
		},
		RequestApiVersion: schedulerapi.OptString{
			Set: false,
		},
		Tag: schedulerapi.OptString{
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
	var params schedulerapi.ModelsAlgorithmRequestAlgorithmParameters
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
	verifyExistingRun(f, helloParams, "hello_test")
}

func Test_GetRunResultsTagExists(t *testing.T) {
	f := newFixture(t)
	expectedLength := 3
	actualLength := 0
	tag := uuid.New()

	for i := 0; i < expectedLength; i++ {
		verifyExistingRun(f, helloParams, tag.String())
	}

	time.Sleep(1 * time.Second)

	for _, err := range f.client.GetRunResults(tag.String(), nil) {
		if err != nil {
			f.t.Error(err)
		}
		actualLength++
	}

	if actualLength != expectedLength {
		f.t.Errorf("GetRunResults should have returned the expected %d submissions", expectedLength)
	}
}

func Test_AwaitRun(t *testing.T) {
	f := newFixture(t)
	tag := uuid.New()
	runId, err := f.client.CreateRun(&schedulerapi.ModelsAlgorithmRequest{
		AlgorithmParameters: helloParams,
		CustomConfiguration: schedulerapi.OptV1NexusAlgorithmSpec{
			Set: false,
		},
		ParentRequest: schedulerapi.OptModelsAlgorithmRequestRef{
			Set: false,
		},
		PayloadValidFor: schedulerapi.OptString{
			Set: false,
		},
		RequestApiVersion: schedulerapi.OptString{
			Set: false,
		},
		Tag: schedulerapi.OptString{
			Value: tag.String(),
			Set:   true,
		},
	}, "hello-world")

	if err != nil {
		f.t.Error(err)
	}

	time.Sleep(1 * time.Second)

	_, err = f.client.AwaitRun(runId, "hello-world", nil)

	if err != nil {
		f.t.Error(err)
	}
}

func Test_AwaitRuns(t *testing.T) {
	f := newFixture(t)
	tags := []string{}
	for i := 0; i < 10; i++ {
		tag := uuid.New()
		tags = append(tags, tag.String())
		_, err := f.client.CreateRun(&schedulerapi.ModelsAlgorithmRequest{
			AlgorithmParameters: helloParams,
			CustomConfiguration: schedulerapi.OptV1NexusAlgorithmSpec{
				Set: false,
			},
			ParentRequest: schedulerapi.OptModelsAlgorithmRequestRef{
				Set: false,
			},
			PayloadValidFor: schedulerapi.OptString{
				Set: false,
			},
			RequestApiVersion: schedulerapi.OptString{
				Set: false,
			},
			Tag: schedulerapi.OptString{
				Value: tag.String(),
				Set:   true,
			},
		}, "hello-world")

		if err != nil {
			f.t.Error(err)
		}
	}

	// make sure runs have been committed
	time.Sleep(1 * time.Second)

	var counterRef *chan int32
	counter := make(chan int32, 10)
	counterRef = &counter
	go func() {
		print("Completed run")
	}()

	runs, err := f.client.AwaitTaggedRuns(tags, nil, nil, counterRef)

	if err != nil {
		f.t.Error(err)
	}

	for run := range runs {
		if run.Status.Value != "DEADLINE_EXCEEDED" {
			f.t.Error(errors.New("this algorithm is expected to fail"))
		}
	}
}

func Test_GetRunMetadata(t *testing.T) {
	f := newFixture(t)
	tag := uuid.New()
	runId, err := f.client.CreateRun(&schedulerapi.ModelsAlgorithmRequest{
		AlgorithmParameters: helloParams,
		CustomConfiguration: schedulerapi.OptV1NexusAlgorithmSpec{
			Set: false,
		},
		ParentRequest: schedulerapi.OptModelsAlgorithmRequestRef{
			Set: false,
		},
		PayloadValidFor: schedulerapi.OptString{
			Set: false,
		},
		RequestApiVersion: schedulerapi.OptString{
			Set: false,
		},
		Tag: schedulerapi.OptString{
			Value: tag.String(),
			Set:   true,
		},
	}, "hello-world")

	if err != nil {
		f.t.Error(err)
	}

	time.Sleep(1 * time.Second)

	metadata, err := f.client.GetMetadata(runId, "hello-world")

	if err != nil {
		f.t.Error(err)
	}

	if metadata == nil {
		f.t.Error(errors.New("run metadata should not be nil"))
	}

	if metadata != nil && metadata.PayloadURI.Set == false {
		f.t.Error(errors.New("run metadata should not be nil"))
	}
}

func Test_GetRun(t *testing.T) {
	f := newFixture(t)
	tag := uuid.New()
	runId, err := f.client.CreateRun(&schedulerapi.ModelsAlgorithmRequest{
		AlgorithmParameters: helloParams,
		CustomConfiguration: schedulerapi.OptV1NexusAlgorithmSpec{
			Set: false,
		},
		ParentRequest: schedulerapi.OptModelsAlgorithmRequestRef{
			Set: false,
		},
		PayloadValidFor: schedulerapi.OptString{
			Set: false,
		},
		RequestApiVersion: schedulerapi.OptString{
			Set: false,
		},
		Tag: schedulerapi.OptString{
			Value: tag.String(),
			Set:   true,
		},
	}, "hello-world")

	if err != nil {
		f.t.Error(err)
	}

	time.Sleep(1 * time.Second)

	if _, err = f.client.AwaitRun(runId, "hello-world", nil); err != nil {
		f.t.Error(err)
	}

	result, err := f.client.GetRun(runId, "hello-world")

	if err != nil {
		f.t.Error(err)
	}

	if result == nil {
		f.t.Error(errors.New("run result should not be nil"))
	}

	if result != nil && result.Status.Value != "FAILED" {
		f.t.Error(errors.New("run result should have status FAILED"))
	}
}

func Test_GetRunResults(t *testing.T) {
	f := newFixture(t)
	tag := uuid.New()
	run1Id, err := f.client.CreateRun(&schedulerapi.ModelsAlgorithmRequest{
		AlgorithmParameters: helloParams,
		CustomConfiguration: schedulerapi.OptV1NexusAlgorithmSpec{
			Set: false,
		},
		ParentRequest: schedulerapi.OptModelsAlgorithmRequestRef{
			Set: false,
		},
		PayloadValidFor: schedulerapi.OptString{
			Set: false,
		},
		RequestApiVersion: schedulerapi.OptString{
			Set: false,
		},
		Tag: schedulerapi.OptString{
			Value: tag.String(),
			Set:   true,
		},
	}, "hello-world")

	run2Id, err := f.client.CreateRun(&schedulerapi.ModelsAlgorithmRequest{
		AlgorithmParameters: helloParams,
		CustomConfiguration: schedulerapi.OptV1NexusAlgorithmSpec{
			Set: false,
		},
		ParentRequest: schedulerapi.OptModelsAlgorithmRequestRef{
			Set: false,
		},
		PayloadValidFor: schedulerapi.OptString{
			Set: false,
		},
		RequestApiVersion: schedulerapi.OptString{
			Set: false,
		},
		Tag: schedulerapi.OptString{
			Value: tag.String(),
			Set:   true,
		},
	}, "hello-world")

	algorithmName := "hello-world"
	runs := []string{run1Id, run2Id}

	if err != nil {
		f.t.Error(err)
	}

	time.Sleep(1 * time.Second)

	if _, err = f.client.AwaitTaggedRuns([]string{tag.String()}, &algorithmName, nil, nil); err != nil {
		f.t.Error(err)
	}

	resultCount := 0

	for runResult := range f.client.GetRunResults(tag.String(), &algorithmName) {
		if runResult.Status.Value != "FAILED" {
			f.t.Error(errors.New("run result should have status FAILED"))
		}
		resultCount++
	}

	if resultCount != len(runs) {
		f.t.Error(errors.New(fmt.Sprintf("expected to find %d runs, but found %d", len(runs), resultCount)))
	}
}

func Test_CompleteRun(t *testing.T) {
	f := newFixture(t)
	tag := uuid.New()
	runId, err := f.client.CreateRun(&schedulerapi.ModelsAlgorithmRequest{
		AlgorithmParameters: helloParams,
		CustomConfiguration: schedulerapi.OptV1NexusAlgorithmSpec{
			Set: false,
		},
		ParentRequest: schedulerapi.OptModelsAlgorithmRequestRef{
			Set: false,
		},
		PayloadValidFor: schedulerapi.OptString{
			Set: false,
		},
		RequestApiVersion: schedulerapi.OptString{
			Set: false,
		},
		Tag: schedulerapi.OptString{
			Value: tag.String(),
			Set:   true,
		},
	}, "hello-world")

	if err != nil {
		f.t.Error(err)
	}

	time.Sleep(1 * time.Second)

	if _, err = f.client.AwaitRun(runId, "hello-world", nil); err != nil {
		f.t.Error(err)
	}

	result := &receiverapi.ModelsAlgorithmResult{
		ErrorCause: receiverapi.OptString{
			Value: "Fail cause overridden",
			Set:   true,
		},
		ErrorDetails: receiverapi.OptString{
			Value: "Algorithm had its cause updated",
			Set:   true,
		},
		ResultUri: receiverapi.OptString{
			Set: false,
		},
	}

	if err := f.receiverClient.CompleteRequest(result, "hello-world", runId); err != nil {
		f.t.Error(err)
	}
}
