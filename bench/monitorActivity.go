package bench

import (
	"context"
	"fmt"
	"github.com/mikhailshilkov/temporal-bench/common"
	"go.temporal.io/api/filter/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/log"
	"strings"
	"time"
)

func MonitorActivity(ctx context.Context, request benchMonitorActivityRequest) ([]histogramValue, error) {
	logger := activity.GetLogger(ctx)
	temporalClient, err := common.GetTemporalClientFromContext(ctx)
	if err != nil {
		return nil, err
	}

	m := benchMonitor{
		ctx:     ctx,
		logger:  logger,
		client:  temporalClient,
		request: request,
	}
	return m.run()
}

type (
	benchMonitorActivityRequest struct {
		BaseID            string
		WorkflowName      string
		Count             int
		StartTime         time.Time
		IntervalInSeconds int
	}
	workflowTiming struct {
		StartTime     time.Time
		ExecutionTime time.Time
		CloseTime     time.Time
	}

	benchMonitor struct {
		ctx     context.Context
		logger  log.Logger
		client  client.Client
		request benchMonitorActivityRequest
	}
)

func (m *benchMonitor) run() ([]histogramValue, error) {
	startTime := activity.GetInfo(m.ctx).StartedTime
	deadline := activity.GetInfo(m.ctx).Deadline.Add(time.Second * -5)

	stats, err := m.validateScenarioCompletion(deadline)
	if err != nil {
		return nil, err
	}

	hist := m.calculateHistogram(stats)

	m.logger.Info("!!! BENCH TEST COMPLETED !!!", "duration", time.Now().Sub(startTime))
	return hist, nil
}

func (m *benchMonitor) validateScenarioCompletion(deadline time.Time) ([]workflowTiming, error) {
	waitStartTime := activity.GetInfo(m.ctx).StartedTime
	for {
		complete, err := m.isComplete()
		if err != nil {
			m.logger.Info("bench monitor failure", "error", err)
			return nil, err
		}

		totalWaitDuration := time.Now().Sub(waitStartTime)

		if complete {
			stats := m.collectWorkflowTimings()

			if len(stats) < m.request.Count {
				m.logger.Warn("no open workflows but fewer closed workflows than expected",
					"expected", m.request.Count,
					"actual", len(stats),
				)
			} else {
				return stats, nil
			}
		}

		m.logger.Info("still waiting for bench test completion",
			"duration", totalWaitDuration,
			"deadline", deadline,
		)
		activity.RecordHeartbeat(
			m.ctx,
			fmt.Sprintf("test scenario - duration: %v, deadline: %v", totalWaitDuration, deadline))

		time.Sleep(time.Second * 3)

		select {
		case <-m.ctx.Done():
			return nil, fmt.Errorf("monitor activity context finished %+v", m.ctx.Err())
		default:
		}

		if time.Now().After(deadline) {
			return nil, &TestError{
				Message: "timed out waiting for Monitoring phase to finish",
			}
		}
	}
}

func (m *benchMonitor) isComplete() (bool, error) {
	m.logger.Info("IsComplete? enter")
	filterStartTime := m.request.StartTime.Add(-10 * time.Second)
	ws, err := m.client.ListOpenWorkflow(m.ctx, &workflowservice.ListOpenWorkflowExecutionsRequest{
		MaximumPageSize: 1,
		Filters: &workflowservice.ListOpenWorkflowExecutionsRequest_TypeFilter{
			TypeFilter: &filter.WorkflowTypeFilter{
				Name: m.request.WorkflowName,
			},
		},
		StartTimeFilter: &filter.StartTimeFilter{
			EarliestTime: &filterStartTime,
		},
	})
	if err != nil {
		m.logger.Info("IsComplete? exit", "error", err)
		return false, err
	}
	done := len(ws.Executions) == 0
	m.logger.Info(fmt.Sprintf("IsComplete? %t", done))
	return done, nil
}

func (m *benchMonitor) collectWorkflowTimings() []workflowTiming {
	var stats []workflowTiming
	filterStartTime := m.request.StartTime.Add(-10 * time.Second)
	prefix := fmt.Sprintf("%s-%s-", m.request.WorkflowName, m.request.BaseID)
	var nextPageToken []byte
	for {
		ws, err := m.client.ListClosedWorkflow(m.ctx, &workflowservice.ListClosedWorkflowExecutionsRequest{
			MaximumPageSize: 1000,
			StartTimeFilter: &filter.StartTimeFilter{
				EarliestTime: &filterStartTime,
			},
			Filters: &workflowservice.ListClosedWorkflowExecutionsRequest_TypeFilter{
				TypeFilter: &filter.WorkflowTypeFilter{
					Name: m.request.WorkflowName,
				},
			},
			NextPageToken: nextPageToken,
		})
		if err != nil {
			m.logger.Info("Stats? exit", "error", err)
			return nil
		}

		for _, w := range ws.Executions {
			if strings.HasPrefix(w.Execution.WorkflowId, prefix) {
				stats = append(stats, workflowTiming{
					StartTime:     *w.StartTime,
					ExecutionTime: *w.ExecutionTime,
					CloseTime:     *w.CloseTime,
				})
			}
		}

		if len(ws.NextPageToken) == 0 {
			break
		}
		nextPageToken = ws.NextPageToken
		activity.RecordHeartbeat(m.ctx, len(stats))
	}
	return stats
}

func (m *benchMonitor) calculateHistogram(stats []workflowTiming) []histogramValue {
	startTime := time.Now().AddDate(0, 0, 1)
	endTime := time.Now().AddDate(0, 0, -1)
	for _, s := range stats {
		if startTime.After(s.StartTime) {
			startTime = s.StartTime
		}
		if endTime.Before(s.CloseTime) {
			endTime = s.CloseTime
		}
	}
	interval := m.request.IntervalInSeconds
	count := int(endTime.Sub(startTime).Seconds())/interval + 1
	hist := make([]histogramValue, count)
	for _, s := range stats {
		si := int(s.StartTime.Sub(startTime).Seconds()) / interval
		hist[si].Started += 1
		ei := int(s.ExecutionTime.Sub(startTime).Seconds()) / interval
		hist[ei].Execution += 1
		ci := int(s.CloseTime.Sub(startTime).Seconds()) / interval
		hist[ci].Closed += 1
		for i := si; i < ci; i++ {
			hist[i].Backlog += 1
		}
	}
	return hist
}
