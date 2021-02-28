package bench

import (
	"time"
)

const (
	// taskQueue is the queue used by worker to pull workflow and activity tasks
	taskQueue = "temporal-bench"
	defaultWorkflowTaskStartToCloseTimeoutDuration = 10 * time.Second
)

// BenchTestError represents an error that should abort / fail the whole bench test
type BenchTestError struct {
	Message string
}

func (b *BenchTestError) Error() string {
	return b.Message
}
