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

package healthz

import (
	"bytes"
	"errors"
	"io/ioutil"
	"net/http"

	"sigs.k8s.io/controller-runtime/pkg/healthz"
)

func Handler(flbListeningIP, port string) http.Handler {
	if flbListeningIP == "" {
		flbListeningIP = "127.0.0.1"
	}
	if port == "" {
		port = "2020"
	}

	chkr := &metricsChecker{
		url: "http://" + flbListeningIP + ":" + port + "/api/v1/metrics",
	}

	return &healthz.Handler{
		Checks: map[string]healthz.Checker{
			"healthz": chkr.stallMetrics,
		},
	}
}

type metricsChecker struct {
	url             string
	previousMetrics []byte
}

func (m *metricsChecker) stallMetrics(_ *http.Request) error {
	resp, err := http.Get(m.url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	//We Read the response body on the line below.
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if bytes.Equal(m.previousMetrics, body) {
		return errors.New("the metrics have not been changed since last healthz check")
	}
	m.previousMetrics = body
	return nil
}
