package bench

import (
	"os"
	"testing"
)

// TODO: make this a runnable example?
func Test_printHistogramGraph(t *testing.T) {
	type args struct {
		request benchWorkflowRequest
		values  []histogramValue
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "Benchmark const 500",
			args: args{
				request: benchWorkflowRequest{
					Workflow: benchWorkflowRequestWorkflow{
						Name: "basic-workflow",
						Args: struct {
							SequenceCount                int    `json:"sequenceCount"`
							ParallelCount                int    `json:"parallelCount"`
							ActivityDurationMilliseconds int    `json:"activityDurationMilliseconds"`
							Payload                      string `json:"payload"`
							ResultPayload                string `json:"resultPayload"`
						}{
							SequenceCount: 3,
						},
					},
					Steps: []benchWorkflowRequestStep{
						{
							Count:         500,
							RatePerSecond: 50,
							Concurrency:   5,
						},
					},
					Report: benchWorkflowRequestReporting{IntervalInSeconds: 5},
				},
				values: []histogramValue{
					{Started: 64, Execution: 64, Closed: 11, Backlog: 53},
					{Started: 35, Execution: 35, Closed: 63, Backlog: 25},
					{Started: 16, Execution: 16, Closed: 88, Backlog: 53},
					{Started: 10, Execution: 10, Closed: 99, Backlog: 53},
					{Started: 23, Execution: 23, Closed: 87, Backlog: 78},
					{Started: 11, Execution: 11, Closed: 93, Backlog: 96},
					{Started: 41, Execution: 41, Closed: 99, Backlog: 50},
					{Started: 10, Execution: 10, Closed: 60, Backlog: 0},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := printHistogramGraph(tt.args.request, tt.args.values)
			f, err := os.Create("test.html")
			if err != nil {
				t.Fatal(err)
			}
			if _, err := f.WriteString(got); err != nil {
				t.Fatal(err)
			}
		})
	}
}
