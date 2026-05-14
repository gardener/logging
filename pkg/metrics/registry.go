package metrics

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
)

var (
	metricsReg     *prometheus.Registry
	metricsRegOnce sync.Once
)

func InitRegistry() {
	metricsRegOnce.Do(func() {
		metricsReg = prometheus.NewRegistry()
		metricsReg.MustRegister(
			collectors.NewGoCollector(),
			collectors.NewProcessCollector(
				collectors.ProcessCollectorOpts{},
			),
		)
	})
}

func RegistryInst() *prometheus.Registry {
	return metricsReg
}
