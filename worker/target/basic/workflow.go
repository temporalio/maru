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
