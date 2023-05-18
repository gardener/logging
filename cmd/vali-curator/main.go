// Copyright (c) 2021 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"runtime"

	"github.com/gardener/logging/cmd/vali-curator/app"
	"github.com/gardener/logging/pkg/vali/curator"
	"github.com/go-kit/kit/log/level"
	"github.com/prometheus/client_golang/prometheus/promhttp"
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
		http.Handle("/curator/metrics", promhttp.Handler())
		if err := http.ListenAndServe(":2718", nil); err != nil {
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
