// Copyright 2025 SPDX-FileCopyrightText: Copyright Contributors to the Gardener project
// SPDX-License-Identifier: Apache-2.0

package otlp

import "errors"

// ErrThrottled is returned when an OTLP client is rate-limited.
var ErrThrottled = errors.New("client throttled: rate limit exceeded")
