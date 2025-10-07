// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package e2e

const (
	shootNamespace             = "shoot--logging--test"
	seedNamespace              = "seed--logging--test"
	backendContainerImage      = "ghcr.io/credativ/vali:v2.2.27"
	logGeneratorContainerImage = "nickytd/log-generator:latest"
	daemonSetName              = "fluent-bit"
	eventLoggerName            = "event-logger"
	seedBackendName            = "seed"
	shootBackendName           = "shoot"
)
