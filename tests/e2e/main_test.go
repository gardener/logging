// Copyright 2025 SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	_ "embed"
	"log/slog"
	"os"
	"testing"

	"sigs.k8s.io/e2e-framework/pkg/env"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/envfuncs"
	"sigs.k8s.io/e2e-framework/support/kind"
)

var testenv env.Environment

//go:embed config/fluent-bit.conf
var config string

//go:embed config/add_tag_to_record.lua
var lua string

func TestMain(m *testing.M) {
	testenv, _ = env.NewFromFlags()
	kindClusterName := envconf.RandomName("kind-local", 16)
	pluginUnderTest := envconf.RandomName("e2e/fluent-bit-vali:test", 30)
	eventLoggerUnderTest := envconf.RandomName("e2e/event-logger:test", 30)
	slog.Info("Running e2e tests", "pluginUnderTest", pluginUnderTest, "KIND_PATH", os.Getenv("KIND_PATH"))

	testenv.Setup(

		envfuncs.CreateClusterWithConfig(
			kind.NewProvider().WithPath(os.Getenv("KIND_PATH")),
			kindClusterName,
			"./config/kind-config.yaml",
		),
		envfuncs.CreateNamespace(shootNamespace),
		envfuncs.CreateNamespace(seedNamespace),
		envfuncs.SetupCRDs("./config", "*-crd.yaml"),
		createContainerImage(pluginUnderTest, "fluent-bit-output"),
		createContainerImage(eventLoggerUnderTest, "event-logger"),
		envfuncs.LoadImageToCluster(kindClusterName, pluginUnderTest),
		envfuncs.LoadImageToCluster(kindClusterName, eventLoggerUnderTest),
		pullAndLoadContainerImage(kindClusterName, backendContainerImage),
		pullAndLoadContainerImage(kindClusterName, logGeneratorContainerImage),
		envfuncs.LoadImageToCluster(kindClusterName, backendContainerImage),
		createBackend(seedNamespace, seedBackendName, backendContainerImage),
		createBackend(shootNamespace, shootBackendName, backendContainerImage),
		createFluentBitDaemonSet(seedNamespace, daemonSetName, pluginUnderTest, config, lua),
		createEventLoggerDeployment(shootNamespace, eventLoggerName, eventLoggerUnderTest),
		createExtensionCluster(shootNamespace),
	)

	testenv.Finish(
		envfuncs.ExportClusterLogs(kindClusterName, "./logs"),
		envfuncs.DestroyCluster(kindClusterName),
	)
	testenv.Run(m)
}
