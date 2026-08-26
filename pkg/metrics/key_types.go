// Copyright 2025 SPDX-FileCopyrightText: Copyright Contributors to the Gardener project
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
	ErrorInvalidRecordKey             = "InvalidRecordKey"
	MissingMetadataType               = "Kubernetes"
)
