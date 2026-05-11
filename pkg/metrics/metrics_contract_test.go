// Copyright 2025 SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
// SPDX-License-Identifier: Apache-2.0

package metrics_test

import (
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"

	"github.com/gardener/logging/v1/pkg/metrics"
)

var _ = Describe("Metrics Contract", func() {
	It("should discover all registered metrics with type and labels (plugin specific + go runtime and process metrics)", func() {
		primeCollectors()

		mfs, err := prometheus.DefaultGatherer.Gather()
		Expect(err).NotTo(HaveOccurred())
		gathered := make(map[string]dto.MetricType, len(mfs))
		for _, mf := range mfs {
			gathered[mf.GetName()] = mf.GetType()
		}
		Expect(gathered).To(HaveLen(46))

		// Verify each plugin metric is present with the correct type
		Expect(gathered).To(HaveKeyWithValue("fluentbit_gardener_clients_total", dto.MetricType_GAUGE))
		Expect(gathered).To(HaveKeyWithValue("fluentbit_gardener_errors_total", dto.MetricType_COUNTER))
		Expect(gathered).To(HaveKeyWithValue("fluentbit_gardener_logs_without_metadata_total", dto.MetricType_COUNTER))
		Expect(gathered).To(HaveKeyWithValue("fluentbit_gardener_incoming_logs_total", dto.MetricType_COUNTER))
		Expect(gathered).To(HaveKeyWithValue("fluentbit_gardener_output_client_logs_total", dto.MetricType_COUNTER))
		Expect(gathered).To(HaveKeyWithValue("fluentbit_gardener_exported_client_logs_total", dto.MetricType_COUNTER))
		Expect(gathered).To(HaveKeyWithValue("fluentbit_gardener_dropped_logs_total", dto.MetricType_COUNTER))
		Expect(gathered).To(HaveKeyWithValue("fluentbit_gardener_throttled_logs_total", dto.MetricType_COUNTER))
		Expect(gathered).To(HaveKeyWithValue("fluentbit_gardener_buffered_logs", dto.MetricType_GAUGE))
		Expect(gathered).To(HaveKeyWithValue("fluentbit_gardener_dque_size", dto.MetricType_GAUGE))

		// Verify go runtime and process metrics are present with the correct type
		Expect(gathered).To(HaveKeyWithValue("go_goroutines", dto.MetricType_GAUGE))
		Expect(gathered).To(HaveKeyWithValue("go_threads", dto.MetricType_GAUGE))
		Expect(gathered).To(HaveKeyWithValue("go_info", dto.MetricType_GAUGE))
		Expect(gathered).To(HaveKeyWithValue("go_memstats_alloc_bytes", dto.MetricType_GAUGE))
		Expect(gathered).To(HaveKeyWithValue("go_memstats_alloc_bytes_total", dto.MetricType_COUNTER))
		Expect(gathered).To(HaveKeyWithValue("go_memstats_sys_bytes", dto.MetricType_GAUGE))
		Expect(gathered).To(HaveKeyWithValue("go_memstats_heap_alloc_bytes", dto.MetricType_GAUGE))
		Expect(gathered).To(HaveKeyWithValue("go_memstats_heap_inuse_bytes", dto.MetricType_GAUGE))
		Expect(gathered).To(HaveKeyWithValue("go_memstats_heap_idle_bytes", dto.MetricType_GAUGE))
		Expect(gathered).To(HaveKeyWithValue("go_memstats_heap_objects", dto.MetricType_GAUGE))
		Expect(gathered).To(HaveKeyWithValue("go_memstats_heap_released_bytes", dto.MetricType_GAUGE))
		Expect(gathered).To(HaveKeyWithValue("go_memstats_heap_sys_bytes", dto.MetricType_GAUGE))
		Expect(gathered).To(HaveKeyWithValue("go_memstats_stack_inuse_bytes", dto.MetricType_GAUGE))
		Expect(gathered).To(HaveKeyWithValue("go_memstats_stack_sys_bytes", dto.MetricType_GAUGE))
		Expect(gathered).To(HaveKeyWithValue("go_memstats_mspan_inuse_bytes", dto.MetricType_GAUGE))
		Expect(gathered).To(HaveKeyWithValue("go_memstats_mspan_sys_bytes", dto.MetricType_GAUGE))
		Expect(gathered).To(HaveKeyWithValue("go_memstats_mcache_inuse_bytes", dto.MetricType_GAUGE))
		Expect(gathered).To(HaveKeyWithValue("go_memstats_mcache_sys_bytes", dto.MetricType_GAUGE))
		Expect(gathered).To(HaveKeyWithValue("go_memstats_buck_hash_sys_bytes", dto.MetricType_GAUGE))
		Expect(gathered).To(HaveKeyWithValue("go_memstats_gc_sys_bytes", dto.MetricType_GAUGE))
		Expect(gathered).To(HaveKeyWithValue("go_memstats_other_sys_bytes", dto.MetricType_GAUGE))
		Expect(gathered).To(HaveKeyWithValue("go_memstats_next_gc_bytes", dto.MetricType_GAUGE))
		Expect(gathered).To(HaveKeyWithValue("go_memstats_last_gc_time_seconds", dto.MetricType_GAUGE))
		Expect(gathered).To(HaveKeyWithValue("go_memstats_frees_total", dto.MetricType_COUNTER))
		Expect(gathered).To(HaveKeyWithValue("go_memstats_mallocs_total", dto.MetricType_COUNTER))
		Expect(gathered).To(HaveKeyWithValue("go_gc_duration_seconds", dto.MetricType_SUMMARY))
		Expect(gathered).To(HaveKeyWithValue("go_gc_gogc_percent", dto.MetricType_GAUGE))
		Expect(gathered).To(HaveKeyWithValue("go_gc_gomemlimit_bytes", dto.MetricType_GAUGE))
		Expect(gathered).To(HaveKeyWithValue("go_sched_gomaxprocs_threads", dto.MetricType_GAUGE))
		Expect(gathered).To(HaveKeyWithValue("process_cpu_seconds_total", dto.MetricType_COUNTER))
		Expect(gathered).To(HaveKeyWithValue("process_max_fds", dto.MetricType_GAUGE))
		Expect(gathered).To(HaveKeyWithValue("process_open_fds", dto.MetricType_GAUGE))
		Expect(gathered).To(HaveKeyWithValue("process_resident_memory_bytes", dto.MetricType_GAUGE))
		Expect(gathered).To(HaveKeyWithValue("process_start_time_seconds", dto.MetricType_GAUGE))
		Expect(gathered).To(HaveKeyWithValue("process_virtual_memory_bytes", dto.MetricType_GAUGE))
		Expect(gathered).To(HaveKeyWithValue("process_virtual_memory_max_bytes", dto.MetricType_GAUGE))
	})
})

