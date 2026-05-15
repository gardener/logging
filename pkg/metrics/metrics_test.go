// Copyright 2025 SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
// SPDX-License-Identifier: Apache-2.0

package metrics_test

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/gardener/logging/v1/pkg/metrics"
)

var _ = Describe("FluentBitGardenerMetrics", func() {
	Describe("NewFluentBitGardenerMetrics", func() {
		It("should create metrics without panic", func() {
			reg := metrics.NewRegistry()
			Expect(func() {
				metrics.NewFluentBitGardenerMetrics(reg)
			}).NotTo(Panic())
		})

		It("should return a non-nil instance with all fields populated", func() {
			reg := metrics.NewRegistry()
			m := metrics.NewFluentBitGardenerMetrics(reg)

			Expect(m).NotTo(BeNil())
			Expect(m.Clients).NotTo(BeNil())
			Expect(m.Errors).NotTo(BeNil())
			Expect(m.LogsWithoutMetadata).NotTo(BeNil())
			Expect(m.IncomingLogs).NotTo(BeNil())
			Expect(m.OutputClientLogs).NotTo(BeNil())
			Expect(m.ExportedClientLogs).NotTo(BeNil())
			Expect(m.DroppedLogs).NotTo(BeNil())
			Expect(m.ThrottledLogs).NotTo(BeNil())
			Expect(m.BufferedLogs).NotTo(BeNil())
			Expect(m.DqueSize).NotTo(BeNil())
		})
	})

	Describe("Metric names and labels", func() {
		var (
			reg *prometheus.Registry
			m   *metrics.FluentBitGardenerMetrics
		)

		BeforeEach(func() {
			reg = metrics.NewRegistry()
			m = metrics.NewFluentBitGardenerMetrics(reg)
		})

		It("should register metrics with correct names", func() {
			// Trigger creation of label vectors by using them
			m.Clients.WithLabelValues("seed")
			m.Errors.WithLabelValues("test_error")
			m.LogsWithoutMetadata.WithLabelValues("Kubernetes")
			m.IncomingLogs.WithLabelValues("garden")
			m.OutputClientLogs.WithLabelValues("http://localhost")
			m.ExportedClientLogs.WithLabelValues("http://localhost")
			m.DroppedLogs.WithLabelValues("http://localhost", "noop")
			m.ThrottledLogs.WithLabelValues("http://localhost")
			m.BufferedLogs.WithLabelValues("http://localhost")
			m.DqueSize.WithLabelValues("test-queue")

			families, err := reg.Gather()
			Expect(err).NotTo(HaveOccurred())

			names := make(map[string]bool)
			for _, f := range families {
				names[f.GetName()] = true
			}

			Expect(names).To(HaveKey("fluentbit_gardener_clients_total"))
			Expect(names).To(HaveKey("fluentbit_gardener_errors_total"))
			Expect(names).To(HaveKey("fluentbit_gardener_logs_without_metadata_total"))
			Expect(names).To(HaveKey("fluentbit_gardener_incoming_logs_total"))
			Expect(names).To(HaveKey("fluentbit_gardener_output_client_logs_total"))
			Expect(names).To(HaveKey("fluentbit_gardener_exported_client_logs_total"))
			Expect(names).To(HaveKey("fluentbit_gardener_dropped_logs_total"))
			Expect(names).To(HaveKey("fluentbit_gardener_throttled_logs_total"))
			Expect(names).To(HaveKey("fluentbit_gardener_buffered_logs"))
			Expect(names).To(HaveKey("fluentbit_gardener_dque_size"))
		})

		It("should have correct label keys for each metric", func() {
			// Clients: type
			m.Clients.WithLabelValues("seed").Inc()
			families, err := reg.Gather()
			Expect(err).NotTo(HaveOccurred())

			for _, f := range families {
				switch f.GetName() {
				case "fluentbit_gardener_clients_total":
					Expect(f.GetMetric()).To(HaveLen(1))
					Expect(f.GetMetric()[0].GetLabel()).To(HaveLen(1))
					Expect(f.GetMetric()[0].GetLabel()[0].GetName()).To(Equal("type"))
				}
			}
		})
	})

	Describe("Functional correctness", func() {
		var m *metrics.FluentBitGardenerMetrics

		BeforeEach(func() {
			reg := metrics.NewRegistry()
			m = metrics.NewFluentBitGardenerMetrics(reg)
		})

		It("should increment counters correctly", func() {
			m.Errors.WithLabelValues("TestError").Inc()
			m.Errors.WithLabelValues("TestError").Inc()

			val := testutil.ToFloat64(m.Errors.WithLabelValues("TestError"))
			Expect(val).To(Equal(float64(2)))
		})

		It("should track gauges correctly", func() {
			m.Clients.WithLabelValues("seed").Inc()
			m.Clients.WithLabelValues("seed").Inc()
			m.Clients.WithLabelValues("seed").Dec()

			val := testutil.ToFloat64(m.Clients.WithLabelValues("seed"))
			Expect(val).To(Equal(float64(1)))
		})

		It("should track DroppedLogs with host and reason labels", func() {
			m.DroppedLogs.WithLabelValues("http://endpoint:3100", "noop").Inc()
			m.DroppedLogs.WithLabelValues("http://endpoint:3100", "queue_full").Add(5)

			noop := testutil.ToFloat64(m.DroppedLogs.WithLabelValues("http://endpoint:3100", "noop"))
			queueFull := testutil.ToFloat64(m.DroppedLogs.WithLabelValues("http://endpoint:3100", "queue_full"))

			Expect(noop).To(Equal(float64(1)))
			Expect(queueFull).To(Equal(float64(5)))
		})

		It("should track BufferedLogs gauge increment and decrement", func() {
			m.BufferedLogs.WithLabelValues("http://endpoint:3100").Inc()
			m.BufferedLogs.WithLabelValues("http://endpoint:3100").Inc()
			m.BufferedLogs.WithLabelValues("http://endpoint:3100").Dec()

			val := testutil.ToFloat64(m.BufferedLogs.WithLabelValues("http://endpoint:3100"))
			Expect(val).To(Equal(float64(1)))
		})

		It("should set DqueSize gauge", func() {
			m.DqueSize.WithLabelValues("test-queue").Set(42)

			val := testutil.ToFloat64(m.DqueSize.WithLabelValues("test-queue"))
			Expect(val).To(Equal(float64(42)))
		})
	})

	Describe("Isolation between registries", func() {
		It("should not interfere between two metrics instances", func() {
			reg1 := metrics.NewRegistry()
			reg2 := metrics.NewRegistry()

			m1 := metrics.NewFluentBitGardenerMetrics(reg1)
			m2 := metrics.NewFluentBitGardenerMetrics(reg2)

			// Increment only m1
			m1.Errors.WithLabelValues("TestError").Inc()
			m1.Errors.WithLabelValues("TestError").Inc()

			// m2 should be unaffected
			val1 := testutil.ToFloat64(m1.Errors.WithLabelValues("TestError"))
			val2 := testutil.ToFloat64(m2.Errors.WithLabelValues("TestError"))

			Expect(val1).To(Equal(float64(2)))
			Expect(val2).To(Equal(float64(0)))
		})

		It("should allow independent gauge tracking", func() {
			reg1 := metrics.NewRegistry()
			reg2 := metrics.NewRegistry()

			m1 := metrics.NewFluentBitGardenerMetrics(reg1)
			m2 := metrics.NewFluentBitGardenerMetrics(reg2)

			m1.Clients.WithLabelValues("seed").Set(5)
			m2.Clients.WithLabelValues("seed").Set(10)

			Expect(testutil.ToFloat64(m1.Clients.WithLabelValues("seed"))).To(Equal(float64(5)))
			Expect(testutil.ToFloat64(m2.Clients.WithLabelValues("seed"))).To(Equal(float64(10)))
		})
	})

	Describe("NewRegistry", func() {
		It("should create a registry with Go and process collectors", func() {
			reg := metrics.NewRegistry()
			Expect(reg).NotTo(BeNil())

			families, err := reg.Gather()
			Expect(err).NotTo(HaveOccurred())

			names := make(map[string]bool)
			for _, f := range families {
				names[f.GetName()] = true
			}

			// Go collector metrics
			Expect(names).To(HaveKey("go_goroutines"))
			// Process collector metrics
			Expect(names).To(HaveKey("process_open_fds"))
		})

		It("should allow registering FluentBitGardenerMetrics", func() {
			reg := metrics.NewRegistry()
			Expect(func() {
				metrics.NewFluentBitGardenerMetrics(reg)
			}).NotTo(Panic())
		})
	})
})
