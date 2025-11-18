package client

import "strings"

// Target represents the deployment client target type, Seed or Shoot
type Target int

const (
	// Seed is the client target
	Seed Target = iota
	// Shoot is the client target
	Shoot
	// UNKNOWN represents an unknown target
	UNKNOWN
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

// GetTargetFromString converts a string to a Target type
func GetTargetFromString(target string) Target {
	switch strings.ToUpper(target) {
	case "SEED":
		return Seed
	case "SHOOT":
		return Shoot
	default:
		return UNKNOWN
	}
}
