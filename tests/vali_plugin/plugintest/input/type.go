// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package input

import "github.com/prometheus/common/model"

type PodOutput interface {
	GetLabelSet() model.LabelSet
	GetGeneratedLogsCount() int
}

type Pod interface {
	GenerateLogRecord() map[interface{}]interface{}
	GetOutput() PodOutput
}
