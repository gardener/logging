// Copyright (c) 2022 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package events

import (
	"github.com/spf13/pflag"
)

func (o *Options) Validate() []error {
	//TODO: vlvasilev implement me
	errors := []error{}
	errors = append(errors, nil)
	return errors
}

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

func (o *ShootOptions) ApplyTo(config *EventWatcherConfig) error {
	return o.Options.ApplyTo(config)
}
