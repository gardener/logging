/*
This file was copied from the grafana/loki project
https://github.com/grafana/loki/blob/v1.6.0/cmd/fluent-bit/loki.go

Modifications Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved.
*/

package lokiplugin

import (
	"fmt"
	"regexp"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/prometheus/common/model"
	"k8s.io/client-go/tools/cache"

	bufferedclient "github.com/gardener/logging/fluent-bit-to-loki/pkg/client"
	"github.com/gardener/logging/fluent-bit-to-loki/pkg/config"
	controller "github.com/gardener/logging/fluent-bit-to-loki/pkg/controller"
	lokiclient "github.com/grafana/loki/pkg/promtail/client"
)

// Loki plugin interface
type Loki interface {
	SendRecord(r map[interface{}]interface{}, ts time.Time) error
	Close()
}

type loki struct {
	cfg               *config.Config
	defaultClient     lokiclient.Client
	dynamicHostRegexp *regexp.Regexp
	controller        controller.Controller
	logger            log.Logger
}

// NewPlugin returns Loki output plugin
func NewPlugin(informer cache.SharedIndexInformer, cfg *config.Config, logger log.Logger) (Loki, error) {
	var dynamicHostRegexp *regexp.Regexp
	var ctl controller.Controller

	defaultLokiClient, err := bufferedclient.NewClient(cfg, logger)
	if err != nil {
		return nil, err
	}

	if cfg.DynamicHostPath != nil {
		dynamicHostRegexp = regexp.MustCompile(cfg.DynamicHostRegex)
		ctl, err = controller.NewController(informer, cfg, logger)
		if err != nil {
			return nil, err
		}
	}

	return &loki{
		cfg:               cfg,
		defaultClient:     defaultLokiClient,
		dynamicHostRegexp: dynamicHostRegexp,
		controller:        ctl,
		logger:            logger,
	}, nil
}

// sendRecord send fluentbit records to loki as an entry.
func (l *loki) SendRecord(r map[interface{}]interface{}, ts time.Time) error {
	start := time.Now()
	records := toStringMap(r)
	level.Debug(l.logger).Log("msg", "processing records", "records", fmt.Sprintf("%+v", records))
	lbs := model.LabelSet{}
	if l.cfg.AutoKubernetesLabels {
		err := autoLabels(records, lbs)
		if err != nil {
			level.Error(l.logger).Log("msg", err.Error(), "records", fmt.Sprintf("%+v", records))
		}
	}

	if l.cfg.LabelMap != nil {
		mapLabels(records, l.cfg.LabelMap, lbs)
	} else {
		lbs = extractLabels(records, l.cfg.LabelKeys)
	}

	dynamicHostName := getDynamicHostName(records, l.cfg.DynamicHostPath)

	removeKeys(records, append(l.cfg.LabelKeys, l.cfg.RemoveKeys...))
	if len(records) == 0 {
		return nil
	}

	client := l.getClient(dynamicHostName)

	if client == nil {
		level.Debug(l.logger).Log("host", dynamicHostName, "issue", "could_not_find_client")
		return nil
	}

	if l.cfg.DropSingleKey && len(records) == 1 {
		for _, v := range records {
			return l.send(client, lbs, ts, fmt.Sprintf("%v", v), start)
		}
	}

	line, err := createLine(records, l.cfg.LineFormat)
	if err != nil {
		return fmt.Errorf("error creating line: %v", err)
	}

	err = l.send(client, lbs, ts, line, start)
	if err != nil {
		level.Error(l.logger).Log("msg", "error sending record to Loki", "error", err)
	}

	return err
}

func (l *loki) Close() {
	l.defaultClient.Stop()
	if l.controller != nil {
		l.controller.Stop()
	}
}

func (l *loki) getClient(dynamicHosName string) lokiclient.Client {
	if l.isDynamicHost(dynamicHosName) && l.controller != nil {
		return l.controller.GetClient(dynamicHosName)
	}

	return l.defaultClient
}

func (l *loki) isDynamicHost(dynamicHostName string) bool {
	return dynamicHostName != "" &&
		l.dynamicHostRegexp != nil &&
		l.dynamicHostRegexp.Match([]byte(dynamicHostName))
}

func (l *loki) send(client lokiclient.Client, lbs model.LabelSet, ts time.Time, line string, startOfSendind time.Time) error {
	elapsedBeforeSend := time.Since(startOfSendind)
	level.Debug(l.logger).Log("Log-Processing-elapsed ", elapsedBeforeSend.String(), "Stream", lbs)

	err := client.Handle(lbs, ts, line)

	if err == nil {
		elapsedAfterSend := time.Since(startOfSendind)
		level.Debug(l.logger).Log("Log-Sending-elapsed", elapsedAfterSend.String(), "Stream", lbs)
	}

	return err
}
