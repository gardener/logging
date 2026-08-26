// Copyright 2025 SPDX-FileCopyrightText: Copyright Contributors to the Gardener project
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"os"

	"github.com/gardener/logging/v1/cmd/event-logger/app"
)

func main() {
	if err := app.NewCommandStartGardenerEventLogger().Execute(); err != nil {
		os.Exit(1)
	}
}
