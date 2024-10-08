// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"context"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
	"testing"
)

func TestShootEventsLogs(t *testing.T) {
	deploymentFeature := features.New("shoot/events").WithLabel("type", "events").
		Setup(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			return ctx
		}).
		Assess("check events in in shoot backend - tbd", func(ctx context.Context, t *testing.T,
			cfg *envconf.Config) context.Context {
			return ctx
		}).
		Teardown(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {

			return ctx
		}).Feature()

	testenv.Test(t, deploymentFeature)
}
