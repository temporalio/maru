package bench

import (
	"fmt"
	"time"

	"go.temporal.io/sdk/log"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// BenchWorkflow represents the main workflow that executes the overall bench test
func BenchWorkflow(ctx workflow.Context, request benchWorkflowRequest) (*benchWorkflowResponse, error) {
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
	benchWorkflowRequest struct {
		Concurrency  int
		Count        int
		Rate         int
		WorkflowName string
		Parameters   interface{}
	}

	benchWorkflowResponse struct {
		Histogram []histogramValue
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

	err := w.executeDriverActivities()
	if err != nil {
		return nil, err
	}

	res, err := w.executeMonitorActivity(startTime)
	if err != nil {
		return nil, err
	}

	w.logger.Info("bench driver workflow completed")
	return res, nil
}

func (w *benchWorkflow) executeDriverActivities() (finalErr error) {
	var futures []workflow.Future

	for i := 0; i < w.request.Concurrency; i++ {
		futures = append(futures, workflow.ExecuteActivity(
			w.withActivityOptions(),
			"bench-driver-activity",
			benchDriverActivityRequest{
				WorkflowName: w.request.WorkflowName,
				BaseID:       fmt.Sprintf("%s-%d", w.baseID, i),
				BatchSize:    w.request.Count,
				Rate:         w.request.Rate,
				Parameters:   w.request.Parameters,
			}))
	}

	for i, f := range futures {
		err := f.Get(w.ctx, nil)
		if err != nil {
			w.logger.Warn("failed to execute request", "WorkflowName", w.request.WorkflowName, "Error", err, "batchID", i)
			finalErr = err
		}
	}

	return finalErr
}

func (w *benchWorkflow) executeMonitorActivity(startTime time.Time) (res *benchWorkflowResponse, err error) {
	err = workflow.ExecuteActivity(
		w.withActivityOptions(),
		"bench-monitor-activity",
		benchMonitorActivityRequest{
			WorkflowName: w.request.WorkflowName,
			StartTime:    startTime,
			BaseID:       w.baseID,
			Count:        w.request.Concurrency * w.request.Count,
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
			NonRetryableErrorTypes: []string{"BenchTestError"},
		},
	}

	return workflow.WithActivityOptions(w.ctx, ao)
}
