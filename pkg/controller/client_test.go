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

package controller

import (
	"os"
	"time"

	"github.com/gardener/logging/pkg/client"
	"github.com/gardener/logging/pkg/config"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/grafana/vali/pkg/logproto"
	"github.com/prometheus/common/model"

	"github.com/weaveworks/common/logging"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = Describe("Controller Client", func() {
	var (
		ctlClient  controllerClient
		logLevel   logging.Level
		_          = logLevel.Set("error")
		logger     = level.NewFilter(log.NewLogfmtLogger(log.NewSyncWriter(os.Stderr)), logLevel.Gokit)
		labels1    = model.LabelSet{model.LabelName("KeyTest1"): model.LabelValue("ValueTest1")}
		labels2    = model.LabelSet{model.LabelName("KeyTest2"): model.LabelValue("ValueTest2")}
		timestamp1 = time.Now()
		timestamp2 = time.Now().Add(time.Second)
		line1      = "testline1"
		line2      = "testline2"
		entry1     = client.Entry{Labels: labels1, Entry: logproto.Entry{Timestamp: timestamp1, Line: line1}}
		entry2     = client.Entry{Labels: labels2, Entry: logproto.Entry{Timestamp: timestamp2, Line: line2}}
	)

	BeforeEach(func() {
		ctlClient = controllerClient{
			mainClient:    &client.FakeLokiClient{},
			defaultClient: &client.FakeLokiClient{},
			logger:        logger,
			name:          "test",
		}
	})

	type handleArgs struct {
		config struct {
			muteDefaultClient bool
			muteMainClient    bool
		}
		input []client.Entry
		want  struct {
			defaultEntries []client.Entry
			mainEntries    []client.Entry
		}
	}

	DescribeTable("#Handle", func(args handleArgs) {
		ctlClient.muteDefaultClient = args.config.muteDefaultClient
		ctlClient.muteMainClient = args.config.muteMainClient
		for _, entry := range args.input {
			err := ctlClient.Handle(entry.Labels, entry.Timestamp, entry.Line)
			Expect(err).ToNot(HaveOccurred())
		}
		Expect(ctlClient.mainClient.(*client.FakeLokiClient).Entries).To(Equal(args.want.mainEntries))
		Expect(ctlClient.defaultClient.(*client.FakeLokiClient).Entries).To(Equal(args.want.defaultEntries))
	},
		Entry("Should send only to the main client", handleArgs{
			config: struct {
				muteDefaultClient bool
				muteMainClient    bool
			}{true, false},
			input: []client.Entry{entry1, entry2},
			want: struct {
				defaultEntries []client.Entry
				mainEntries    []client.Entry
			}{nil, []client.Entry{entry1, entry2}},
		}),
		Entry("Should send only to the default client", handleArgs{
			config: struct {
				muteDefaultClient bool
				muteMainClient    bool
			}{false, true},
			input: []client.Entry{entry1, entry2},
			want: struct {
				defaultEntries []client.Entry
				mainEntries    []client.Entry
			}{[]client.Entry{entry1, entry2}, nil},
		}),
		Entry("Should send to both clients", handleArgs{
			config: struct {
				muteDefaultClient bool
				muteMainClient    bool
			}{false, false},
			input: []client.Entry{entry1, entry2},
			want: struct {
				defaultEntries []client.Entry
				mainEntries    []client.Entry
			}{[]client.Entry{entry1, entry2}, []client.Entry{entry1, entry2}},
		}),
		Entry("Shouldn't send to both clients", handleArgs{
			config: struct {
				muteDefaultClient bool
				muteMainClient    bool
			}{true, true},
			input: []client.Entry{entry1, entry2},
			want: struct {
				defaultEntries []client.Entry
				mainEntries    []client.Entry
			}{nil, nil},
		}),
	)
	type setStateArgs struct {
		inputState        clusterState
		currentState      clusterState
		defaultClientConf *config.ControllerClientConfiguration
		mainClientConf    *config.ControllerClientConfiguration
		want              struct {
			muteMainClient    bool
			muteDefaultClient bool
			state             clusterState
		}
	}
	DescribeTable("#SetState", func(args setStateArgs) {
		ctlClient.defaultClientConf = args.defaultClientConf
		ctlClient.mainClientConf = args.mainClientConf
		ctlClient.state = args.currentState
		ctlClient.SetState(args.inputState)

		Expect(ctlClient.state).To(Equal(args.want.state))
		Expect(ctlClient.muteDefaultClient).To(Equal(args.want.muteDefaultClient))
		Expect(ctlClient.muteMainClient).To(Equal(args.want.muteMainClient))
	},
		Entry("Change state from create to creation", setStateArgs{
			inputState:        clusterStateCreation,
			currentState:      clusterStateCreation,
			defaultClientConf: &config.DefaultControllerClientConfig,
			mainClientConf:    &config.MainControllerClientConfig,
			want: struct {
				muteMainClient    bool
				muteDefaultClient bool
				state             clusterState
			}{false, false, clusterStateCreation},
		}),
		Entry("Change state from create to ready", setStateArgs{
			inputState:        clusterStateReady,
			currentState:      clusterStateCreation,
			defaultClientConf: &config.DefaultControllerClientConfig,
			mainClientConf:    &config.MainControllerClientConfig,
			want: struct {
				muteMainClient    bool
				muteDefaultClient bool
				state             clusterState
			}{false, true, clusterStateReady},
		}),
		Entry("Change state from create to hibernating", setStateArgs{
			inputState:        clusterStateHibernating,
			currentState:      clusterStateCreation,
			defaultClientConf: &config.DefaultControllerClientConfig,
			mainClientConf:    &config.MainControllerClientConfig,
			want: struct {
				muteMainClient    bool
				muteDefaultClient bool
				state             clusterState
			}{true, true, clusterStateHibernating},
		}),
		Entry("Change state from create to hibernated", setStateArgs{
			inputState:        clusterStateHibernated,
			currentState:      clusterStateCreation,
			defaultClientConf: &config.DefaultControllerClientConfig,
			mainClientConf:    &config.MainControllerClientConfig,
			want: struct {
				muteMainClient    bool
				muteDefaultClient bool
				state             clusterState
			}{true, true, clusterStateHibernated},
		}),
		Entry("Change state from create to waking", setStateArgs{
			inputState:        clusterStateWakingUp,
			currentState:      clusterStateCreation,
			defaultClientConf: &config.DefaultControllerClientConfig,
			mainClientConf:    &config.MainControllerClientConfig,
			want: struct {
				muteMainClient    bool
				muteDefaultClient bool
				state             clusterState
			}{false, true, clusterStateWakingUp},
		}),
		Entry("Change state from create to deletion", setStateArgs{
			inputState:        clusterStateDeletion,
			currentState:      clusterStateCreation,
			defaultClientConf: &config.DefaultControllerClientConfig,
			mainClientConf:    &config.MainControllerClientConfig,
			want: struct {
				muteMainClient    bool
				muteDefaultClient bool
				state             clusterState
			}{false, false, clusterStateDeletion},
		}),
	)

	Describe("#Stop", func() {
		It("Should stop immediately", func() {
			ctlClient.Stop()
			Expect(ctlClient.mainClient.(*client.FakeLokiClient).IsStopped).To(BeTrue())
			Expect(ctlClient.defaultClient.(*client.FakeLokiClient).IsStopped).To(BeFalse())
		})

		It("Should stop gracefully", func() {
			ctlClient.StopWait()
			Expect(ctlClient.mainClient.(*client.FakeLokiClient).IsGracefullyStopped).To(BeTrue())
			Expect(ctlClient.defaultClient.(*client.FakeLokiClient).IsGracefullyStopped).To(BeFalse())
		})
	})

	Describe("#GetState", func() {
		It("Should get the state", func() {
			ctlClient.defaultClientConf = &config.DefaultControllerClientConfig
			ctlClient.mainClientConf = &config.MainControllerClientConfig
			ctlClient.SetState(clusterStateReady)
			currentState := ctlClient.GetState()
			Expect(currentState).To(Equal(clusterStateReady))
		})
	})

	Describe("#GetClient", func() {
		var (
			ctl                  *controller
			clientName           = "test-client"
			testControllerClient = &fakeControllerClient{
				FakeLokiClient: client.FakeLokiClient{},
				name:           clientName,
				state:          clusterStateCreation,
			}
		)

		BeforeEach(func() {
			ctl = &controller{
				clients: map[string]ControllerClient{
					clientName: testControllerClient,
				},
				logger: logger,
			}
		})

		It("Should return the right client", func() {
			c, closed := ctl.GetClient(clientName)
			Expect(closed).To(BeFalse())
			Expect(c).To(Equal(testControllerClient))
		})

		It("Should not return the right client", func() {
			c, closed := ctl.GetClient("some-fake-name")
			Expect(closed).To(BeFalse())
			Expect(c).To(BeNil())
		})

		It("Should not return client when controller is stopped", func() {
			ctl.Stop()
			c, closed := ctl.GetClient(clientName)
			Expect(closed).To(BeTrue())
			Expect(c).To(BeNil())
		})
	})
})

type fakeControllerClient struct {
	client.FakeLokiClient
	state clusterState
	name  string
}

func (c *fakeControllerClient) Handle(labels model.LabelSet, time time.Time, entry string) error {
	return c.FakeLokiClient.Handle(labels, time, entry)
}

func (c *fakeControllerClient) SetState(state clusterState) {
	c.state = state
}

func (c *fakeControllerClient) GetState() clusterState {
	return c.state
}
