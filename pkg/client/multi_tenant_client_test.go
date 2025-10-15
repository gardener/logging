// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package client_test

import (
	"os"
	"time"

	valitailclient "github.com/credativ/vali/pkg/valitail/client"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	ginkgov2 "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"github.com/prometheus/common/model"
	"github.com/weaveworks/common/logging"

	"github.com/gardener/logging/pkg/client"
	"github.com/gardener/logging/pkg/config"
)

var _ = ginkgov2.Describe("Multi Tenant Client", func() {
	var (
		fakeClient *client.FakeValiClient
		mtc        client.OutputClient
	)

	ginkgov2.BeforeEach(func() {
		var err error
		fakeClient = &client.FakeValiClient{}
		var infoLogLevel logging.Level
		_ = infoLogLevel.Set("info")

		mtc, err = client.NewMultiTenantClientDecorator(config.Config{},
			func(_ config.Config, _ log.Logger) (client.OutputClient, error) {
				return fakeClient, nil
			},
			level.NewFilter(log.NewLogfmtLogger(log.NewSyncWriter(os.Stderr)), infoLogLevel.Gokit))
		gomega.Expect(err).ToNot(gomega.HaveOccurred())
		gomega.Expect(mtc).NotTo(gomega.BeNil())
	})

	type handleArgs struct {
		ls            model.LabelSet
		t             time.Time
		s             string
		wantedTenants []model.LabelValue
	}
	ginkgov2.DescribeTable("#Handle", func(args handleArgs) {
		err := mtc.Handle(args.ls, args.t, args.s)
		gomega.Expect(err).ToNot(gomega.HaveOccurred())
		gomega.Expect(len(fakeClient.Entries) > 0).To(gomega.BeTrue())
		var gotTenants []model.LabelValue
		for _, entry := range fakeClient.Entries {
			// __gardener_multitenant_id__ should be removed after Handle() call
			_, ok := entry.Labels[client.MultiTenantClientLabel]
			gomega.Expect(ok).To(gomega.BeFalse())
			// Each tenant in the MultiTenantClientLabel should be transferred to __tenant_id__
			if tenant, ok := entry.Labels[valitailclient.ReservedLabelTenantID]; ok {
				gotTenants = append(gotTenants, tenant)
			}
		}
		// Check if all multitenants are parsed and used for forwarding to the wrapped Handle func
		gomega.Expect(len(args.wantedTenants)).To(gomega.Equal(len(gotTenants)))
		for _, tenant := range args.wantedTenants {
			gomega.Expect(tenant).To(gomega.BeElementOf(gotTenants))
		}
		// Sanity check if the timestamp and the messages re the same
		for _, entry := range fakeClient.Entries {
			gomega.Expect(entry.Timestamp).To(gomega.Equal(args.t))
			gomega.Expect(entry.Line).To(gomega.Equal(args.s))
		}
	},
		ginkgov2.Entry("Handle record without reserved labels", handleArgs{
			ls:            model.LabelSet{"hostname": "test"},
			t:             time.Now(),
			s:             "test1",
			wantedTenants: []model.LabelValue{},
		}),
		ginkgov2.Entry("Handle record without __gardener_multitenant_id__ reserved label", handleArgs{
			ls:            model.LabelSet{"hostname": "test", "__tenant_id__": "user"},
			t:             time.Now(),
			s:             "test1",
			wantedTenants: []model.LabelValue{"user"},
		}),
		ginkgov2.Entry("Handle record with __gardener_multitenant_id__ reserved label. Separator \"; \"", handleArgs{
			ls:            model.LabelSet{"hostname": "test", "__gardener_multitenant_id__": "operator; user"},
			t:             time.Now(),
			s:             "test1",
			wantedTenants: []model.LabelValue{"operator", "user"},
		}),
		ginkgov2.Entry("Handle record with __gardener_multitenant_id__ reserved label. Separator \";\"", handleArgs{
			ls:            model.LabelSet{"hostname": "test", "__gardener_multitenant_id__": "operator;user"},
			t:             time.Now(),
			s:             "test1",
			wantedTenants: []model.LabelValue{"operator", "user"},
		}),
		ginkgov2.Entry("Handle record with __gardener_multitenant_id__ reserved label. Separator \" ; \" and leading and trailing spaces", handleArgs{
			ls:            model.LabelSet{"hostname": "test", "__gardener_multitenant_id__": "  operator ; user  "},
			t:             time.Now(),
			s:             "test1",
			wantedTenants: []model.LabelValue{"operator", "user"},
		}),
		ginkgov2.Entry("Handle record with __gardener_multitenant_id__ reserved label with one empty. Separator \" ; \" and leading and trailing spaces.", handleArgs{
			ls:            model.LabelSet{"hostname": "test", "__gardener_multitenant_id__": "  operator ; ; user  "},
			t:             time.Now(),
			s:             "test1",
			wantedTenants: []model.LabelValue{"operator", "user"},
		}),
		ginkgov2.Entry("Handle record with __gardener_multitenant_id__ and __tenant_id__ reserved labels.", handleArgs{
			ls:            model.LabelSet{"hostname": "test", "__tenant_id__": "pinokio", "__gardener_multitenant_id__": "operator; user"},
			t:             time.Now(),
			s:             "test1",
			wantedTenants: []model.LabelValue{"operator", "user"},
		}),
	)

	ginkgov2.Describe("#Stop", func() {
		ginkgov2.It("should stop", func() {
			gomega.Expect(fakeClient.IsGracefullyStopped).To(gomega.BeFalse())
			gomega.Expect(fakeClient.IsStopped).To(gomega.BeFalse())
			mtc.Stop()
			gomega.Expect(fakeClient.IsGracefullyStopped).To(gomega.BeFalse())
			gomega.Expect(fakeClient.IsStopped).To(gomega.BeTrue())
		})
	})

	ginkgov2.Describe("#StopWait", func() {
		ginkgov2.It("should stop", func() {
			gomega.Expect(fakeClient.IsGracefullyStopped).To(gomega.BeFalse())
			gomega.Expect(fakeClient.IsStopped).To(gomega.BeFalse())
			mtc.StopWait()
			gomega.Expect(fakeClient.IsGracefullyStopped).To(gomega.BeTrue())
			gomega.Expect(fakeClient.IsStopped).To(gomega.BeFalse())
		})
	})
})

