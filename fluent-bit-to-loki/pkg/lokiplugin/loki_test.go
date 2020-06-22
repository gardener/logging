package lokiplugin

import (
	"os"
	"regexp"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"github.com/weaveworks/common/logging"

	"github.com/gardener/logging/fluent-bit-to-loki/pkg/config"

	lokiclient "github.com/grafana/loki/pkg/promtail/client"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/prometheus/common/model"
)

type entry struct {
	lbs  model.LabelSet
	line string
	ts   time.Time
}

type recorder struct {
	*entry
}

func (r *recorder) Handle(labels model.LabelSet, time time.Time, e string) error {
	r.entry = &entry{
		labels,
		e,
		time,
	}
	return nil
}

func (r *recorder) toEntry() *entry { return r.entry }

func (r *recorder) Stop() {}

type sendRecordArgs struct {
	cfg     *config.Config
	record  map[interface{}]interface{}
	want    *entry
	wantErr bool
}

type fakeLokiClient struct{}

func (c *fakeLokiClient) Handle(labels model.LabelSet, time time.Time, entry string) error {
	return nil
}

func (c *fakeLokiClient) Stop() {}

type fakeController struct {
	clients map[string]lokiclient.Client
}

func (ctl *fakeController) GetClient(name string) lokiclient.Client {
	if client, ok := ctl.clients[name]; ok {
		return client
	}
	return nil
}

func (ctl *fakeController) Stop() {}

var (
	now      = time.Now()
	logLevel logging.Level
	logger   = log.NewLogfmtLogger(log.NewSyncWriter(os.Stderr))
)

