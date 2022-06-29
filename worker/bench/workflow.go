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

package bench

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"

	"go.temporal.io/sdk/log"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// Workflow represents the main workflow that executes the overall bench test
func Workflow(ctx workflow.Context, request benchWorkflowRequest) error {
	logger := workflow.GetLogger(ctx)
	w := benchWorkflow{
		ctx:      ctx,
		logger:   logger,
		request:  request,
		baseID:   workflow.GetInfo(ctx).WorkflowExecution.ID,
		deadline: workflow.Now(ctx).Add(workflow.GetInfo(ctx).WorkflowExecutionTimeout),
	}
	return w.run()
}

type (
	benchWorkflowRequestStep struct {
		// Count is the total number of workflows to execute across all concurrent drivers.
		Count int `json:"count"`
		// Concurrency defines how many driver activities should be started in parallel.
		Concurrency int `json:"concurrency"`
		// RatePerSecond is the maximum number of workflows to start per second.
		RatePerSecond int `json:"ratePerSecond"`
	}
	benchWorkflowRequestWorkflow struct {
		// Name is the name of the workflow to run for benchmarking (workflow under test).
		Name string `json:"name"`
		// TaskQueue is the name of the task queue to use when executing the workflow.
		TaskQueue string `json:"taskqueue"`
		// Args is the argument that should be the input of all executions of the workflow under test.
		Args interface{} `json:"args"`
	}
	benchWorkflowRequestReporting struct {
		// IntervalInSeconds defines the granularity of the result histogram.
		IntervalInSeconds int `json:"intervalInSeconds"`
		// CsvSeparator defines the separator for the CSV report.
		CsvSeparator string `json:"csvSeparator"`
	}
	benchWorkflowRequest struct {
		Steps    []benchWorkflowRequestStep    `json:"steps"`
		Workflow benchWorkflowRequestWorkflow  `json:"workflow"`
		Report   benchWorkflowRequestReporting `json:"report"`
	}

	histogramValue struct {
		Started   int `json:"started"`
		Execution int `json:"execution"`
		Closed    int `json:"closed"`
		Backlog   int `json:"backlog"`
	}

	metricValue struct {
		Persistence    *int `json:"persistence"`
		HistoryService *int `json:"historyService"`
		PersistenceCpu *int `json:"persistenceCpu"`
		HistoryCpu     *int `json:"historyCpu"`
		HistoryMemory  *float64 `json:"historyMemory"`
	}

	benchWorkflow struct {
		ctx      workflow.Context
		logger   log.Logger
		request  benchWorkflowRequest
		baseID   string
		deadline time.Time
	}
)

func (w *benchWorkflow) run() error {
	w.logger.Info("bench driver workflow started")

	startTime := workflow.Now(w.ctx)

	if len(w.request.Steps) == 0 {
		return errors.New("request must have at least one step defined")
	}

	if w.request.Report.IntervalInSeconds <= 0 {
		w.request.Report.IntervalInSeconds = 60
	}

	for i, step := range w.request.Steps {
		if err := w.executeDriverActivities(i, step); err != nil {
			return err
		}
	}

	res, err := w.executeMonitorActivity(startTime)
	if err != nil {
		return err
	}

	if err = w.setupQueries(res, startTime); err != nil {
		return err
	}

	w.logger.Info("bench driver workflow completed")
	return nil
}

func (w *benchWorkflow) executeDriverActivities(stepIndex int, step benchWorkflowRequestStep) (finalErr error) {
	concurrency := 1
	switch {
	case step.Concurrency > 0:
		concurrency = step.Concurrency
		if step.Count%concurrency != 0 {
			return errors.Errorf("request count %d must be a multiple of concurrency %d", step.Count, concurrency)
		}
	case step.RatePerSecond > 10:
		concurrency = step.RatePerSecond / 10
	}

	var futures []workflow.Future

	for i := 0; i < concurrency; i++ {
		futures = append(futures, workflow.ExecuteActivity(
			w.withActivityOptions(),
			"bench-DriverActivity",
			benchDriverActivityRequest{
				BaseID:        fmt.Sprintf("%s-%d-%d", w.baseID, stepIndex, i),
				BatchSize:     step.Count / concurrency,
				Rate:          step.RatePerSecond / concurrency,
				WorkflowName:  w.request.Workflow.Name,
				TaskQueueName: w.request.Workflow.TaskQueue,
				Parameters:    w.request.Workflow.Args,
			}))
	}

	for i, f := range futures {
		err := f.Get(w.ctx, nil)
		if err != nil {
			w.logger.Warn("failed to execute request", "WorkflowName", w.request.Workflow.Name, "Error", err, "batchID", i)
			finalErr = err
		}
	}

	return finalErr
}

