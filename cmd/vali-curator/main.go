// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"net/http"
	_ "net/http/pprof" // #nosec: G108
	"os"
	"os/signal"
	"runtime"
	"time"

	"github.com/go-kit/log/level"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/gardener/logging/cmd/vali-curator/app"
	"github.com/gardener/logging/pkg/vali/curator"
)

func main() {
	conf, logger, err := app.ParseConfiguration()
	if err != nil {
		_ = level.Error(logger).Log("msg", "error", err)
		os.Exit(1)
	}

	// metrics
	go func() {
		runtime.SetMutexProfileFraction(5)
		runtime.SetBlockProfileRate(1)
		mux := http.NewServeMux()
		mux.Handle("/curator/metrics", promhttp.Handler())
		server := &http.Server{
			Addr:              ":2718",
			ReadHeaderTimeout: time.Second * 30,
			Handler:           mux,
		}
		if err := server.ListenAndServe(); err != nil {
			_ = level.Error(logger).Log("Curator metric server error", err.Error())
		}
	}()

	curator := curator.NewCurator(*conf, logger)
	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt)
	go func() { curator.Run() }()
	sig := <-c
	_ = level.Error(logger).Log("msg", "error", "Got %s signal. Aborting...", sig)
	curator.Stop()
}
