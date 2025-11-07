// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"errors"
	"time"

	"github.com/credativ/vali/pkg/logproto"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/prometheus/common/model"

	"github.com/gardener/logging/pkg/config"
)

const (
	minWaitCheckFrequency       = 10 * time.Millisecond
	waitCheckFrequencyDelimiter = 10
)

// Entry represent a Vali log record.
type Entry struct {
	Labels model.LabelSet
	logproto.Entry
}

// NewValiClientFunc returns a OutputClient on success.
type NewValiClientFunc func(cfg config.Config, logger log.Logger) (OutputClient, error)

// ErrInvalidLabelType is returned when the provided labels are not of type model.LabelSet
var ErrInvalidLabelType = errors.New("labels are not a valid model.LabelSet")

// Options for creating a Vali client
type valiOptions struct {
	// PreservedLabels is the labels to preserve
	PreservedLabels model.LabelSet
}

// valiPreservedLabels implements Options for Vali preserved labels
type valiPreservedLabels model.LabelSet

func (v valiPreservedLabels) apply(opts *clientOptions) error {
	if opts.vali == nil {
		opts.vali = &valiOptions{}
	}
	opts.vali.PreservedLabels = model.LabelSet(v)

	return nil
}

// WithPreservedLabels creates a functional option for preserved labels (Vali only)
func WithPreservedLabels(labels model.LabelSet) Options {
	return valiPreservedLabels(labels)
}

func newValiClient(cfg config.Config, logger log.Logger, options valiOptions) (OutputClient, error) {
	var ncf NewValiClientFunc

	if cfg.ClientConfig.TestingClient == nil {
		ncf = func(c config.Config, l log.Logger) (OutputClient, error) {
			return NewPromtailClient(c.ClientConfig.CredativValiConfig, l)
		}
	} else {
		ncf = func(c config.Config, _ log.Logger) (OutputClient, error) {
			return newTestingPromtailClient(cfg.ClientConfig.TestingClient, c.ClientConfig.CredativValiConfig)
		}
	}

	// When label processing is done the sorting client could be used.
	if cfg.ClientConfig.SortByTimestamp {
		tempNCF := ncf
		ncf = func(c config.Config, l log.Logger) (OutputClient, error) {
			return NewSortedClientDecorator(c, tempNCF, l)
		}
	}

	// The last wrapper which process labels should be the pack client.
	// After the pack labels which are needed for the record processing
	// cloud be packed and thus no long existing
	if options.PreservedLabels != nil {
		tempNCF := ncf
		ncf = func(c config.Config, l log.Logger) (OutputClient, error) {
			return NewPackClientDecorator(c, tempNCF, l)
		}
	}

	if cfg.ClientConfig.BufferConfig.Buffer {
		tempNCF := ncf
		ncf = func(c config.Config, l log.Logger) (OutputClient, error) {
			return NewBufferDecorator(c, tempNCF, l)
		}
	}

	outputClient, err := ncf(cfg, logger)
	if err != nil {
		return nil, err
	}
	_ = level.Debug(logger).Log("msg", "client created", "url", outputClient.GetEndPoint())

	return outputClient, nil
}

func newValiTailClient(cfg config.Config, newClient NewValiClientFunc, logger log.Logger) (OutputClient, error) {
	if newClient != nil {
		return newClient(cfg, logger)
	}

	return NewPromtailClient(cfg.ClientConfig.CredativValiConfig, logger)
}
