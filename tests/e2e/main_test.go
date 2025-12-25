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
	victoriaLogsImage    = "quay.io/victoriametrics/victoria-logs:v1.43.0"
	fluentBitImage       = "ghcr.io/fluent/fluent-operator/fluent-bit:v4.2.0"
	fluentBitPluginImage = "fluent-bit-plugin:e2e"
	eventLoggerImage     = "event-logger:e2e"
	fetcherImage         = "fetcher:e2e"
	namespace            = "fluent-bit"
)

var (
	testenv env.Environment
)

func TestMain(m *testing.M) {
	testenv = env.New()

	logger := log.NewLogger("debug")

	kindClusterName := envconf.RandomName("logging", 16)

	// Use pre-defined environment funcs to create a kind cluster prior to test run
	testenv.Setup(
		envfuncs.CreateClusterWithConfig(
			kind.NewProvider(),
			kindClusterName,
			"./config/kind-config.yaml",
			kind.WithImage("kindest/node:v1.35.0"),
		),
		
		envfuncs.SetupCRDs("./config", "*-crd.yaml"),
		envfuncs.CreateNamespace(namespace),
		loadContainerImage(logger, kindClusterName, fluentBitImage),
		loadContainerImage(logger, kindClusterName, victoriaLogsImage),
		buildFluentBitImages(logger, fluentBitPluginImage, eventLoggerImage),
		buildFetcherImage(logger, fetcherImage),
		envfuncs.LoadImageToCluster(kindClusterName, fluentBitPluginImage),
		envfuncs.LoadImageToCluster(kindClusterName, eventLoggerImage),
		envfuncs.LoadImageToCluster(kindClusterName, fetcherImage),
		createFluentBitDaemonSet(logger, namespace, fluentBitPluginImage, fluentBitImage),
		createVictoriaLogsStatefulSet(logger, namespace, victoriaLogsImage),
		createFetcherDeployment(logger, namespace, fetcherImage, "http://victoria-logs-0.victoria-logs.fluent-bit.svc.cluster.local:9428"),

		// with this environment we can run e2e tests against the setup
	)

	// Use pre-defined environment funcs to teardown kind cluster after tests
	testenv.Finish(
		envfuncs.ExportClusterLogs(kindClusterName, "./logs"),
		//envfuncs.DeleteNamespace(namespace),
		//envfuncs.DestroyCluster(kindClusterName),
	)

	testenv.BeforeEachFeature(func(ctx context.Context, cfg *envconf.Config, t *testing.T, f features.Feature) (context.Context, error) {
		// ensure fluent-bit is running before each feature
		if err := waitForDaemonSetReady(ctx, cfg, namespace, "fluent-bit"); err != nil {
			return ctx, fmt.Errorf("fluent-bit DaemonSet is not ready: %w", err)
		}

		// ensure victoria-logs is up and running before each feature
		if err := waitForStatefulSetReady(ctx, cfg, namespace, "victoria-logs"); err != nil {
			return ctx, fmt.Errorf("victoria-logs StatefulSet is not ready: %w", err)
		}

		// ensure fetcher deployment is up and running before each feature
		if err := waitForDeploymentReady(ctx, cfg, namespace, "log-fetcher"); err != nil {
			return ctx, fmt.Errorf("log-fetcher Deployment is not ready: %w", err)
		}

		return ctx, nil
	})

	// launch package tests
	os.Exit(testenv.Run(m))
}
