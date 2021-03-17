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

package basic

import (
	"time"

	"go.temporal.io/sdk/workflow"
)

// WorkflowRequest is used for starting workflow for Basic bench workflow
type workflowRequest struct {
	SequenceCount                int    `json:"sequenceCount"`
	ActivityDurationMilliseconds int    `json:"activityDurationMilliseconds"`
	Payload                      string `json:"payload"`
	ResultPayload                string `json:"resultPayload"`
}

const taskQueue = "temporal-bench"

// Workflow implements a basic bench scenario to schedule activities in sequence.
func Workflow(ctx workflow.Context, request workflowRequest) (string, error) {

	logger := workflow.GetLogger(ctx)

	logger.Info("basic workflow started", "activity task queue", taskQueue)

	ao := workflow.ActivityOptions{
		TaskQueue:           taskQueue,
		StartToCloseTimeout: time.Duration(request.ActivityDurationMilliseconds)*time.Millisecond + time.Hour,
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	for i := 0; i < request.SequenceCount; i++ {
		var result string
		req := basicActivityRequest{
			ActivityDelayMilliseconds: request.ActivityDurationMilliseconds,
			Payload:                   request.Payload,
			ResultPayload:             request.ResultPayload,
		}
		err := workflow.ExecuteActivity(ctx, "basic-activity", req).Get(ctx, &result)
		if err != nil {
			return "", err
		}
		logger.Info("activity returned result to the workflow", "value", result)
	}

	logger.Info("basic workflow completed")
	return request.ResultPayload, nil
}
