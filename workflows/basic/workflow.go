package basic

import (
	"time"

	"go.temporal.io/sdk/workflow"

	"github.com/mikhailshilkov/temporal-bench/common"
)

// WorkflowRequest is used for starting workflow for Basic bench workflow
type workflowRequest struct {
	SequenceCount                int
	ActivityDurationMilliseconds int
}

// BasicWorkflow implements a basic bench scenario to schedule activities in sequence and parallel
func BasicWorkflow(ctx workflow.Context, request workflowRequest) error {

	logger := workflow.GetLogger(ctx)

	logger.Info("basic workflow started", "activity task queue", common.TaskQueue)

	ao := workflow.ActivityOptions{
		TaskQueue:           common.TaskQueue,
		StartToCloseTimeout: time.Duration(request.ActivityDurationMilliseconds)*time.Millisecond + time.Hour,
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	for i := 0; i < request.SequenceCount; i++ {
		var result string
		err := workflow.ExecuteActivity(ctx, "basic-activity", request.ActivityDurationMilliseconds).Get(ctx, &result)
		if err != nil {
			return err
		}
		logger.Info("activity returned result to the workflow", "value", result)
	}

	logger.Info("basic workflow completed")
	return nil
}
