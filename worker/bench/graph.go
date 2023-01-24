package bench

import (
	"bytes"
	"fmt"
	"strconv"

	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/components"
	"github.com/go-echarts/go-echarts/v2/opts"
)

func printHistogramGraph(request benchWorkflowRequest, values []histogramValue) string {
	chart := charts.NewLine()
	chart.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{
			Title: fmt.Sprintf("Benchmark: %s", request.Workflow.Name),
			//Subtitle: fmt.Sprintf("Steps:\n%+v", request.Steps),
		}),
		charts.WithLegendOpts(opts.Legend{Show: true}),
		charts.WithTooltipOpts(opts.Tooltip{Show: true, Trigger: "axis"}),
		charts.WithXAxisOpts(opts.XAxis{Name: "Time (s)"}),
		charts.WithInitializationOpts(opts.Initialization{
			Width:  "1440px",
			Height: "900px",
		}),
	)

	interval := request.Report.IntervalInSeconds
	times := make([]string, len(values))
	workflowsStartedRate := make([]float32, len(values))
	workflowsExecutionRate := make([]float32, len(values))
	workflowsClosedRate := make([]float32, len(values))
	backlog := make([]float32, len(values))
	for i, v := range values {
		times[i] = strconv.Itoa((i + 1) * interval)
		workflowsStartedRate[i] = float32(v.Started) / float32(interval)
		workflowsExecutionRate[i] = float32(v.Execution) / float32(interval)
		workflowsClosedRate[i] = float32(v.Closed) / float32(interval)
		backlog[i] = float32(v.Backlog)
	}

	seriesOpts := []charts.SeriesOpts{
		charts.WithLabelOpts(opts.Label{Show: true, Position: "top"}),
		charts.WithLineChartOpts(opts.LineChart{Smooth: true}),
	}

	chart.SetXAxis(times).
		AddSeries("Workflows Started Rate", generateLineData(workflowsStartedRate), seriesOpts...).
		AddSeries("Workflows Execution Rate", generateLineData(workflowsExecutionRate), seriesOpts...).
		AddSeries("Workflows Closed Rate", generateLineData(workflowsClosedRate), seriesOpts...).
		AddSeries("Backlog", generateLineData(backlog), seriesOpts...)

	// TODO:
	// 1. Add raw data as a table (collapsed if possible)
	// 2. Multiple charts / page?
	// 3. Add the remaining data in a second chart: workflows started (v.Started),
	//    workflow executions (v.Execution), workflows closed (v.Closed)
	// 4. Example used: https://github.com/go-echarts/examples/blob/master/examples/line.go

	page := components.NewPage()
	page.AddCharts(chart)

	var b bytes.Buffer
	if err := page.Render(&b); err != nil {
		return err.Error()
	}

	return b.String()
}

func generateLineData(data []float32) []opts.LineData {
	items := make([]opts.LineData, 0)
	for i := 0; i < len(data); i++ {
		items = append(items, opts.LineData{Value: data[i]})
	}
	return items
}
