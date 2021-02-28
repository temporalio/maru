package common

import (
	"context"
	"errors"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
)

const (
	// TaskQueue is the queue used by worker to pull workflow and activity tasks
	TaskQueue = "temporal-bench"
)

// ClientKey is the key for lookup
type ClientKey int

const (
	// TemporalClientKey for retrieving temporal client from context
	TemporalClientKey ClientKey = iota
)

var (
	// ErrTemporalClientNotFound when temporal client is not found on context
	ErrTemporalClientNotFound = errors.New("failed to retrieve temporal client from context")
)

func GetTemporalClientFromContext(ctx context.Context) (client.Client, error) {
	logger := activity.GetLogger(ctx)
	temporalClient := ctx.Value(TemporalClientKey).(client.Client)
	if temporalClient == nil {
		logger.Error("Could not retrieve temporal client from context.")
		return nil, ErrTemporalClientNotFound
	}

	return temporalClient, nil
}
