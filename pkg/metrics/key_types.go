// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package metrics

// Constants which hold metric types
const (
	ErrorFLBPluginInit                = "FLBPluginInit"
	ErrorNewPlugin                    = "NewPlugin"
	ErrorFLBPluginFlushCtx            = "FLBPluginFlushCtx"
	ErrorEnqueuer                     = "Enqueuer"
	ErrorDequeuer                     = "Dequeuer"
	ErrorDequeuerNotValidType         = "DequeuerNotValidType"
	ErrorDequeuerSendRecord           = "DequeuerSendRecord"
	ErrorFailedToMakeOutputClient     = "FailedToMakeOutputClient"
	ErrorCanNotExtractMetadataFromTag = "CanNotExtractMetadataFromTag"
	ErrorK8sLabelsNotFound            = "K8sLabelsNotFound"
	ErrorCreateLine                   = "CreateLine"
	ErrorSendRecord                   = "SendRecord"
	MissingMetadataType               = "Kubernetes"
)
