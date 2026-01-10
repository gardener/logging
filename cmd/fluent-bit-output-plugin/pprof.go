// Copyright 2025 SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"runtime"
)

// setPprofProfile configures pprof profiling settings.
// It uses sync.Once to ensure these settings are only configured once during the plugin's lifetime.
// This function enables mutex profiling at 1/5 fraction and block profiling for performance analysis.
func setPprofProfile() {
	pprofOnce.Do(func() {
		runtime.SetMutexProfileFraction(5)
		runtime.SetBlockProfileRate(1)
	})
}
