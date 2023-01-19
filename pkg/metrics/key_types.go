// Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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

package metrics

// Contants which hold metric types
const (
	ErrorFLBPluginInit                = "FLBPluginInit"
	ErrorNewPlugin                    = "NewPlugin"
	ErrorFLBPluginFlushCtx            = "FLBPluginFlushCtx"
	ErrorDequeuer                     = "Dequeuer"
	ErrorDequeuerNotValidType         = "DequeuerNotValidType"
	ErrorDequeuerSendRecord           = "DequeuerSendRecord"
	ErrorCreateDecoder                = "CreateDecoder"
	ErrorAddFuncNotACluster           = "AddFuncNotACluster"
	ErrorUpdateFuncOldNotACluster     = "UpdateFuncOldNotACluster"
	ErrorUpdateFuncNewNotACluster     = "AddFuncNewNotACluster"
	ErrorDeleteFuncNotAcluster        = "DeleteFuncNotAcluster"
	ErrorFailedToParseURL             = "FailedToParseUrl"
	ErrorCanNotExtractShoot           = "CanNotExtractShoot"
	ErrorFailedToMakeValiClient       = "FailedToMakeValiClient"
	ErrorCanNotExtractMetadataFromTag = "CanNotExtractMetadataFromTag"
	ErrorK8sLabelsNotFound            = "K8sLabelsNotFound"
	ErrorCreateLine                   = "CreateLine"
	ErrorSendRecordToVali             = "SendRecordToVali"

	MissingMetadataType = "Kubernetes"
)
