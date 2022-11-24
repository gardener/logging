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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EventWatcherConfig is a configuration fot the event watcher.
type EventWatcherConfig struct {
	// Kubeconfig is the path to the kubeconfig file
	Kubeconfig string //Do I need this field?
	// Namespaces to watch for events in.
	Namespaces []string
}

// Options has all the context and parameters needed to run a Gardener Event Logger.
type Options struct {
	// Kubeconfig is the path to the kubeconfig file
	Kubeconfig string
	// Namespaces to watch for events in.
	Namespaces []string
}

// SeedOptions has all the context and parameters needed to run a Gardener Event Logger in the seed cluster
type SeedOptions struct {
	// Options has all the context and parameters needed to run a Gardener Event Logger.
	Options
}

// ShootOptions has all the context and parameters needed to run a Gardener Event Logger in the shoot cluster
type ShootOptions struct {
	// Options has all the context and parameters needed to run a Gardener Event Logger.
	Options
}

type event struct {
	Origin         string           `json:"origin" protobuf:"bytes,1,name=origin"`
	Namespace      string           `json:"namespace" protobuf:"bytes,2,name=namespace"`
	Type           string           `json:"type,omitempty" protobuf:"bytes,3,opt,name=type"`
	Count          int32            `json:"count,omitempty" protobuf:"varint,4,opt,name=count"`
	EventTime      metav1.MicroTime `json:"eventTime,omitempty" protobuf:"bytes,5,opt,name=eventTime"`
	FirstTimestamp metav1.Time      `json:"firstTimestamp,omitempty" protobuf:"bytes,6,opt,name=firstTimestamp"`
	LastTimestamp  metav1.Time      `json:"lastTimestamp,omitempty" protobuf:"bytes,7,opt,name=lastTimestamp"`
	Reason         string           `json:"reason,omitempty" protobuf:"bytes,8,opt,name=reason"`
	Object         string           `json:"object" protobuf:"bytes,9,opt,name=object"`
	Message        string           `json:"_entry,omitempty" protobuf:"bytes,10,opt,name=_entry"`
	Source         string           `json:"source,omitempty" protobuf:"bytes,11,opt,name=source"`
	SourceHost     string           `json:"sourceHost,omitempty" protobuf:"bytes,12,opt,name=sourceHost"`
}
