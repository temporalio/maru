package basic

import (
	"context"
	"time"

	"go.temporal.io/sdk/activity"
)

type basicActivityRequest struct {
	ActivityDelayMilliseconds int
	Payload                   string
	ResultPayload             string
}

// Activity is the implementation for Basic Workflow
func Activity(ctx context.Context, req basicActivityRequest) (string, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Activity: start", "Duration, ms", req.ActivityDelayMilliseconds)
	time.Sleep(time.Duration(req.ActivityDelayMilliseconds) * time.Millisecond)
	logger.Info("Activity: end")
	return req.ResultPayload, nil
}
