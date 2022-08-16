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

type EventWatcherConfig struct {
	Kubeconfig string //Do I need this field?
	Namespaces []string
}

// Options has all the context and parameters needed to run a Gardener Event Logger.
type Options struct {
	Kubeconfig string
	Namespaces []string
}

type SeedOptions struct {
	Options
}

type ShootOptions struct {
	Options
}

type event struct {
	Origin         string      `json:"origin" protobuf:"bytes,6,name=origin"`
	Namespace      string      `json:"namespace" protobuf:"bytes,9,name=namespace"`
	Type           string      `json:"type,omitempty" protobuf:"bytes,4,opt,name=type"`
	Count          int32       `json:"count,omitempty" protobuf:"varint,8,opt,name=count"`
	FirstTimestamp metav1.Time `json:"firstTimestamp,omitempty" protobuf:"bytes,6,opt,name=firstTimestamp"`
	LastTimestamp  metav1.Time `json:"lastTimestamp,omitempty" protobuf:"bytes,7,opt,name=lastTimestamp"`
	Reason         string      `json:"reason,omitempty" protobuf:"bytes,3,opt,name=reason"`
	Object         string      `json:"object" protobuf:"bytes,2,opt,name=object"`
	Message        string      `json:"_entry,omitempty" protobuf:"bytes,4,opt,name=_entry"`
	Source         string      `json:"source,omitempty" protobuf:"bytes,5,opt,name=source"`
	SourceHost     string      `json:"sourceHost,omitempty" protobuf:"bytes,2,opt,name=sourceHost"`
}
