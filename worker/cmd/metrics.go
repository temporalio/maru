package main

import (
	"time"

	"go.temporal.io/sdk/log"
	"go.uber.org/zap"

	prom "github.com/prometheus/client_golang/prometheus"
	"github.com/temporalio/maru/common"
	"github.com/uber-go/tally/v4"
	"github.com/uber-go/tally/v4/prometheus"
)

func newPrometheusScope(logger log.Logger, c prometheus.Configuration) tally.Scope {
	reporter, err := c.NewReporter(
		prometheus.ConfigurationOptions{
			Registry: prom.NewRegistry(),
			OnError: func(err error) {
				logger.Error("error in prometheus reporter", err)
			},
		},
	)
	if err != nil {
		logger.(*ZapAdapter).zl.Fatal("error creating prometheus reporter", zap.Error(err))
	}
	scopeOpts := tally.ScopeOptions{
		CachedReporter:  reporter,
		Separator:       prometheus.DefaultSeparator,
		SanitizeOptions: &sanitizeOptions,
		Prefix:          common.GetEnvOrDefaultString(logger, "PROMETHEUS_METRICS_PREFIX", ""),
	}
	scope, _ := tally.NewRootScope(scopeOpts, time.Second)

	logger.Info("prometheus metrics scope created")
	return scope
}

// tally sanitizer options that satisfy Prometheus restrictions.
// This will rename metrics at the tally emission level, so metrics name we
// use maybe different from what gets emitted. In the current implementation
// it will replace - and . with _
var (
	safeCharacters = []rune{'_'}

	sanitizeOptions = tally.SanitizeOptions{
		NameCharacters: tally.ValidCharacters{
			Ranges:     tally.AlphanumericRange,
			Characters: safeCharacters,
		},
		KeyCharacters: tally.ValidCharacters{
			Ranges:     tally.AlphanumericRange,
			Characters: safeCharacters,
		},
		ValueCharacters: tally.ValidCharacters{
			Ranges:     tally.AlphanumericRange,
			Characters: safeCharacters,
		},
		ReplacementCharacter: tally.DefaultReplacementCharacter,
	}
)
