// Copyright (c) 2021 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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

package client

import (
	"time"

	"github.com/gardener/logging/pkg/types"
	"github.com/grafana/loki/pkg/promtail/client"
	. "github.com/onsi/ginkgo"
	ginkotable "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"github.com/prometheus/common/model"
)

var _ = Describe("Multi Tenant Client", func() {
	var (
		fakeClient *FakeLokiClient
		mtc        types.LokiClient
	)

	BeforeEach(func() {
		fakeClient = &FakeLokiClient{}
		mtc = NewMultiTenantClientWrapper(fakeClient, false)
	})

	Describe("#NewMultiTenantClientWrapper", func() {

		It("Should return wrapper client which does not copy the label set before sending it to the wrapped client", func() {
			mtc := NewMultiTenantClientWrapper(fakeClient, false)
			Expect(mtc).NotTo(BeNil())
			Expect(mtc.(*multiTenantClient).lokiclient).To(Equal(fakeClient))
			Expect(mtc.(*multiTenantClient).copyLabelSet).To(BeFalse())
		})

		It("Should return wrapper client which copies the label set before sending it to the wrapped client", func() {
			mtc := NewMultiTenantClientWrapper(fakeClient, true)
			Expect(mtc).NotTo(BeNil())
			Expect(mtc.(*multiTenantClient).lokiclient).To(Equal(fakeClient))
			Expect(mtc.(*multiTenantClient).copyLabelSet).To(BeTrue())
		})
	})

	type handleArgs struct {
		ls            model.LabelSet
		t             time.Time
		s             string
		wantedTenants []model.LabelValue
	}
	ginkotable.DescribeTable("#Handle", func(args handleArgs) {
		err := mtc.Handle(args.ls, args.t, args.s)
		Expect(err).ToNot(HaveOccurred())
		Expect(len(fakeClient.Entries) > 0).To(BeTrue())
		var gotTenants []model.LabelValue
		for _, entry := range fakeClient.Entries {
			// __gardener_multitenant_id__ should be removed after Handle() call
			_, ok := entry.Labels[MultiTenantClientLabel]
			Expect(ok).To(BeFalse())
			// Each tenant in the MultiTenantClientLabel should be transferred to __tenant_id__
			if tenant, ok := entry.Labels[client.ReservedLabelTenantID]; ok {
				gotTenants = append(gotTenants, tenant)
			}
		}
		// Check if all multitenants are parsed and used for forwarding to the wrapped Handle func
		Expect(len(args.wantedTenants)).To(Equal(len(gotTenants)))
		for _, tenant := range args.wantedTenants {
			Expect(tenant).To(BeElementOf(gotTenants))
		}
		// Sanity check if the timestamp and the messages re the same
		for _, entry := range fakeClient.Entries {
			Expect(entry.Timestamp).To(Equal(args.t))
			Expect(entry.Line).To(Equal(args.s))
		}
	},
		ginkotable.Entry("Handle record without reserved labels", handleArgs{
			ls:            model.LabelSet{"hostname": "test"},
			t:             time.Now(),
			s:             "test1",
			wantedTenants: []model.LabelValue{},
		}),
		ginkotable.Entry("Handle record without __gardener_multitenant_id__ reserved label", handleArgs{
			ls:            model.LabelSet{"hostname": "test", "__tenant_id__": "user"},
			t:             time.Now(),
			s:             "test1",
			wantedTenants: []model.LabelValue{"user"},
		}),
		ginkotable.Entry("Handle record with __gardener_multitenant_id__ reserved label. Separator \"; \"", handleArgs{
			ls:            model.LabelSet{"hostname": "test", "__gardener_multitenant_id__": "operator; user"},
			t:             time.Now(),
			s:             "test1",
			wantedTenants: []model.LabelValue{"operator", "user"},
		}),
		ginkotable.Entry("Handle record with __gardener_multitenant_id__ reserved label. Separator \";\"", handleArgs{
			ls:            model.LabelSet{"hostname": "test", "__gardener_multitenant_id__": "operator;user"},
			t:             time.Now(),
			s:             "test1",
			wantedTenants: []model.LabelValue{"operator", "user"},
		}),
		ginkotable.Entry("Handle record with __gardener_multitenant_id__ reserved label. Separator \" ; \" and leading and trailing spaces", handleArgs{
			ls:            model.LabelSet{"hostname": "test", "__gardener_multitenant_id__": "  operator ; user  "},
			t:             time.Now(),
			s:             "test1",
			wantedTenants: []model.LabelValue{"operator", "user"},
		}),
		ginkotable.Entry("Handle record with __gardener_multitenant_id__ reserved label with one empty. Separator \" ; \" and leading and trailing spaces.", handleArgs{
			ls:            model.LabelSet{"hostname": "test", "__gardener_multitenant_id__": "  operator ; ; user  "},
			t:             time.Now(),
			s:             "test1",
			wantedTenants: []model.LabelValue{"operator", "user"},
		}),
		ginkotable.Entry("Handle record with __gardener_multitenant_id__ and __tenant_id__ reserved labels.", handleArgs{
			ls:            model.LabelSet{"hostname": "test", "__tenant_id__": "pinokio", "__gardener_multitenant_id__": "operator; user"},
			t:             time.Now(),
			s:             "test1",
			wantedTenants: []model.LabelValue{"operator", "user"},
		}),
	)

	Describe("#Stop", func() {
		It("should stop", func() {
			Expect(mtc.(*multiTenantClient).lokiclient.(*FakeLokiClient).IsGracefullyStopped).To(BeFalse())
			Expect(mtc.(*multiTenantClient).lokiclient.(*FakeLokiClient).IsStopped).To(BeFalse())
			mtc.Stop()
			Expect(mtc.(*multiTenantClient).lokiclient.(*FakeLokiClient).IsGracefullyStopped).To(BeFalse())
			Expect(mtc.(*multiTenantClient).lokiclient.(*FakeLokiClient).IsStopped).To(BeTrue())
		})
	})

	Describe("#StopWait", func() {
		It("should stop", func() {
			Expect(mtc.(*multiTenantClient).lokiclient.(*FakeLokiClient).IsGracefullyStopped).To(BeFalse())
			Expect(mtc.(*multiTenantClient).lokiclient.(*FakeLokiClient).IsStopped).To(BeFalse())
			mtc.StopWait()
			Expect(mtc.(*multiTenantClient).lokiclient.(*FakeLokiClient).IsGracefullyStopped).To(BeTrue())
			Expect(mtc.(*multiTenantClient).lokiclient.(*FakeLokiClient).IsStopped).To(BeFalse())
		})
	})

})
