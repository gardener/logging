// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

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
