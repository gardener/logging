package e2e

import (
	"context"
	"fmt"
	"os"
	"testing"

	"sigs.k8s.io/e2e-framework/pkg/env"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/envfuncs"
	"sigs.k8s.io/e2e-framework/pkg/features"
	"sigs.k8s.io/e2e-framework/support/kind"

	"github.com/gardener/logging/v1/pkg/log"
)

type contextKey string

const (
	fluentBitImage       = "ghcr.io/fluent/fluent-operator/fluent-bit:4.2.0"
	fluentBitPluginImage = "fluent-bit-plugin:e2e"
	eventLoggerImage     = "event-logger:e2e"
	namespaceKey         = contextKey("namespace")
)

var (
	testenv env.Environment
)

func TestMain(m *testing.M) {
	testenv = env.New()

	logger := log.NewLogger("debug")

	kindClusterName := envconf.RandomName("logging", 16)
	namespace := envconf.RandomName("fluent-bit", 16)

	// Use pre-defined environment funcs to create a kind cluster prior to test run
	testenv.Setup(
		envfuncs.CreateClusterWithConfig(kind.NewProvider(), kindClusterName, "./config/kind-config.yaml"),
		envfuncs.SetupCRDs("./config", "*-crd.yaml"),
		envfuncs.CreateNamespace(namespace),
		pullFluentBitImage(logger, fluentBitImage),
		envfuncs.LoadImageToCluster(kindClusterName, fluentBitImage),
		buildFluentBitImages(logger, fluentBitPluginImage, eventLoggerImage),
		envfuncs.LoadImageToCluster(kindClusterName, fluentBitPluginImage),
		envfuncs.LoadImageToCluster(kindClusterName, eventLoggerImage),
		createFluentBitDaemonSet(logger, namespace, fluentBitPluginImage, fluentBitImage),

		// create single victoria-logs-seed statefulset in the namespace
		// create single victoria-logs-shoot statefulset in the namespace
		// create otel-collector-shoot deployment in the namespace
		// link otel-collector-shoot exporter to victoria-logs-shoot
		// create otel-collector-seed deployment in the namespace
		// link otel-collector-seed exporter to victoria-logs-seed
		// create event-logger deployment in the namespace
		// create 100 cluster resources
		// create 100 namespaces (shoots)
		// create single external service per shoot namespace to link to otel-collector-shoot

		// with this environment we can run e2e tests against the setup
	)

	// Use pre-defined environment funcs to teardown kind cluster after tests
	testenv.Finish(
		envfuncs.ExportClusterLogs(kindClusterName, "./logs"),
		envfuncs.DeleteNamespace(namespace),
		envfuncs.DestroyCluster(kindClusterName),
	)

	testenv.BeforeEachFeature(func(ctx context.Context, cfg *envconf.Config, t *testing.T, f features.Feature) (context.Context, error) {
		// ensure fluent-bit is running before each feature
		if err := waitForDaemonSetReady(ctx, cfg, namespace, "fluent-bit"); err != nil {
			return ctx, fmt.Errorf("fluent-bit DaemonSet is not ready: %w", err)
		}

		return ctx, nil
	})

	// launch package tests
	os.Exit(testenv.Run(m))
}
