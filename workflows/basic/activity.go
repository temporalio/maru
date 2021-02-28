package basic

import (
	"context"
	"time"

	"go.temporal.io/sdk/activity"
)

// BasicActivity is the implementation for Basic Workflow
func BasicActivity(ctx context.Context, activityDelayMilliseconds int) (string, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("BasicActivity: start", "Duration, ms", activityDelayMilliseconds)
	time.Sleep(time.Duration(activityDelayMilliseconds) * time.Millisecond)
	logger.Info("BasicActivity: end")
	return "apple", nil
}
