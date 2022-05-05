// The MIT License
//
// Copyright (c) 2021 Temporal Technologies Inc.  All rights reserved.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package bench

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/log"
	"golang.org/x/time/rate"
)

func (a *Activities) DriverActivity(ctx context.Context, request benchDriverActivityRequest) error {
	logger := activity.GetLogger(ctx)
	driver := benchDriver{
		logger:  logger,
		client:  a.temporalClient,
		request: request,
	}
	return driver.run(ctx)
}

type (
	benchDriverActivityRequest struct {
		WorkflowName string
		BaseID       string
		BatchSize    int
		Rate         int
		Parameters   interface{}
	}
	benchDriver struct {
		logger  log.Logger
		client  client.Client
		request benchDriverActivityRequest
	}
)

const defaultWorkflowTaskStartToCloseTimeoutDuration = 10 * time.Second

func (d *benchDriver) run(ctx context.Context) error {
	idx := 0
	deadline := activity.GetInfo(ctx).Deadline.Add(-2 * time.Second)
	if activity.HasHeartbeatDetails(ctx) {
		// we are retrying from an activity timeout, and there is reported progress that we should resume from.
		var completedIdx int
		if err := activity.GetHeartbeatDetails(ctx, &completedIdx); err == nil {
			idx = completedIdx + 1
			d.logger.Info("resuming from failed attempt", "ReportedProgress", completedIdx)
		}
	}

	limit := rate.Inf
	if d.request.Rate > 0 {
		limit = rate.Every(time.Second / time.Duration(d.request.Rate))
	}
	limiter := rate.NewLimiter(limit, 1)
	for i := idx; i < d.request.BatchSize; i++ {
		if err := limiter.Wait(ctx); err != nil {
			return errors.Wrapf(err, "waiting for limiter")
		}

		if err := d.execute(ctx, i); err != nil {
			d.logger.Error("driver failed to execute", "Error", err, "ID", i)
			return err
		}

		activity.RecordHeartbeat(ctx, i)

		if time.Now().After(deadline) {
			return &TestError{
				Message: fmt.Sprintf("Timed out driving bench test activity. Progress: %v out of %v",
					i, d.request.BatchSize),
			}
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("driver activity context finished: %+v", ctx.Err())
		default:
		}
	}

	return nil
}

func (d *benchDriver) execute(ctx context.Context, iterationID int) error {
	d.logger.Info("driver.execute starting", "workflowName", d.request.WorkflowName, "basedID", d.request.BaseID, "iterationID", iterationID)
	workflowID := fmt.Sprintf("%s-%s-%d", d.request.WorkflowName, d.request.BaseID, iterationID)
	startOptions := client.StartWorkflowOptions{
		ID:                       workflowID,
		TaskQueue:                targetTaskQueue,
		WorkflowExecutionTimeout: 30 * time.Minute,
		WorkflowTaskTimeout:      defaultWorkflowTaskStartToCloseTimeoutDuration,
	}
	_, err := d.client.ExecuteWorkflow(ctx, startOptions, d.request.WorkflowName, buildPayload(d.request.Parameters))
	if err != nil {
		d.logger.Error("failed to start workflow", "Error", err, "ID", workflowID)
	}

	d.logger.Info("driver.execute completed", "workflowName", d.request.WorkflowName, "basedID", d.request.BaseID, "iterationID", iterationID)
	return err
}
