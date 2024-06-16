// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/prometheus/common/model"
)

type localStream struct {
	lastTimestamp time.Time
	logCount      int
}

func (s *localStream) add(timestamp time.Time) error {
	if s.lastTimestamp.After(timestamp) {
		return errors.New("entry out of order")
	}
	s.lastTimestamp = timestamp
	s.logCount++

	return nil
}

func labelSetToString(ls model.LabelSet) string {
	var labelSetStr []string

	for key, value := range ls {
		labelSetStr = append(labelSetStr, string(key)+"="+string(value))
	}

	sort.Strings(labelSetStr)
	return strings.Join(labelSetStr, ",")
}
