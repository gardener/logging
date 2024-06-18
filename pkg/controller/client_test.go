// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"os"
	"time"

	"github.com/credativ/vali/pkg/logproto"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/prometheus/common/model"
	"github.com/weaveworks/common/logging"

	"github.com/gardener/logging/pkg/client"
	"github.com/gardener/logging/pkg/config"
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
			mainClient:    &client.FakeValiClient{},
			defaultClient: &client.FakeValiClient{},
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
		Expect(ctlClient.mainClient.(*client.FakeValiClient).Entries).To(Equal(args.want.mainEntries))
		Expect(ctlClient.defaultClient.(*client.FakeValiClient).Entries).To(Equal(args.want.defaultEntries))
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
			Expect(ctlClient.mainClient.(*client.FakeValiClient).IsStopped).To(BeTrue())
			Expect(ctlClient.defaultClient.(*client.FakeValiClient).IsStopped).To(BeFalse())
		})

		It("Should stop gracefully", func() {
			ctlClient.StopWait()
			Expect(ctlClient.mainClient.(*client.FakeValiClient).IsGracefullyStopped).To(BeTrue())
			Expect(ctlClient.defaultClient.(*client.FakeValiClient).IsGracefullyStopped).To(BeFalse())
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
				FakeValiClient: client.FakeValiClient{},
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
	client.FakeValiClient
	state clusterState
	name  string
}

func (c *fakeControllerClient) Handle(labels model.LabelSet, time time.Time, entry string) error {
	return c.FakeValiClient.Handle(labels, time, entry)
}

func (c *fakeControllerClient) SetState(state clusterState) {
	c.state = state
}

func (c *fakeControllerClient) GetState() clusterState {
	return c.state
}
