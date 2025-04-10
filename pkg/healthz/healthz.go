// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package healthz

import (
	"bytes"
	"errors"
	"io"
	"net/http"

	"sigs.k8s.io/controller-runtime/pkg/healthz"
)

// Handler returns http.Handler
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
	defer func() { _ = resp.Body.Close() }()

	// We Read the response body on the line below.
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if bytes.Equal(m.previousMetrics, body) {
		return errors.New("the metrics have not been changed since last healthz check")
	}
	m.previousMetrics = body

	return nil
}
