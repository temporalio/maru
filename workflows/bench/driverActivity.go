package bench

import (
	"context"
	"fmt"
	"github.com/mikhailshilkov/temporal-bench/common"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/log"
	"time"
)

func DriverActivity(ctx context.Context, request benchDriverActivityRequest) error {
	logger := activity.GetLogger(ctx)
	temporalClient, err := common.GetTemporalClientFromContext(ctx)
	if err != nil {
		return err
	}

	driver := benchDriver{
		ctx:     ctx,
		logger:  logger,
		client:  temporalClient,
		request: request,
	}
	return driver.run()
}

type (
	benchDriverActivityRequest struct {
		WorkflowName string
		BaseID    string
		BatchSize int
		Rate      int
		Parameters   interface{}
	}
	benchDriver struct {
		ctx     context.Context
		logger  log.Logger
		client  client.Client
		request benchDriverActivityRequest
	}
)

func (d *benchDriver) run() error {
	idx := 0
	deadline := activity.GetInfo(d.ctx).Deadline.Add(-2 * time.Second)
	if activity.HasHeartbeatDetails(d.ctx) {
		// we are retrying from an activity timeout, and there is reported progress that we should resume from.
		var completedIdx int
		if err := activity.GetHeartbeatDetails(d.ctx, &completedIdx); err == nil {
			idx = completedIdx + 1
			d.logger.Info("resuming from failed attempt", "ReportedProgress", completedIdx)
		}
	}

	startTime := time.Now()
	for i := idx; i < d.request.BatchSize; i++ {
		if d.request.Rate > 0 {
			now := time.Now()
			targetTime := startTime.Add(time.Millisecond * time.Duration(1000*(i-idx)/d.request.Rate))
			if now.Before(targetTime) {
				time.Sleep(targetTime.Sub(now))
			}
		}

		if err := d.execute(i); err != nil {
			d.logger.Error("driver failed to execute", "Error", err, "ID", i)
			return err
		}

		activity.RecordHeartbeat(d.ctx, i)

		if time.Now().After(deadline) {
			return &BenchTestError{
				Message: fmt.Sprintf("Timed out driving bench test activity. Progress: %v out of %v",
					i, d.request.BatchSize),
			}
		}

		select {
		case <-d.ctx.Done():
			return fmt.Errorf("driver activity context finished: %+v", d.ctx.Err())
		default:
		}
	}

	return nil
}

func (d *benchDriver) execute(iterationID int) error {
	d.logger.Info("driver.execute starting", "workflowName", d.request.WorkflowName, "basedID", d.request.BaseID, "iterationID", iterationID)
	workflowID := fmt.Sprintf("%s-%s-%d", d.request.WorkflowName, d.request.BaseID, iterationID)
	startOptions := client.StartWorkflowOptions{
		ID:                       workflowID,
		TaskQueue:                taskQueue,
		WorkflowExecutionTimeout: 168 * time.Hour,
		WorkflowTaskTimeout:      defaultWorkflowTaskStartToCloseTimeoutDuration,
	}
	_, err := d.client.ExecuteWorkflow(d.ctx, startOptions, d.request.WorkflowName, d.request.Parameters)
	if err != nil {
		d.logger.Error("failed to start workflow", "Error", err, "ID", workflowID)
	}

	d.logger.Info("driver.execute completed", "workflowName", d.request.WorkflowName, "basedID", d.request.BaseID, "iterationID", iterationID)
	return err
}
