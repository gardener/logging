// Copyright 2025 SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package types

import "strings"

// Target represents the deployment client target type, Seed or Shoot
type Target int

const (
	// SeedTarget is the client target
	SeedTarget Target = iota
	// ShootTarget is the client target
	ShootTarget
	// UnknownTarget represents an unknown target
	UnknownTarget
)

func (t Target) String() string {
	switch t {
	case SeedTarget:
		return "SEED"
	case ShootTarget:
		return "SHOOT"
	default:
		return "UNKNOWN"
	}
}

// GetTargetFromString converts a string to a Target type
func GetTargetFromString(target string) Target {
	switch strings.ToUpper(target) {
	case "SEED":
		return SeedTarget
	case "SHOOT":
		return ShootTarget
	default:
		return UnknownTarget
	}
}
