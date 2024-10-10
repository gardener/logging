// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
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
	slog.Info("Running e2e tests", "pluginUnderTest", pluginUnderTest)

	testenv.Setup(

		envfuncs.CreateClusterWithConfig(
			kind.NewProvider(),
			kindClusterName,
			"./config/kind-config.yaml",
		),
		envfuncs.CreateNamespace(ShootNamespace),
		envfuncs.CreateNamespace(SeedNamespace),
		envfuncs.SetupCRDs("./config", "*-crd.yaml"),
		createFluentBitImage(pluginUnderTest),
		envfuncs.LoadImageToCluster(kindClusterName, pluginUnderTest),
		pullAndLoadContainerImage(kindClusterName, BackendContainerImage),
		pullAndLoadContainerImage(kindClusterName, LogGeneratorContainerImage),
		envfuncs.LoadImageToCluster(kindClusterName, BackendContainerImage),
		createBackend(SeedNamespace, SeedBackendName, BackendContainerImage),
		createBackend(ShootNamespace, ShootBackendName, BackendContainerImage),
		createFluentBitDaemonSet(SeedNamespace, DaemonSetName, pluginUnderTest, config, lua),
		createExtensionCluster(ShootNamespace),
	)

	testenv.Finish(
		envfuncs.ExportClusterLogs(kindClusterName, "./logs"),
		envfuncs.DestroyCluster(kindClusterName),
	)
	os.Exit(testenv.Run(m))
}
