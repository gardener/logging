package e2e

import (
	"context"
	"testing"

	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func TestOutputPlugin(t *testing.T) {
	f1 := features.New("shoot/logs").
		WithLabel("type", "plugin").
		Setup(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			// create 1 job with 1 logger instances per shoot namespace, each logger generates 1000 logs
			return ctx
		}).
		Assess("logs per shoot namespace", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			// check collected logs for all namespaces in victoria-logs-shoot instance
			return ctx
		}).
		Assess("total logs", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			// check total logs count in victoria-logs-shoot instance
			return ctx
		}).
		Teardown(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			// clean up created jobs and logger instances
			return ctx
		}).
		Feature()

	f2 := features.New("seed/logs").
		WithLabel("type", "plugin").
		Setup(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			// create 1 job with 1 logger instances in fluent-bit namespace, each logger generates 1000 logs
			return ctx
		}).
		Assess("logs in seed", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			// check total logs count in victoria-logs-seed instance
			return ctx
		}).
		Teardown(func(ctx context.Context, t *testing.T, config *envconf.Config) context.Context {
			// clean up created jobs and logger instances
			return ctx
		}).Feature()

	// test feature
	testenv.Test(t, f1, f2)
}

func TestEventLogger(t *testing.T) {
	f1 := features.New("shoot/events").
		WithLabel("type", "event-logger").
		Setup(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			// create single namespace k8s event in each shoot namespace
			return ctx
		}).
		Assess("events per shoot namespace", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			// check created event in victoria-logs-shoot instance per shoot namespace
			return ctx
		}).Feature()

	// test feature
	testenv.Test(t, f1)
}

func TestSystemdLogs(t *testing.T) {
	f1 := features.New("systemd/logs").
		WithLabel("type", "systemd-logs").
		Assess("system logs", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			// check systemd logs exist in victoria-logs-seed instance
			return ctx
		}).Feature()

	// test feature
	testenv.Test(t, f1)
}
