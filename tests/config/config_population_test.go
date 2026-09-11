// Copyright 2025 SPDX-FileCopyrightText: Contributors to the Gardener project
// SPDX-License-Identifier: Apache-2.0

package config_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"text/template"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const pluginSoPath = "../../build/output_plugin.so"

var expectedFields = [][2]string{
	{"SeedType", "otlp_http"},
	{"ShootType", "otlp_grpc"},
	{"Endpoint", "localhost:4317"},
	{"EndpointURL", "http://localhost:4317"},
	{"EndpointURLPath", "/v1/logs"},
	{"Insecure", "true"},
	{"Compression", "1"},
	{"Timeout", "10s"},
	{"Origin", "test-origin"},
	{"HostnameValue", "test-host"},
	{"FallbackToTagWhenMetadataIsMissing", "true"},
	{"DropLogEntryWithoutK8sMetadata", "true"},
	{"TagKey", "custom_tag"},
	{"DynamicHostPath", "map[kubernetes:map[namespace_name:namespace]]"},
	{"DynamicHostPrefix", "logging."},
	{"DynamicHostSuffix", ".svc.cluster.local:4317"},
	{"DynamicHostRegex", "^shoot-"},
	{"ControllerSyncTimeout", "2m0s"},
	{"WatchOpenTelemetryCollector", "true"},
	{"OpenTelemetryCollectorLabelSelector", "app=otel"},
	{"OpenTelemetryCollectorNamespaceLabelSelector", "env=prod"},
	{"DQueDir", "/tmp/test-dque"},
	{"DQueSegmentSize", "100"},
	{"DQueSync", "true"},
	{"DQueName", "test-queue"},
	{"DQueBatchProcessorMaxQueueSize", "256"},
	{"DQueBatchProcessorMaxBatchSize", "64"},
	{"DQueBatchProcessorExportTimeout", "15s"},
	{"DQueBatchProcessorExportInterval", "5s"},
	{"DQueBatchProcessorExportBufferSize", "32"},
	{"UseSDKBatchProcessor", "true"},
	{"SDKBatchMaxQueueSize", "1024"},
	{"SDKBatchExportTimeout", "20s"},
	{"SDKBatchExportInterval", "3s"},
	{"SDKBatchExportMaxBatchSize", "128"},
	{"RetryEnabled", "true"},
	{"RetryInitialInterval", "2s"},
	{"RetryMaxInterval", "1m0s"},
	{"RetryMaxElapsedTime", "5m0s"},
	{"RetryConfig", "configured"},
	{"ThrottleEnabled", "true"},
	{"ThrottleRequestsPerSec", "50"},
	{"TLSServerName", "test-server"},
	{"TLSInsecureSkipVerify", "true"},
	{"TLSMinVersion", "1.2"},
	{"TLSMaxVersion", "1.3"},
	{"TLSConfig", "configured"},
}

var _ = BeforeSuite(func() {
	if _, err := os.Stat(pluginSoPath); os.IsNotExist(err) {
		cmd := exec.Command("make", "plugin")
		cmd.Dir = "../.."
		cmd.Stdout = GinkgoWriter
		cmd.Stderr = GinkgoWriter
		Expect(cmd.Run()).To(Succeed(), "failed to build output_plugin.so")
	}
})

var _ = Describe("Config population", func() {
	for _, tc := range []struct {
		name      string
		configExt string
	}{
		{name: "YAML format", configExt: "yaml"},
		{name: "classic .conf format", configExt: "conf"},
	} {
		It("populates all config fields correctly from "+tc.name, func() {
			ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
			defer cancel()

			buildDir := GinkgoT().TempDir()

			// .dockerignore excludes tests/, so both files must be physically
			// copied into the temp build directory.
			copyFile(pluginSoPath, filepath.Join(buildDir, "output_plugin.so"))
			copyFile("testdata/config."+tc.configExt, filepath.Join(buildDir, "config"))

			var df bytes.Buffer
			Expect(template.Must(template.ParseFiles("testdata/Dockerfile")).Execute(&df, struct{ ConfigExt string }{tc.configExt})).To(Succeed())
			Expect(os.WriteFile(filepath.Join(buildDir, "Dockerfile"), df.Bytes(), 0o644)).To(Succeed())

			container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
				ContainerRequest: testcontainers.ContainerRequest{
					FromDockerfile: testcontainers.FromDockerfile{
						Context:    buildDir,
						Dockerfile: "Dockerfile",
						KeepImage:  false,
					},
					WaitingFor: wait.ForLog("TLSConfig").WithStartupTimeout(60 * time.Second),
				},
				Started: true,
			})
			Expect(err).NotTo(HaveOccurred())
			defer func() { _ = container.Terminate(context.Background()) }()

			logs, err := container.Logs(ctx)
			Expect(err).NotTo(HaveOccurred())
			output, err := io.ReadAll(logs)
			Expect(err).NotTo(HaveOccurred())
			_, _ = GinkgoWriter.Write(output)
			AddReportEntry("container logs", string(output))

			for _, kv := range expectedFields {
				key, val := kv[0], kv[1]
				// Match slog text handler (key=value or key="value") and JSON handler ("key":"value").
				pattern := fmt.Sprintf(
					`%s="?%s|"%s":"%s"`,
					regexp.QuoteMeta(key),
					regexp.QuoteMeta(val),
					regexp.QuoteMeta(key),
					regexp.QuoteMeta(val),
				)
				Expect(string(output)).To(MatchRegexp(pattern), "missing field: key=%s value=%s", key, val)
			}
		})
	}
})

func copyFile(src, dst string) {
	data, err := os.ReadFile(src)
	Expect(err).NotTo(HaveOccurred())
	Expect(os.WriteFile(dst, data, 0o644)).To(Succeed())
}