var _ = ginkgov2.Describe("Remove Multi Tenant Client", func() {
	var (
		fakeClient *client.FakeValiClient
		mtc        client.OutputClient
	)

	ginkgov2.BeforeEach(func() {
		var err error
		fakeClient = &client.FakeValiClient{}
		var infoLogLevel logging.Level
		_ = infoLogLevel.Set("info")

		mtc, err = client.NewRemoveMultiTenantIDClientDecorator(config.Config{},
			func(_ config.Config, _ log.Logger) (client.OutputClient, error) {
				return fakeClient, nil
			},
			level.NewFilter(log.NewLogfmtLogger(log.NewSyncWriter(os.Stderr)), infoLogLevel.Gokit))
		gomega.Expect(err).ToNot(gomega.HaveOccurred())
		gomega.Expect(mtc).NotTo(gomega.BeNil())
	})

	type handleArgs struct {
		ls             model.LabelSet
		t              time.Time
		s              string
		wantedLabelSet model.LabelSet
	}
	ginkgov2.DescribeTable("#Handle", func(args handleArgs) {
		err := mtc.Handle(args.ls, args.t, args.s)
		gomega.Expect(err).ToNot(gomega.HaveOccurred())
		gomega.Expect(len(fakeClient.Entries) > 0).To(gomega.BeTrue())

		for _, entry := range fakeClient.Entries {
			gomega.Expect(entry.Labels).To(gomega.Equal(args.wantedLabelSet))
		}
	},
		ginkgov2.Entry("Handle record without __gardener_multitenant_id__ labels.", handleArgs{
			ls:             model.LabelSet{"hostname": "test"},
			t:              time.Now(),
			s:              "test1",
			wantedLabelSet: model.LabelSet{"hostname": "test"},
		}),
		ginkgov2.Entry("Handle record without __gardener_multitenant_id__ label and with __tenant_id__ label.", handleArgs{
			ls:             model.LabelSet{"hostname": "test", "__tenant_id__": "user"},
			t:              time.Now(),
			s:              "test1",
			wantedLabelSet: model.LabelSet{"hostname": "test", "__tenant_id__": "user"},
		}),
		ginkgov2.Entry("Handle record with __gardener_multitenant_id__ reserved label.", handleArgs{
			ls:             model.LabelSet{"hostname": "test", "__gardener_multitenant_id__": "operator; user"},
			t:              time.Now(),
			s:              "test1",
			wantedLabelSet: model.LabelSet{"hostname": "test"},
		}),
		ginkgov2.Entry("Handle record with __gardener_multitenant_id__ reserved label and __tenant_id__ label.", handleArgs{
			ls:             model.LabelSet{"hostname": "test", "__gardener_multitenant_id__": "operator;user", "__tenant_id__": "user"},
			t:              time.Now(),
			s:              "test1",
			wantedLabelSet: model.LabelSet{"hostname": "test", "__tenant_id__": "user"},
		}),
		ginkgov2.Entry("Handle record without labels", handleArgs{
			ls:             model.LabelSet{},
			t:              time.Now(),
			s:              "test1",
			wantedLabelSet: model.LabelSet{},
		}),
		ginkgov2.Entry("Handle record with nil label set", handleArgs{
			ls:             nil,
			t:              time.Now(),
			s:              "test1",
			wantedLabelSet: model.LabelSet{},
		}),
	)

	ginkgov2.Describe("#Stop", func() {
		ginkgov2.It("should stop", func() {
			gomega.Expect(fakeClient.IsGracefullyStopped).To(gomega.BeFalse())
			gomega.Expect(fakeClient.IsStopped).To(gomega.BeFalse())
			mtc.Stop()
			gomega.Expect(fakeClient.IsGracefullyStopped).To(gomega.BeFalse())
			gomega.Expect(fakeClient.IsStopped).To(gomega.BeTrue())
		})
	})

	ginkgov2.Describe("#StopWait", func() {
		ginkgov2.It("should stop", func() {
			gomega.Expect(fakeClient.IsGracefullyStopped).To(gomega.BeFalse())
			gomega.Expect(fakeClient.IsStopped).To(gomega.BeFalse())
			mtc.StopWait()
			gomega.Expect(fakeClient.IsGracefullyStopped).To(gomega.BeTrue())
			gomega.Expect(fakeClient.IsStopped).To(gomega.BeFalse())
		})
	})
})
