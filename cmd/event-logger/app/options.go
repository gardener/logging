// Copyright (c) 2018 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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

package app

import (
	"errors"
	"flag"

	"github.com/gardener/gardener/pkg/logger"

	"github.com/gardener/logging/pkg/events"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/runtime"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/component-base/version"
	"k8s.io/component-base/version/verflag"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
)

// NewCommandStartGardenerEventLogger creates a *cobra.Command object with default parameters.
func NewCommandStartGardenerEventLogger() *cobra.Command {
	opts := NewOptions()

	cmd := &cobra.Command{
		Use:   "gardener-event-logger",
		Short: "Launch the Gardener Event Logger",
		Long:  "Launch the Gardener Event Logger",
		RunE: func(c *cobra.Command, args []string) error {
			verflag.PrintAndExitIfRequested()

			if err := opts.Validate(); err != nil {
				return err
			}

			stopCh := signals.SetupSignalHandler()
			return opts.Run(stopCh)
		},
		SilenceUsage: true,
	}

	flags := cmd.Flags()
	verflag.AddFlags(flags)
	opts.AddFlags(flags)
	// has to be after opts.AddFlags because controller-runtime registers "--kubeconfig" on flag.CommandLine
	// see https://github.com/kubernetes-sigs/controller-runtime/blob/v0.8.0/pkg/client/config/config.go#L38
	flags.AddGoFlagSet(flag.CommandLine)

	return cmd
}

// Options has all the context and parameters needed to run a Gardener Events logger.
type Options struct {
	SeedEventWatcher  events.SeedOptions
	ShootEventWatcher events.ShootOptions
	//Logs              *logs.KlogWriter
}

// NewOptions returns a new Options object.
func NewOptions() *Options {
	o := &Options{
		SeedEventWatcher:  events.SeedOptions{},
		ShootEventWatcher: events.ShootOptions{},
		//Logs:              logs.KlogWriter,
	}

	return o
}

// AddFlags adds all flags to the given FlagSet.
func (o *Options) AddFlags(flags *pflag.FlagSet) {
	o.SeedEventWatcher.AddFlags(flags)
	o.ShootEventWatcher.AddFlags(flags)
	//o.Logs.AddFlags(flags)
}

// Validate validates all the required options.
func (o *Options) Validate() error {
	var errs []error
	errs = append(errs, o.SeedEventWatcher.Validate()...)
	errs = append(errs, o.ShootEventWatcher.Validate()...)

	// Activate logging as soon as possible
	// if err := o.Logs.ValidateAndApply(); err != nil {
	// 	return err
	// }

	return utilerrors.NewAggregate(errs)
}

func (o *Options) config(seedKubeAPIServerConfig *rest.Config, seedKubeClient *kubernetes.Clientset, shootKubeAPIServerConfig *rest.Config, shootKubeClient *kubernetes.Clientset) (*events.GardenerEventWatcherConfig, error) {
	config := &events.GardenerEventWatcherConfig{}

	for _, namespace := range o.SeedEventWatcher.Namespaces {
		config.SeedKubeInformerFactories = append(config.SeedKubeInformerFactories,
			kubeinformers.NewFilteredSharedInformerFactory(seedKubeClient, shootKubeAPIServerConfig.Timeout, namespace, nil),
		)
	}

	for _, namespace := range o.ShootEventWatcher.Namespaces {
		config.ShootKubeInformerFactories = append(config.ShootKubeInformerFactories,
			kubeinformers.NewFilteredSharedInformerFactory(shootKubeClient, shootKubeAPIServerConfig.Timeout, namespace, nil),
		)
	}

	if err := o.ApplyTo(config); err != nil {
		return nil, err
	}

	return config, nil
}

// Run runs gardener-apiserver with the given Options.
func (o *Options) Run(stopCh <-chan struct{}) error {
	logger := logger.NewLogger("")
	logger.Info("Starting Gardener Event Logger...")
	logger.Infof("Version: %+v", version.Get())

	// Create clientset for the native Kubernetes API group
	// Use remote kubeconfig file (if set) or in-cluster config to create a new Kubernetes client for the native Kubernetes API groups
	seedKubeAPIServerConfig, err := clientcmd.BuildConfigFromFlags("", o.SeedEventWatcher.Kubeconfig)
	if err != nil {
		return err
	}

	protobufConfig := *seedKubeAPIServerConfig
	if protobufConfig.ContentType == "" {
		protobufConfig.ContentType = runtime.ContentTypeProtobuf
	}

	// seed kube client
	seedKubeClient, err := kubernetes.NewForConfig(&protobufConfig)
	if err != nil {
		return err
	}

	var shootKubeClient *kubernetes.Clientset
	var shootKubeAPIServerConfig *rest.Config

	if o.ShootEventWatcher.Kubeconfig != "" {
		if o.ShootEventWatcher.Kubeconfig == "inClusterConfig" {
			return errors.New("inClusterConfig cannot be used for shoot kube client")
		}

		shootKubeAPIServerConfig, err = clientcmd.BuildConfigFromFlags("", o.ShootEventWatcher.Kubeconfig)
		if err != nil {
			return err
		}

		protobufConfig := *shootKubeAPIServerConfig
		if protobufConfig.ContentType == "" {
			protobufConfig.ContentType = runtime.ContentTypeProtobuf
		}

		// shoot kube client
		shootKubeClient, err = kubernetes.NewForConfig(&protobufConfig)
		if err != nil {
			return err
		}
	}

	config, err := o.config(seedKubeAPIServerConfig, seedKubeClient, shootKubeAPIServerConfig, shootKubeClient)
	if err != nil {
		return err
	}

	eventLogger := config.New()
	if err != nil {
		return err
	}

	eventLogger.Run(stopCh)

	return nil
}

// ApplyTo applies the options to the given config.
func (o *Options) ApplyTo(config *events.GardenerEventWatcherConfig) error {

	if err := o.SeedEventWatcher.ApplyTo(&config.SeedEventWatcherConfig); err != nil {
		return err
	}
	if err := o.ShootEventWatcher.ApplyTo(&config.ShootEventWatcherConfig); err != nil {
		return err
	}
	return nil
}
