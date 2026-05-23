// Copyright 2025 SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
// SPDX-License-Identifier: Apache-2.0

package metrics_test

import (
	"io"
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/gardener/logging/v1/pkg/metrics"
)

var _ = Describe("FluentBitGardenerMetrics", func() {
	var (
		reg *prometheus.Registry
		m   *metrics.FluentBitGardenerMetrics
	)

	BeforeEach(func() {
		reg = metrics.NewRegistry()
		m = metrics.RegisterFluentBitGardenerMetrics(reg)
	})

	DescribeTable("Metric exposition via /metrics endpoint",
		func(typeLine, metricLine string) {
			body := scrapeMetrics(reg, m)
			Expect(body).To(ContainSubstring(typeLine))
			Expect(body).To(ContainSubstring(metricLine))
		},
		Entry("fluentbit_gardener_clients_total",
			"# TYPE fluentbit_gardener_clients_total gauge",
			`fluentbit_gardener_clients_total{type="seed"} 1`,
		),
		Entry("fluentbit_gardener_errors_total",
			"# TYPE fluentbit_gardener_errors_total counter",
			`fluentbit_gardener_errors_total{type="test_error"} 1`,
		),
		Entry("fluentbit_gardener_logs_without_metadata_total",
			"# TYPE fluentbit_gardener_logs_without_metadata_total counter",
			`fluentbit_gardener_logs_without_metadata_total{type="Kubernetes"} 1`,
		),
		Entry("fluentbit_gardener_incoming_logs_total",
			"# TYPE fluentbit_gardener_incoming_logs_total counter",
			`fluentbit_gardener_incoming_logs_total{host="http://localhost"} 1`,
		),
		Entry("fluentbit_gardener_output_client_logs_total",
			"# TYPE fluentbit_gardener_output_client_logs_total counter",
			`fluentbit_gardener_output_client_logs_total{host="http://localhost"} 1`,
		),
		Entry("fluentbit_gardener_exported_client_logs_total",
			"# TYPE fluentbit_gardener_exported_client_logs_total counter",
			`fluentbit_gardener_exported_client_logs_total{host="http://localhost"} 1`,
		),
		Entry("fluentbit_gardener_dropped_logs_total",
			"# TYPE fluentbit_gardener_dropped_logs_total counter",
			`fluentbit_gardener_dropped_logs_total{host="http://localhost",reason="noop"} 1`,
		),
		Entry("fluentbit_gardener_throttled_logs_total",
			"# TYPE fluentbit_gardener_throttled_logs_total counter",
			`fluentbit_gardener_throttled_logs_total{host="http://localhost"} 1`,
		),
		Entry("fluentbit_gardener_buffered_logs",
			"# TYPE fluentbit_gardener_buffered_logs gauge",
			`fluentbit_gardener_buffered_logs{host="http://localhost"} 1`,
		),
		Entry("fluentbit_gardener_dque_size",
			"# TYPE fluentbit_gardener_dque_size gauge",
			`fluentbit_gardener_dque_size{name="test-queue"} 42`,
		),
	)

	Describe("Functional correctness", func() {
		It("should track multiple label values independently", func() {
			m.DroppedLogs.WithLabelValues("http://host1", "noop").Inc()
			m.DroppedLogs.WithLabelValues("http://host1", "queue_full").Add(5)

			handler := promhttp.HandlerFor(reg, promhttp.HandlerOpts{})
			req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			body, _ := io.ReadAll(rec.Result().Body)
			Expect(string(body)).To(ContainSubstring(`fluentbit_gardener_dropped_logs_total{host="http://host1",reason="noop"} 1`))
			Expect(string(body)).To(ContainSubstring(`fluentbit_gardener_dropped_logs_total{host="http://host1",reason="queue_full"} 5`))
		})
	})

	Describe("Isolation between registries", func() {
		It("should not interfere between two metrics instances", func() {
			reg2 := metrics.NewRegistry()
			m2 := metrics.RegisterFluentBitGardenerMetrics(reg2)

			m.Errors.WithLabelValues("TestError").Inc()
			m.Errors.WithLabelValues("TestError").Inc()

			// Scrape first registry
			handler1 := promhttp.HandlerFor(reg, promhttp.HandlerOpts{})
			rec1 := httptest.NewRecorder()
			handler1.ServeHTTP(rec1, httptest.NewRequest(http.MethodGet, "/metrics", nil))
			body1, _ := io.ReadAll(rec1.Result().Body)

			// Trigger m2 so the metric appears with value 0
			m2.Errors.WithLabelValues("TestError")

			// Scrape second registry
			handler2 := promhttp.HandlerFor(reg2, promhttp.HandlerOpts{})
			rec2 := httptest.NewRecorder()
			handler2.ServeHTTP(rec2, httptest.NewRequest(http.MethodGet, "/metrics", nil))
			body2, _ := io.ReadAll(rec2.Result().Body)

			Expect(string(body1)).To(ContainSubstring(`fluentbit_gardener_errors_total{type="TestError"} 2`))
			Expect(string(body2)).To(ContainSubstring(`fluentbit_gardener_errors_total{type="TestError"} 0`))
		})
	})
})

func scrapeMetrics(reg *prometheus.Registry, m *metrics.FluentBitGardenerMetrics) string {
	m.Clients.WithLabelValues("seed").Set(1)
	m.Errors.WithLabelValues("test_error").Inc()
	m.LogsWithoutMetadata.WithLabelValues("Kubernetes").Inc()
	m.IncomingLogs.WithLabelValues("http://localhost").Inc()
	m.OutputClientLogs.WithLabelValues("http://localhost").Inc()
	m.ExportedClientLogs.WithLabelValues("http://localhost").Inc()
	m.DroppedLogs.WithLabelValues("http://localhost", "noop").Inc()
	m.ThrottledLogs.WithLabelValues("http://localhost").Inc()
	m.BufferedLogs.WithLabelValues("http://localhost").Set(1)
	m.DqueSize.WithLabelValues("test-queue").Set(42)

	handler := promhttp.HandlerFor(reg, promhttp.HandlerOpts{})
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	resp := rec.Result()
	defer func() { _ = resp.Body.Close() }()
	body, _ := io.ReadAll(resp.Body)

	return string(body)
}
