package basic

import (
	"context"
	"time"

	"go.temporal.io/sdk/activity"
)

// Activity is the implementation for Basic Workflow
func Activity(ctx context.Context, activityDelayMilliseconds int) (string, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Activity: start", "Duration, ms", activityDelayMilliseconds)
	time.Sleep(time.Duration(activityDelayMilliseconds) * time.Millisecond)
	logger.Info("Activity: end")
	return "apple", nil
}
