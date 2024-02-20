// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"os"

	"github.com/gardener/logging/cmd/event-logger/app"
)

func main() {
	if err := app.NewCommandStartGardenerEventLogger().Execute(); err != nil {
		os.Exit(1)
	}
}
