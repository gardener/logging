// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"os"
	"time"

	"github.com/credativ/vali/pkg/logproto"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	ginkgov2 "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"github.com/prometheus/common/model"
	"github.com/weaveworks/common/logging"

	"github.com/gardener/logging/pkg/client"
	"github.com/gardener/logging/pkg/config"
)

var _ = ginkgov2.Describe("Controller Client", func() {
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

	ginkgov2.BeforeEach(func() {
		ctlClient = controllerClient{
			shootTarget: target{
				valiClient: &client.FakeValiClient{},
				mute:       false,
				conf:       nil,
			},
			seedTarget: target{
				valiClient: &client.FakeValiClient{},
				mute:       false,
				conf:       nil,
			},
			logger: logger,
			name:   "test",
		}
	})

	// revive:disable:nested-structs
	type handleArgs struct {
		config struct {
			muteSeedClient  bool
			muteShootClient bool
		}
		input []client.Entry
		want  struct {
			seedEntries  []client.Entry
			shootEntries []client.Entry
		}
	}
	// revive:enable:nested-structs

	ginkgov2.DescribeTable("#Handle", func(args handleArgs) {
		ctlClient.seedTarget.mute = args.config.muteSeedClient
		ctlClient.shootTarget.mute = args.config.muteShootClient
		for _, entry := range args.input {
			err := ctlClient.Handle(entry.Labels, entry.Timestamp, entry.Line)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
		}
		gomega.Expect(ctlClient.shootTarget.valiClient.(*client.FakeValiClient).Entries).To(gomega.Equal(args.want.shootEntries))
		gomega.Expect(ctlClient.seedTarget.valiClient.(*client.FakeValiClient).Entries).To(gomega.Equal(args.want.seedEntries))
	},
		ginkgov2.Entry("Should send only to the main client", handleArgs{
			config: struct {
				muteSeedClient  bool
				muteShootClient bool
			}{true, false},
			input: []client.Entry{entry1, entry2},
			want: struct {
				seedEntries  []client.Entry
				shootEntries []client.Entry
			}{nil, []client.Entry{entry1, entry2}},
		}),
		ginkgov2.Entry("Should send only to the default client", handleArgs{
			config: struct {
				muteSeedClient  bool
				muteShootClient bool
			}{false, true},
			input: []client.Entry{entry1, entry2},
			want: struct {
				seedEntries  []client.Entry
				shootEntries []client.Entry
			}{[]client.Entry{entry1, entry2}, nil},
		}),
		ginkgov2.Entry("Should send to both clients", handleArgs{
			config: struct {
				muteSeedClient  bool
				muteShootClient bool
			}{false, false},
			input: []client.Entry{entry1, entry2},
			want: struct {
				seedEntries  []client.Entry
				shootEntries []client.Entry
			}{[]client.Entry{entry1, entry2}, []client.Entry{entry1, entry2}},
		}),
		ginkgov2.Entry("Shouldn't send to both clients", handleArgs{
			config: struct {
				muteSeedClient  bool
				muteShootClient bool
			}{true, true},
			input: []client.Entry{entry1, entry2},
			want: struct {
				seedEntries  []client.Entry
				shootEntries []client.Entry
			}{nil, nil},
		}),
	)

	// revive:disable:nested-structs
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
	// revive:enable:nested-structs

	ginkgov2.DescribeTable("#SetState", func(args setStateArgs) {
		ctlClient.seedTarget.conf = args.defaultClientConf
		ctlClient.shootTarget.conf = args.mainClientConf
		ctlClient.state = args.currentState
		ctlClient.SetState(args.inputState)

		gomega.Expect(ctlClient.state).To(gomega.Equal(args.want.state))
		gomega.Expect(ctlClient.seedTarget.mute).To(gomega.Equal(args.want.muteDefaultClient))
		gomega.Expect(ctlClient.shootTarget.mute).To(gomega.Equal(args.want.muteMainClient))
	},
		ginkgov2.Entry("Change state from create to creation", setStateArgs{
			inputState:        clusterStateCreation,
			currentState:      clusterStateCreation,
			defaultClientConf: &config.SeedControllerClientConfig,
			mainClientConf:    &config.ShootControllerClientConfig,
			want: struct {
				muteMainClient    bool
				muteDefaultClient bool
				state             clusterState
			}{false, false, clusterStateCreation},
		}),
		ginkgov2.Entry("Change state from create to ready", setStateArgs{
			inputState:        clusterStateReady,
			currentState:      clusterStateCreation,
			defaultClientConf: &config.SeedControllerClientConfig,
			mainClientConf:    &config.ShootControllerClientConfig,
			want: struct {
				muteMainClient    bool
				muteDefaultClient bool
				state             clusterState
			}{false, true, clusterStateReady},
		}),
		ginkgov2.Entry("Change state from create to hibernating", setStateArgs{
			inputState:        clusterStateHibernating,
			currentState:      clusterStateCreation,
			defaultClientConf: &config.SeedControllerClientConfig,
			mainClientConf:    &config.ShootControllerClientConfig,
			want: struct {
				muteMainClient    bool
				muteDefaultClient bool
				state             clusterState
			}{true, true, clusterStateHibernating},
		}),
		ginkgov2.Entry("Change state from create to hibernated", setStateArgs{
			inputState:        clusterStateHibernated,
			currentState:      clusterStateCreation,
			defaultClientConf: &config.SeedControllerClientConfig,
			mainClientConf:    &config.ShootControllerClientConfig,
			want: struct {
				muteMainClient    bool
				muteDefaultClient bool
				state             clusterState
			}{true, true, clusterStateHibernated},
		}),
		ginkgov2.Entry("Change state from create to waking", setStateArgs{
			inputState:        clusterStateWakingUp,
			currentState:      clusterStateCreation,
			defaultClientConf: &config.SeedControllerClientConfig,
			mainClientConf:    &config.ShootControllerClientConfig,
			want: struct {
				muteMainClient    bool
				muteDefaultClient bool
				state             clusterState
			}{false, true, clusterStateWakingUp},
		}),
		ginkgov2.Entry("Change state from create to deletion", setStateArgs{
			inputState:        clusterStateDeletion,
			currentState:      clusterStateCreation,
			defaultClientConf: &config.SeedControllerClientConfig,
			mainClientConf:    &config.ShootControllerClientConfig,
			want: struct {
				muteMainClient    bool
				muteDefaultClient bool
				state             clusterState
			}{false, false, clusterStateDeletion},
		}),
	)

	ginkgov2.Describe("#Stop", func() {
		ginkgov2.It("Should stop immediately", func() {
			ctlClient.Stop()
			gomega.Expect(ctlClient.shootTarget.valiClient.(*client.FakeValiClient).IsStopped).To(gomega.BeTrue())
			gomega.Expect(ctlClient.seedTarget.valiClient.(*client.FakeValiClient).IsStopped).To(gomega.BeFalse())
		})

		ginkgov2.It("Should stop gracefully", func() {
			ctlClient.StopWait()
			gomega.Expect(ctlClient.shootTarget.valiClient.(*client.FakeValiClient).IsGracefullyStopped).To(gomega.BeTrue())
			gomega.Expect(ctlClient.seedTarget.valiClient.(*client.FakeValiClient).IsGracefullyStopped).To(gomega.BeFalse())
		})
	})

	ginkgov2.Describe("#GetState", func() {
		ginkgov2.It("Should get the state", func() {
			ctlClient.seedTarget.conf = &config.SeedControllerClientConfig
			ctlClient.shootTarget.conf = &config.ShootControllerClientConfig
			ctlClient.SetState(clusterStateReady)
			currentState := ctlClient.GetState()
			gomega.Expect(currentState).To(gomega.Equal(clusterStateReady))
		})
	})

	ginkgov2.Describe("#GetClient", func() {
		var (
			ctl                  *controller
			clientName           = "test-client"
			testControllerClient = &fakeControllerClient{
				FakeValiClient: client.FakeValiClient{},
				name:           clientName,
				state:          clusterStateCreation,
			}
		)

		ginkgov2.BeforeEach(func() {
			ctl = &controller{
				clients: map[string]Client{
					clientName: testControllerClient,
				},
				logger: logger,
			}
		})

		ginkgov2.It("Should return the right client", func() {
			c, closed := ctl.GetClient(clientName)
			gomega.Expect(closed).To(gomega.BeFalse())
			gomega.Expect(c).To(gomega.Equal(testControllerClient))
		})

		ginkgov2.It("Should not return the right client", func() {
			c, closed := ctl.GetClient("some-fake-name")
			gomega.Expect(closed).To(gomega.BeFalse())
			gomega.Expect(c).To(gomega.BeNil())
		})

		ginkgov2.It("Should not return client when controller is stopped", func() {
			ctl.Stop()
			c, closed := ctl.GetClient(clientName)
			gomega.Expect(closed).To(gomega.BeTrue())
			gomega.Expect(c).To(gomega.BeNil())
		})
	})
})

type fakeControllerClient struct {
	client.FakeValiClient
	state clusterState
	name  string
}

func (c *fakeControllerClient) Handle(labels any, t time.Time, entry string) error {
	return c.FakeValiClient.Handle(labels, t, entry)
}

func (c *fakeControllerClient) SetState(state clusterState) {
	c.state = state
}

func (c *fakeControllerClient) GetState() clusterState {
	return c.state
}