// primeCollectors forces all metric vecs to emit on Gather().
//
// Prometheus vecs are lazy containers — a CounterVec/GaugeVec registered via
// promauto exists in the registry but produces NO output from Gather() until
// at least one time series is created via WithLabelValues(). This function
// creates a single dummy series per vec so Gather() returns the full set.
//
// To call WithLabelValues we need to know how many labels each vec expects.
// Since Desc has no public field for that, we parse it from Desc.String() which
// outputs variableLabels: {host,reason}
func primeCollectors() {
	for _, c := range metrics.AllCollectors() {
		// Get the descriptor to learn the number of variable labels.
		ch := make(chan *prometheus.Desc, 1)
		c.Describe(ch)
		s := (<-ch).String()

		// Parse label count from "variableLabels: {label1,label2}" in Desc.String().
		_, after, _ := strings.Cut(s, "variableLabels: {")
		content, _, _ := strings.Cut(after, "}")
		n := 0
		if content != "" {
			n = strings.Count(content, ",") + 1
		}

		// Create a dummy series with empty label values — the values don't matter,
		// we just need WithLabelValues called once to make Gather() emit this metric.
		labels := make([]string, n)
		switch v := c.(type) {
		case *prometheus.CounterVec:
			v.WithLabelValues(labels...)
		case *prometheus.GaugeVec:
			v.WithLabelValues(labels...)
		case *prometheus.HistogramVec:
			v.WithLabelValues(labels...)
		case *prometheus.SummaryVec:
			v.WithLabelValues(labels...)
		}
	}
}
