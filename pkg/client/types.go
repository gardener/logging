// Copyright 2025 SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"

	"github.com/go-logr/logr"

	"github.com/gardener/logging/v1/pkg/config"
	"github.com/gardener/logging/v1/pkg/metrics"
	"github.com/gardener/logging/v1/pkg/types"
)

// NewClientFunc is a function type for creating new OutputClient instances
type NewClientFunc func(ctx context.Context, cfg config.Config, logger logr.Logger, m *metrics.FluentBitGardenerMetrics) (types.OutputClient, error)
