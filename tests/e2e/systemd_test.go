// Copyright 2025 SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"context"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func TestSystemdLogs(t *testing.T) {
	f1 := features.New("systemd/logs").
		WithLabel("type", "systemd-logs").
		Assess("kubelet and containerd logs", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			g := NewWithT(t)

			var kubeletCount, containerdCount int

			// Use Eventually to poll for positive counts with timeout
			g.Eventually(func(g Gomega) {
				// Query kubelet logs
				kubeletQuery := `_time:24h unit:"kubelet.service" | count()`
				kubeletOutput, err := queryCurl(ctx, cfg, namespace, kubeletQuery)
				g.Expect(err).NotTo(HaveOccurred(), "Failed to query kubelet logs")

				kubelet, err := parseQueryResponse(kubeletOutput)
				g.Expect(err).NotTo(HaveOccurred(), "Failed to parse kubelet query response")

				// Query containerd logs
				containerdQuery := `_time:24h unit:"containerd.service" | count()`
				containerdOutput, err := queryCurl(ctx, cfg, namespace, containerdQuery)
				g.Expect(err).NotTo(HaveOccurred(), "Failed to query containerd logs")

				containerd, err := parseQueryResponse(containerdOutput)
				g.Expect(err).NotTo(HaveOccurred(), "Failed to parse containerd query response")

				t.Logf("Found kubelet-service count: %d, containerd-service count: %d", kubelet, containerd)

				// Expect both counts to be positive
				g.Expect(kubelet).To(BeNumerically(">", 0), "kubelet-service log count should be positive")
				g.Expect(containerd).To(BeNumerically(">", 0), "containerd-service log count should be positive")

				// Store the counts for final logging
				kubeletCount = kubelet
				containerdCount = containerd
			}).WithTimeout(2 * time.Minute).WithPolling(10 * time.Second).Should(Succeed())

			t.Logf("Successfully verified systemd logs: kubelet-service=%d, containerd-service=%d", kubeletCount, containerdCount)

			return ctx
		}).Feature()

	// test feature
	testenv.Test(t, f1)
}
