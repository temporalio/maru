package bench

import (
	"fmt"
	"github.com/pkg/errors"
	"strconv"
	"strings"
	"time"

	"go.temporal.io/sdk/log"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// Workflow represents the main workflow that executes the overall bench test
func Workflow(ctx workflow.Context, request benchWorkflowRequest) (*benchWorkflowResponse, error) {
	logger := workflow.GetLogger(ctx)
	w := benchWorkflow{
		ctx:      ctx,
		logger:   logger,
		request:  request,
		baseID:   workflow.GetInfo(ctx).WorkflowExecution.ID,
		deadline: workflow.Now(ctx).Add(workflow.GetInfo(ctx).WorkflowExecutionTimeout),
	}
	return w.run()
}

type (
	benchWorkflowRequestScenario struct {
		// Count is the total number of workflows to execute across all concurrent drivers.
		Count int `json:"count"`
		// Concurrency defines how many driver activities should be started in parallel.
		Concurrency int `json:"concurrency"`
		// RatePerSecond is the maximum number of workflows to start per second.
		RatePerSecond int `json:"ratePerSecond"`
	}
	benchWorkflowRequestWorkflow struct {
		// Name is the name of the workflow to run for benchmarking (workflow under test).
		Name string `json:"name"`
		// Args is the argument that should be the input of all executions of the workflow under test.
		Args interface{} `json:"args"`
	}
	benchWorkflowRequestReporting struct {
		// IntervalInSeconds defines the granularity of the result histogram.
		IntervalInSeconds int `json:"intervalInSeconds"`
		// CsvSeparator defines the separator for the CSV report.
		CsvSeparator string `json:"csvSeparator"`
	}
	benchWorkflowRequest struct {
		Scenario benchWorkflowRequestScenario  `json:"scenario"`
		Workflow benchWorkflowRequestWorkflow  `json:"workflow"`
		Report   benchWorkflowRequestReporting `json:"report"`
	}

	histogramOutputs struct {
		Json []histogramValue
		Csv  string
	}

	benchWorkflowResponse struct {
		Histogram histogramOutputs
	}

	histogramValue struct {
		Started int
		Closed  int
		Backlog int
	}

	benchWorkflow struct {
		ctx      workflow.Context
		logger   log.Logger
		request  benchWorkflowRequest
		baseID   string
		deadline time.Time
	}
)

func (w *benchWorkflow) run() (*benchWorkflowResponse, error) {
	w.logger.Info("bench driver workflow started")

	startTime := workflow.Now(w.ctx)

	if w.request.Scenario.Count <= 0 {
		return nil, errors.Errorf("request count %d must be a positive number", w.request.Scenario.Count)
	}

	concurrency := 1
	if w.request.Scenario.Concurrency > 0 {
		if w.request.Scenario.Count%concurrency != 0 {
			return nil, errors.Errorf("request count %d must be a multiple of concurrency %d", w.request.Scenario.Count, concurrency)
		}
	}

	if w.request.Report.IntervalInSeconds <= 0 {
		w.request.Report.IntervalInSeconds = 60
	}

	err := w.executeDriverActivities(concurrency)
	if err != nil {
		return nil, err
	}

	res, err := w.executeMonitorActivity(startTime)
	if err != nil {
		return nil, err
	}

	w.logger.Info("bench driver workflow completed")
	return &benchWorkflowResponse{
		Histogram: histogramOutputs{
			Json: res,
			Csv:  w.printCsv(res),
		},
	}, nil
}

func (w *benchWorkflow) executeDriverActivities(concurrency int) (finalErr error) {
	var futures []workflow.Future

	for i := 0; i < concurrency; i++ {
		futures = append(futures, workflow.ExecuteActivity(
			w.withActivityOptions(),
			"bench-driver-activity",
			benchDriverActivityRequest{
				BaseID:       fmt.Sprintf("%s-%d", w.baseID, i),
				BatchSize:    w.request.Scenario.Count / concurrency,
				Rate:         w.request.Scenario.RatePerSecond / concurrency,
				WorkflowName: w.request.Workflow.Name,
				Parameters:   w.request.Workflow.Args,
			}))
	}

	for i, f := range futures {
		err := f.Get(w.ctx, nil)
		if err != nil {
			w.logger.Warn("failed to execute request", "WorkflowName", w.request.Workflow.Name, "Error", err, "batchID", i)
			finalErr = err
		}
	}

	return finalErr
}

func (w *benchWorkflow) executeMonitorActivity(startTime time.Time) (res []histogramValue, err error) {
	err = workflow.ExecuteActivity(
		w.withActivityOptions(),
		"bench-monitor-activity",
		benchMonitorActivityRequest{
			WorkflowName:      w.request.Workflow.Name,
			StartTime:         startTime,
			BaseID:            w.baseID,
			Count:             w.request.Scenario.Count,
			IntervalInSeconds: w.request.Report.IntervalInSeconds,
		}).Get(w.ctx, &res)
	return
}

func (w *benchWorkflow) withActivityOptions() workflow.Context {
	ao := workflow.ActivityOptions{
		HeartbeatTimeout:    60 * time.Second,
		StartToCloseTimeout: w.deadline.Sub(workflow.Now(w.ctx)),
		TaskQueue:           taskQueue,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:        50 * time.Millisecond,
			BackoffCoefficient:     1.2,
			NonRetryableErrorTypes: []string{"TestError"},
		},
	}

	return workflow.WithActivityOptions(w.ctx, ao)
}

func (w *benchWorkflow) printCsv(values []histogramValue) string {
	separator := ";"
	if w.request.Report.CsvSeparator != "" {
		separator = w.request.Report.CsvSeparator
	}
	interval := w.request.Report.IntervalInSeconds
	header := strings.Join([]string{
		"Time (seconds)",
		"Workflows Started",
		"Workflows Started Rate",
		"Workflow Closed",
		"Workflow Closed Rate",
		"Backlog",
	}, separator)
	lines := []string { header }
	for i, v := range values {
		line := strings.Join([]string{
			strconv.Itoa((i + 1) * interval),
			strconv.Itoa(v.Started),
			fmt.Sprintf("%f", float32(v.Started)/float32(interval)),
			strconv.Itoa(v.Closed),
			fmt.Sprintf("%f", float32(v.Closed)/float32(interval)),
			strconv.Itoa(v.Backlog),
		}, separator)
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}
