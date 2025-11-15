// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package producer

// PodOutput is an interface that defines the methods for getting label set and generated logs count.
type PodOutput interface {
	GetGeneratedLogsCount() int
}

// Pod is an interface that defines the methods for generating log records and getting output information.
type Pod interface {
	GenerateLogRecord() map[any]any
	GetOutput() PodOutput
}