func (w *benchWorkflow) executeMonitorActivity(startTime time.Time) (res []histogramValue, err error) {
	var count int
	for _, step := range w.request.Steps {
		count += step.Count
	}
	err = workflow.ExecuteActivity(
		w.withActivityOptions(),
		"bench-MonitorActivity",
		benchMonitorActivityRequest{
			WorkflowName:      w.request.Workflow.Name,
			StartTime:         startTime,
			BaseID:            w.baseID,
			Count:             count,
			IntervalInSeconds: w.request.Report.IntervalInSeconds,
		}).Get(w.ctx, &res)
	return
}

func (w *benchWorkflow) setupQueries(res []histogramValue, startTime time.Time) error {
	if err := workflow.SetQueryHandler(w.ctx, "histogram", func(input []byte) (string, error) {
		return w.printJson(res), nil
	}); err != nil {
		return err
	}

	if err := workflow.SetQueryHandler(w.ctx, "histogram_csv", func(input []byte) (string, error) {
		return w.printHistogramCsv(res), nil
	}); err != nil {
		return err
	}

	if err := workflow.SetQueryHandler(w.ctx, "metrics", func(input []byte) (string, error) {
		endTime := startTime.Add(time.Duration(w.request.Report.IntervalInSeconds * len(res)) * time.Second)
		values, err := w.collectMetrics(startTime, endTime)
		if err != nil {
			return "", err
		}

		return w.printJson(values), nil
	}); err != nil {
		return err
	}

	if err := workflow.SetQueryHandler(w.ctx, "metrics_csv", func(input []byte) (string, error) {
		endTime := startTime.Add(time.Duration(w.request.Report.IntervalInSeconds * len(res)) * time.Second)
		values, err := w.collectMetrics(startTime, endTime)
		if err != nil {
			return "", err
		}

		return w.printMetricsCsv(values), nil
	}); err != nil {
		return err
	}

	return nil
}

func (w *benchWorkflow) withActivityOptions() workflow.Context {
	ao := workflow.ActivityOptions{
		HeartbeatTimeout:    60 * time.Second,
		StartToCloseTimeout: w.deadline.Sub(workflow.Now(w.ctx)),
		TaskQueue:           benchTaskQueue,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:        50 * time.Millisecond,
			BackoffCoefficient:     1.2,
			NonRetryableErrorTypes: []string{"TestError"},
		},
	}

	return workflow.WithActivityOptions(w.ctx, ao)
}

func (w *benchWorkflow) collectMetrics(startTime, endTime time.Time) ([]metricValue, error) {
	updates, err := w.queryPrometheusHistogram("persistence_latency_bucket{type='history'}", startTime, endTime)
	if err != nil {
		return nil, errors.Wrapf(err, "query UpdateWorkflowExecution")
	}

	appends, err := w.queryPrometheusHistogram("persistence_latency_bucket{type='history'}", startTime, endTime)
	if err != nil {
		return nil, errors.Wrapf(err, "query AppendHistoryNodes")
	}

	services, err := w.queryPrometheusHistogram("service_latency_bucket{type='history'}", startTime, endTime)
	if err != nil {
		return nil, errors.Wrapf(err, "query service latency")
	}

	historyCpus, err := w.queryPrometheusValues("sum(rate(container_cpu_usage_seconds_total{container=\"temporal-history\"}[2m]))", startTime, endTime)
	if err != nil {
		return nil, errors.Wrapf(err, "query history CPU")
	}

	historyMem, err := w.queryPrometheusValues("max(container_memory_working_set_bytes{container=\"temporal-history\"})", startTime, endTime)
	if err != nil {
		return nil, errors.Wrapf(err, "query history memory")
	}

	persistenceCpus, err := w.queryPrometheusValues("sum(rate(container_cpu_usage_seconds_total{container=\"cass-cassandra\"}[2m]))", startTime, endTime)
	if err != nil {
		return nil, errors.Wrapf(err, "query history CPU")
	}

	values := make([]metricValue, len(updates))
	convert := func(f float64) *int {
		if math.IsNaN(f) {
			return nil
		}
		res := int(f*1000)
		return &res
	}
	for i, update := range updates {
		storage := update
		if len(appends) > i {
			storage = math.Max(update, appends[i])
		}
		value := metricValue{Persistence: convert(storage)}
		if len(services) > i {
			value.HistoryService = convert(services[i])
		}
		if len(persistenceCpus) > i {
			value.PersistenceCpu = convert(persistenceCpus[i])
		}
		if len(historyCpus) > i {
			value.HistoryCpu = convert(historyCpus[i])
		}
		if len(historyMem) > i {
			value.HistoryMemory = &historyMem[i]
		}
		values[i] = value
	}
	return values, nil
}

