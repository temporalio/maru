package bench

import "go.temporal.io/sdk/client"

// Activities is a structure with bench activity functions.
type Activities struct {
	temporalClient client.Client
}

// NewActivities creates a new structure with bench activity functions.
func NewActivities(temporalClient client.Client) *Activities {
	return &Activities{temporalClient: temporalClient}
}

// taskQueue is the queue used by worker to pull workflow and activity tasks
const taskQueue = "temporal-bench"

// TestError represents an error that should abort / fail the whole bench test
type TestError struct {
	Message string
}

func (b *TestError) Error() string {
	return b.Message
}