var _ = Describe("Loki plugin", func() {
	var (
		simpleRecordFixture = map[interface{}]interface{}{
			"foo":   "bar",
			"bar":   500,
			"error": make(chan struct{}),
		}
		mapRecordFixture = map[interface{}]interface{}{
			// lots of key/value pairs in map to increase chances of test hitting in case of unsorted map marshalling
			"A": "A",
			"B": "B",
			"C": "C",
			"D": "D",
			"E": "E",
			"F": "F",
			"G": "G",
			"H": "H",
		}

		byteArrayRecordFixture = map[interface{}]interface{}{
			"label": "label",
			"outer": []byte("foo"),
			"map": map[interface{}]interface{}{
				"inner": []byte("bar"),
			},
		}

		mixedTypesRecordFixture = map[interface{}]interface{}{
			"label": "label",
			"int":   42,
			"float": 42.42,
			"array": []interface{}{42, 42.42, "foo"},
			"map": map[interface{}]interface{}{
				"nested": map[interface{}]interface{}{
					"foo":     "bar",
					"invalid": []byte("a\xc5z"),
				},
			},
		}
	)

	logLevel.Set("info")
	logger = level.NewFilter(logger, logLevel.Gokit)
	logger = log.With(logger, "caller", log.Caller(3))

	DescribeTable("#SendRecord",
		func(args sendRecordArgs) {
			rec := &recorder{}
			l := &loki{
				cfg:           args.cfg,
				defaultClient: rec,
				logger:        logger,
			}
			err := l.SendRecord(args.record, now)
			if args.wantErr {
				Expect(err).To(HaveOccurred())
				return
			}
			Expect(err).ToNot(HaveOccurred())
			got := rec.toEntry()
			Expect(got).To(Equal(args.want))
		},
		Entry("map to JSON",
			sendRecordArgs{
				cfg:     &config.Config{LabelKeys: []string{"A"}, LineFormat: config.JSONFormat},
				record:  mapRecordFixture,
				want:    &entry{model.LabelSet{"A": "A"}, `{"B":"B","C":"C","D":"D","E":"E","F":"F","G":"G","H":"H"}`, now},
				wantErr: false,
			}),
		Entry("map to kvPairFormat",
			sendRecordArgs{
				cfg:     &config.Config{LabelKeys: []string{"A"}, LineFormat: config.KvPairFormat},
				record:  mapRecordFixture,
				want:    &entry{model.LabelSet{"A": "A"}, `B=B C=C D=D E=E F=F G=G H=H`, now},
				wantErr: false,
			}),
		Entry(
			"not enough records",
			sendRecordArgs{
				cfg:     &config.Config{LabelKeys: []string{"foo"}, LineFormat: config.JSONFormat, RemoveKeys: []string{"bar", "error"}},
				record:  simpleRecordFixture,
				want:    nil,
				wantErr: false,
			}),
		Entry("labels",
			sendRecordArgs{
				cfg:     &config.Config{LabelKeys: []string{"bar", "fake"}, LineFormat: config.JSONFormat, RemoveKeys: []string{"fuzz", "error"}},
				record:  simpleRecordFixture,
				want:    &entry{model.LabelSet{"bar": "500"}, `{"foo":"bar"}`, now},
				wantErr: false,
			}),
		Entry("remove key",
			sendRecordArgs{
				cfg:     &config.Config{LabelKeys: []string{"fake"}, LineFormat: config.JSONFormat, RemoveKeys: []string{"foo", "error", "fake"}},
				record:  simpleRecordFixture,
				want:    &entry{model.LabelSet{}, `{"bar":500}`, now},
				wantErr: false,
			}),
		Entry("error",
			sendRecordArgs{
				cfg:     &config.Config{LabelKeys: []string{"fake"}, LineFormat: config.JSONFormat, RemoveKeys: []string{"foo"}},
				record:  simpleRecordFixture,
				want:    nil,
				wantErr: true,
			}),
		Entry("key value",
			sendRecordArgs{
				cfg:     &config.Config{LabelKeys: []string{"fake"}, LineFormat: config.KvPairFormat, RemoveKeys: []string{"foo", "error", "fake"}},
				record:  simpleRecordFixture,
				want:    &entry{model.LabelSet{}, `bar=500`, now},
				wantErr: false,
			}),
		Entry("single",
			sendRecordArgs{
				cfg:     &config.Config{LabelKeys: []string{"fake"}, DropSingleKey: true, LineFormat: config.KvPairFormat, RemoveKeys: []string{"foo", "error", "fake"}},
				record:  simpleRecordFixture,
				want:    &entry{model.LabelSet{}, `500`, now},
				wantErr: false,
			}),
		Entry("labelmap",
			sendRecordArgs{
				cfg:     &config.Config{LabelMap: map[string]interface{}{"bar": "other"}, LineFormat: config.JSONFormat, RemoveKeys: []string{"bar", "error"}},
				record:  simpleRecordFixture,
				want:    &entry{model.LabelSet{"other": "500"}, `{"foo":"bar"}`, now},
				wantErr: false,
			}),
		Entry("byte array",
			sendRecordArgs{
				cfg:     &config.Config{LabelKeys: []string{"label"}, LineFormat: config.JSONFormat},
				record:  byteArrayRecordFixture,
				want:    &entry{model.LabelSet{"label": "label"}, `{"map":{"inner":"bar"},"outer":"foo"}`, now},
				wantErr: false,
			}),
		Entry("mixed types",
			sendRecordArgs{
				cfg:     &config.Config{LabelKeys: []string{"label"}, LineFormat: config.JSONFormat},
				record:  mixedTypesRecordFixture,
				want:    &entry{model.LabelSet{"label": "label"}, `{"array":[42,42.42,"foo"],"float":42.42,"int":42,"map":{"nested":{"foo":"bar","invalid":"a\ufffdz"}}}`, now},
				wantErr: false,
			},
		),
	)

	Describe("#getClient", func() {
		fc := fakeController{
			clients: map[string]lokiclient.Client{
				"shoot--dev--test1": &fakeLokiClient{},
				"shoot--dev--test2": &fakeLokiClient{},
			},
		}
		lokiplug := loki{
			dynamicHostRegexp: regexp.MustCompile("shoot--.*"),
			defaultClient:     &fakeLokiClient{},
			controller:        &fc,
		}

		type getClientArgs struct {
			dynamicHostName string
			expectToExists  bool
		}

		DescribeTable("#getClient",
			func(args getClientArgs) {
				c := lokiplug.getClient(args.dynamicHostName)
				if args.expectToExists {
					Expect(c).ToNot(BeNil())
				} else {
					Expect(c).To(BeNil())
				}
			},
			Entry("Not existing host",
				getClientArgs{
					dynamicHostName: "shoot--dev--missing",
					expectToExists:  false,
				}),
			Entry("Existing host",
				getClientArgs{
					dynamicHostName: "shoot--dev--test1",
					expectToExists:  true,
				}),
			Entry("Empty host",
				getClientArgs{
					dynamicHostName: "",
					expectToExists:  true,
				}),
			Entry("Not dynamic host",
				getClientArgs{
					dynamicHostName: "kube-system",
					expectToExists:  true,
				}),
		)
	})
})
