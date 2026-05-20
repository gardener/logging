// Copyright 2025 SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package targets

import "strings"

// Target represents the deployment client target type, Seed or Shoot
type Target int

const (
	// Seed is the client target for the Gardener Seed cluster
	Seed Target = iota
	// Shoot is the client target for the Gardener Shoot clusters
	Shoot
	// Unknown represents an unknown target
	Unknown
)

func (t Target) String() string {
	switch t {
	case Seed:
		return "SEED"
	case Shoot:
		return "SHOOT"
	default:
		return "UNKNOWN"
	}
}

// FromString converts a string to a Target type
func FromString(target string) Target {
	switch strings.ToUpper(target) {
	case "SEED":
		return Seed
	case "SHOOT":
		return Shoot
	default:
		return Unknown
	}
}
