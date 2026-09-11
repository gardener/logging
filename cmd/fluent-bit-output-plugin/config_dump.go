// Copyright 2025 SPDX-FileCopyrightText: Contributors to the Gardener project
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/go-logr/logr"
)

// dumpConfiguration logs the complete plugin configuration at debug level (V(1)).
// It walks v's fields recursively, using mapstructure tags to decide how to log each one:
//   - ",squash" → recurse inline (flattened into the same log level)
//   - "-" on a pointer → log "<name>: configured" when non-nil, skip when nil
//   - "-" on any other type → log "<name>: <value>"
//   - normal tag → log "<tagName>: <value>"
func dumpConfiguration(cfg reflect.Value, logger logr.Logger) {
	t := cfg.Type()
	for i := range t.NumField() {
		field := t.Field(i)
		fval := cfg.Field(i)

		tag := field.Tag.Get("mapstructure")
		tagName, opts, _ := strings.Cut(tag, ",")

		switch {
		case opts == "squash":
			dumpConfiguration(fval, logger)

		case tagName == "-" && field.Type.Kind() == reflect.Pointer:
			if !fval.IsNil() {
				logger.Info("[flb-go]", field.Name, "configured")
			}

		default:
			name := tagName
			if name == "" || name == "-" {
				name = field.Name
			}
			logger.Info("[flb-go]", name, fmt.Sprintf("%+v", fval.Interface()))
		}
	}
}