func (w *benchWorkflow) queryPrometheusValues(query string, startTime, endTime time.Time) ([]float64, error) {
	matrix, err := w.queryPrometheus(query, startTime, endTime)
	if err != nil {
		return nil, err
	}

	var res []float64
	for _, sample := range *matrix {
		for _, value := range sample.Values {
			res = append(res, float64(value.Value))
		}
	}
	return res, nil
}

func (w *benchWorkflow) queryPrometheusHistogram(metric string, startTime, endTime time.Time) ([]float64, error) {
	query := fmt.Sprintf("histogram_quantile(0.95,sum(rate(%s[5m])) by (le))", metric)
	matrix, err := w.queryPrometheus(query, startTime, endTime)
	if err != nil {
		return nil, err
	}

	var res []float64
	for _, sample := range *matrix {
		for _, value := range sample.Values {
			res = append(res, float64(value.Value))
		}
	}
	return res, nil
}

func (w *benchWorkflow) queryPrometheus(query string, startTime, endTime time.Time) (*model.Matrix, error) {
	client, err := api.NewClient(api.Config{
		Address: "http://prometheus-server",
	})
	if err != nil {
		return nil, errors.Wrapf(err, "creating API client")
	}
	v1api := v1.NewAPI(client)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	result, _, err := v1api.QueryRange(ctx, query, v1.Range{
		Start: startTime,
		End:   endTime,
		Step:  time.Duration(w.request.Report.IntervalInSeconds) * time.Second,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "query %q", query)
	}
	matrix, ok := result.(model.Matrix)
	if !ok {
		return nil, errors.New("query yielded no results")
	}

	return &matrix, nil
}

func (w *benchWorkflow) printJson(values interface{}) string {
	b, err := json.Marshal(values)
	if err != nil {
		return errors.Wrapf(err, "printing JSON").Error()
	}
	return string(b)
}

func (w *benchWorkflow) printHistogramCsv(values []histogramValue) string {
	separator := ";"
	if w.request.Report.CsvSeparator != "" {
		separator = w.request.Report.CsvSeparator
	}
	interval := w.request.Report.IntervalInSeconds
	header := strings.Join([]string{
		"Time (seconds)",
		"Workflows Started",
		"Workflows Started Rate",
		"Workflows Executions",
		"Workflows Execution Rate",
		"Workflow Closed",
		"Workflow Closed Rate",
		"Backlog",
	}, separator)
	lines := []string{header}
	for i, v := range values {
		line := strings.Join([]string{
			strconv.Itoa((i + 1) * interval),
			strconv.Itoa(v.Started),
			fmt.Sprintf("%f", float32(v.Started)/float32(interval)),
			strconv.Itoa(v.Execution),
			fmt.Sprintf("%f", float32(v.Execution)/float32(interval)),
			strconv.Itoa(v.Closed),
			fmt.Sprintf("%f", float32(v.Closed)/float32(interval)),
			strconv.Itoa(v.Backlog),
		}, separator)
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}

func (w *benchWorkflow) printMetricsCsv(values []metricValue) string {
	separator := ";"
	if w.request.Report.CsvSeparator != "" {
		separator = w.request.Report.CsvSeparator
	}
	interval := w.request.Report.IntervalInSeconds
	header := strings.Join([]string{
		"Time (seconds)",
		"Persistence Latency (ms)",
		"History Service Latency (ms)",
		"Persistence CPU (mcores)",
		"History Service CPU (mcores)",
		"History Service Memory Working Set (MB)",
	}, separator)
	lines := []string{header}
	for i, v := range values {
		var pv string
		if v.Persistence != nil {
			pv = strconv.Itoa(*v.Persistence)
		}
		var hv string
		if v.HistoryService != nil {
			hv = strconv.Itoa(*v.HistoryService)
		}
		var pcpu string
		if v.PersistenceCpu != nil {
			pcpu = strconv.Itoa(*v.PersistenceCpu)
		}
		var hcpu string
		if v.HistoryCpu != nil {
			hcpu = strconv.Itoa(*v.HistoryCpu)
		}
		var hmem string
		if v.HistoryMemory != nil {
			hmem = strconv.Itoa(int(*v.HistoryMemory / 1048576.0))
		}
		line := strings.Join([]string{
			strconv.Itoa((i + 1) * interval),
			pv,
			hv,
			pcpu,
			hcpu,
			hmem,
		}, separator)
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}
