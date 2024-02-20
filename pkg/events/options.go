// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package events

import (
	"errors"
	"os"
	"strings"

	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/api/validation"
)

// Validate validates the Options
func (o *Options) Validate() []error {
	allErrors := []error{}
	if o.Kubeconfig != "inClusterConfig" && o.Kubeconfig != "" {
		if _, err := os.Stat(o.Kubeconfig); err != nil {
			allErrors = append(allErrors, err)
		}
	}

	for _, ns := range o.Namespaces {
		if errs := validation.ValidateNamespaceName(ns, false); len(errs) > 0 {
			allErrors = append(allErrors, errors.New(strings.Join(errs, "; ")))
		}
	}

	return allErrors
}

// ApplyTo applies the Options to an EventWatcherConfig
func (o *Options) ApplyTo(config *EventWatcherConfig) error {
	config.Kubeconfig = o.Kubeconfig
	config.Namespaces = o.Namespaces
	return nil
}

// AddFlags adds all flags to the given FlagSet.
func (o *SeedOptions) AddFlags(fs *pflag.FlagSet) {
	if o == nil {
		return
	}

	fs.StringVar(&o.Kubeconfig, "seed-kubeconfig", "", "The kubeconfig for the seed cluster")
	fs.StringSliceVar(&o.Namespaces, "seed-event-namespaces", []string{"kube-system"}, "The namespaces of the seed events")
}

// Validate all flags of the given Options.
func (o *SeedOptions) Validate() []error {
	return o.Options.Validate()
}

// ApplyTo applies the SeedOptions to an EventWatcherConfig
func (o *SeedOptions) ApplyTo(config *EventWatcherConfig) error {
	return o.Options.ApplyTo(config)
}

// AddFlags adds all flags to the given FlagSet.
func (o *ShootOptions) AddFlags(fs *pflag.FlagSet) {
	if o == nil {
		return
	}

	fs.StringVar(&o.Kubeconfig, "shoot-kubeconfig", "", "The kubeconfig for the shoot cluster")
	fs.StringSliceVar(&o.Namespaces, "shoot-event-namespaces", []string{"kube-system", "default"}, "The namespaces of the shoot events")
}

// Validate all flags of the given Options.
func (o *ShootOptions) Validate() []error {
	return o.Options.Validate()
}

// ApplyTo applies the ShootOptions to an EventWatcherConfig
func (o *ShootOptions) ApplyTo(config *EventWatcherConfig) error {
	return o.Options.ApplyTo(config)
}
