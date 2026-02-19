// Copyright 2025 SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"context"
	"fmt"
	"sync"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/prometheus/client_golang/prometheus/testutil"

	"github.com/gardener/logging/v1/pkg/client"
	"github.com/gardener/logging/v1/pkg/config"
	"github.com/gardener/logging/v1/pkg/log"
	"github.com/gardener/logging/v1/pkg/metrics"
	"github.com/gardener/logging/v1/pkg/types"
)

var _ = Describe("Controller Client", func() {
	var (
		ctlClient controllerClient
		logger    = log.NewLogger("info")
		line1     = "testline1"
		line2     = "testline2"
		entry1    = types.OutputEntry{
			Timestamp: time.Now(),
			Record:    map[string]any{"msg": line1},
		}
		entry2 = types.OutputEntry{
			Timestamp: time.Now().Add(time.Second),
			Record:    map[string]any{"msg": line2},
		}
	)

	BeforeEach(func() {
		// Create separate NoopClient instances with different endpoints for separate metrics
		shootClient, err := client.NewNoopClient(
			context.Background(),
			config.Config{
				OTLPConfig: config.OTLPConfig{
					Endpoint: "shoot-endpoint:4317",
				},
			}, logger)
		Expect(err).ToNot(HaveOccurred())

		seedClient, err := client.NewNoopClient(
			context.Background(),
			config.Config{
				OTLPConfig: config.OTLPConfig{
					Endpoint: "seed-endpoint:4317",
				},
			}, logger)
		Expect(err).ToNot(HaveOccurred())

		ctlClient = controllerClient{
			shootTarget: target{
				client: shootClient,
				mute:   false,
				conf:   nil,
			},
			seedTarget: target{
				client: seedClient,
				mute:   false,
				conf:   nil,
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
		input []types.OutputEntry
		want  struct {
			seedLogCount  int
			shootLogCount int
		}
	}
	// revive:enable:nested-structs

	DescribeTable("#Handle", func(args handleArgs) {
		// Get initial metrics (noop client drops all logs)
		initialShootDropped := testutil.ToFloat64(metrics.DroppedLogs.WithLabelValues("shoot-endpoint:4317", "noop"))
		initialSeedDropped := testutil.ToFloat64(metrics.DroppedLogs.WithLabelValues("seed-endpoint:4317", "noop"))

		ctlClient.seedTarget.mute = args.config.muteSeedClient
		ctlClient.shootTarget.mute = args.config.muteShootClient
		for _, entry := range args.input {
			err := ctlClient.Handle(entry)
			Expect(err).ToNot(HaveOccurred())
		}

		// Get final metrics
		finalShootDropped := testutil.ToFloat64(metrics.DroppedLogs.WithLabelValues("shoot-endpoint:4317", "noop"))
		finalSeedDropped := testutil.ToFloat64(metrics.DroppedLogs.WithLabelValues("seed-endpoint:4317", "noop"))

		// Calculate actual counts
		shootCount := int(finalShootDropped - initialShootDropped)
		seedCount := int(finalSeedDropped - initialSeedDropped)

		Expect(shootCount).To(Equal(args.want.shootLogCount), "Shoot client should have received %d logs", args.want.shootLogCount)
		Expect(seedCount).To(Equal(args.want.seedLogCount), "Seed client should have received %d logs", args.want.seedLogCount)
	},
		Entry("Should send only to the shoot client", handleArgs{
			config: struct {
				muteSeedClient  bool
				muteShootClient bool
			}{true, false},
			input: []types.OutputEntry{entry1, entry2},
			want: struct {
				seedLogCount  int
				shootLogCount int
			}{0, 2},
		}),
		Entry("Should send only to the seed client", handleArgs{
			config: struct {
				muteSeedClient  bool
				muteShootClient bool
			}{false, true},
			input: []types.OutputEntry{entry1, entry2},
			want: struct {
				seedLogCount  int
				shootLogCount int
			}{2, 0},
		}),
		Entry("Should send to both clients", handleArgs{
			config: struct {
				muteSeedClient  bool
				muteShootClient bool
			}{false, false},
			input: []types.OutputEntry{entry1, entry2},
			want: struct {
				seedLogCount  int
				shootLogCount int
			}{2, 2},
		}),
		Entry("Shouldn't send to both clients", handleArgs{
			config: struct {
				muteSeedClient  bool
				muteShootClient bool
			}{true, true},
			input: []types.OutputEntry{entry1, entry2},
			want: struct {
				seedLogCount  int
				shootLogCount int
			}{0, 0},
		}),
	)

	// revive:disable:nested-structs
	type setStateArgs struct {
		inputState        clusterState
		currentState      clusterState
		seedClientConfig  *config.ControllerClientConfiguration
		shootClientConfig *config.ControllerClientConfiguration
		want              struct {
			muteShootClient bool
			muteSeedClient  bool
			state           clusterState
		}
	}
	// revive:enable:nested-structs

	DescribeTable("#SetState", func(args setStateArgs) {
		ctlClient.seedTarget.conf = args.seedClientConfig
		ctlClient.shootTarget.conf = args.shootClientConfig
		ctlClient.state = args.currentState
		ctlClient.SetState(args.inputState)

		Expect(ctlClient.state).To(Equal(args.want.state))
		Expect(ctlClient.seedTarget.mute).To(Equal(args.want.muteSeedClient))
		Expect(ctlClient.shootTarget.mute).To(Equal(args.want.muteShootClient))
	},
		Entry("Change state from create to creation", setStateArgs{
			inputState:        clusterStateCreation,
			currentState:      clusterStateCreation,
			seedClientConfig:  &config.SeedControllerClientConfig,
			shootClientConfig: &config.ShootControllerClientConfig,
			want: struct {
				muteShootClient bool
				muteSeedClient  bool
				state           clusterState
			}{false, false, clusterStateCreation},
		}),
		Entry("Change state from create to ready", setStateArgs{
			inputState:        clusterStateReady,
			currentState:      clusterStateCreation,
			seedClientConfig:  &config.SeedControllerClientConfig,
			shootClientConfig: &config.ShootControllerClientConfig,
			want: struct {
				muteShootClient bool
				muteSeedClient  bool
				state           clusterState
			}{false, true, clusterStateReady},
		}),
		Entry("Change state from create to hibernating", setStateArgs{
			inputState:        clusterStateHibernating,
			currentState:      clusterStateCreation,
			seedClientConfig:  &config.SeedControllerClientConfig,
			shootClientConfig: &config.ShootControllerClientConfig,
			want: struct {
				muteShootClient bool
				muteSeedClient  bool
				state           clusterState
			}{true, true, clusterStateHibernating},
		}),
		Entry("Change state from create to hibernated", setStateArgs{
			inputState:        clusterStateHibernated,
			currentState:      clusterStateCreation,
			seedClientConfig:  &config.SeedControllerClientConfig,
			shootClientConfig: &config.ShootControllerClientConfig,
			want: struct {
				muteShootClient bool
				muteSeedClient  bool
				state           clusterState
			}{true, true, clusterStateHibernated},
		}),
		Entry("Change state from create to waking", setStateArgs{
			inputState:        clusterStateWakingUp,
			currentState:      clusterStateCreation,
			seedClientConfig:  &config.SeedControllerClientConfig,
			shootClientConfig: &config.ShootControllerClientConfig,
			want: struct {
				muteShootClient bool
				muteSeedClient  bool
				state           clusterState
			}{false, true, clusterStateWakingUp},
		}),
		Entry("Change state from create to deletion", setStateArgs{
			inputState:        clusterStateDeletion,
			currentState:      clusterStateCreation,
			seedClientConfig:  &config.SeedControllerClientConfig,
			shootClientConfig: &config.ShootControllerClientConfig,
			want: struct {
				muteShootClient bool
				muteSeedClient  bool
				state           clusterState
			}{false, false, clusterStateDeletion},
		}),
	)

	Describe("#Stop", func() {
		It("Should stop immediately without errors", func() {
			// Stop should not panic or error
			ctlClient.Stop()

			// Since NoopClient doesn't enforce stopping behavior (it's a no-op),
			// we just verify that Stop() can be called without issues
			// The actual stopping behavior would be tested with real clients
		})

		It("Should stop gracefully and wait for processing", func() {
			// Send some logs first
			initialShootDropped := testutil.ToFloat64(metrics.DroppedLogs.WithLabelValues("shoot-endpoint:4317", "noop"))
			initialSeedDropped := testutil.ToFloat64(metrics.DroppedLogs.WithLabelValues("seed-endpoint:4317", "noop"))

			entry := types.OutputEntry{
				Timestamp: time.Now(),
				Record:    map[string]any{"msg": "test before graceful stop"},
			}
			err := ctlClient.Handle(entry)
			Expect(err).ToNot(HaveOccurred())

			// StopWait should not panic or error
			ctlClient.StopWait()

			// Verify the log was processed
			finalShootDropped := testutil.ToFloat64(metrics.DroppedLogs.WithLabelValues("shoot-endpoint:4317", "noop"))
			finalSeedDropped := testutil.ToFloat64(metrics.DroppedLogs.WithLabelValues("seed-endpoint:4317", "noop"))
			shootCount := int(finalShootDropped - initialShootDropped)
			seedCount := int(finalSeedDropped - initialSeedDropped)
			Expect(shootCount).To(Equal(1), "Shoot client should have processed log before stopping")
			Expect(seedCount).To(Equal(1), "Seed client should have processed log")
		})
	})

	Describe("#ConcurrentAccess", func() {
		It("Should handle concurrent log writes safely", func() {
			// Get initial metrics
			initialShootDropped := testutil.ToFloat64(metrics.DroppedLogs.WithLabelValues("shoot-endpoint:4317", "noop"))
			initialSeedDropped := testutil.ToFloat64(metrics.DroppedLogs.WithLabelValues("seed-endpoint:4317", "noop"))

			// Send logs concurrently from multiple goroutines
			numGoroutines := 10
			logsPerGoroutine := 10
			var wg sync.WaitGroup
			wg.Add(numGoroutines)

			for i := range numGoroutines {
				go func(id int) {
					defer wg.Done()
					for j := range logsPerGoroutine {
						entry := types.OutputEntry{
							Timestamp: time.Now(),
							Record:    map[string]any{"msg": fmt.Sprintf("concurrent log from goroutine %d, message %d", id, j)},
						}
						err := ctlClient.Handle(entry)
						Expect(err).ToNot(HaveOccurred())
					}
				}(i)
			}

			wg.Wait()

			// Get final metrics
			finalShootDropped := testutil.ToFloat64(metrics.DroppedLogs.WithLabelValues("shoot-endpoint:4317", "noop"))
			finalSeedDropped := testutil.ToFloat64(metrics.DroppedLogs.WithLabelValues("seed-endpoint:4317", "noop"))

			// Verify all logs were processed
			shootCount := int(finalShootDropped - initialShootDropped)
			seedCount := int(finalSeedDropped - initialSeedDropped)
			expectedCount := numGoroutines * logsPerGoroutine
			Expect(shootCount).To(Equal(expectedCount), "Shoot client should have processed all concurrent logs")
			Expect(seedCount).To(Equal(expectedCount), "Seed client should have processed all concurrent logs")
		})

		It("Should handle concurrent state changes safely", func() {
			// Get initial metrics
			initialShootDropped := testutil.ToFloat64(metrics.DroppedLogs.WithLabelValues("shoot-endpoint:4317", "noop"))
			initialSeedDropped := testutil.ToFloat64(metrics.DroppedLogs.WithLabelValues("seed-endpoint:4317", "noop"))

			// Set up config for state changes
			ctlClient.seedTarget.conf = &config.SeedControllerClientConfig
			ctlClient.shootTarget.conf = &config.ShootControllerClientConfig

			var wg sync.WaitGroup
			numGoroutines := 5
			wg.Add(numGoroutines * 2) // Half for state changes, half for log writes

			// Goroutines changing states
			states := []clusterState{clusterStateCreation, clusterStateReady, clusterStateHibernating, clusterStateWakingUp, clusterStateDeletion}
			for i := range numGoroutines {
				go func(_ int) {
					defer wg.Done()
					for j := range 10 {
						ctlClient.SetState(states[j%len(states)])
					}
				}(i)
			}

			// Goroutines sending logs
			for i := range numGoroutines {
				go func(id int) {
					defer wg.Done()
					for j := range 10 {
						entry := types.OutputEntry{
							Timestamp: time.Now(),
							Record:    map[string]any{"msg": fmt.Sprintf("concurrent log during state changes %d-%d", id, j)},
						}
						_ = ctlClient.Handle(entry)
					}
				}(i)
			}

			wg.Wait()

			// Get final metrics - we don't know exact count due to muting during state changes
			// but we verify no panics occurred and some logs were processed
			finalShootDropped := testutil.ToFloat64(metrics.DroppedLogs.WithLabelValues("shoot-endpoint:4317", "noop"))
			finalSeedDropped := testutil.ToFloat64(metrics.DroppedLogs.WithLabelValues("seed-endpoint:4317", "noop"))

			shootCount := int(finalShootDropped - initialShootDropped)
			seedCount := int(finalSeedDropped - initialSeedDropped)

			// At least some logs should have been processed
			Expect(shootCount+seedCount).To(BeNumerically(">", 0), "At least some logs should have been processed during concurrent operations")
		})

		It("Should handle concurrent writes with stop", func() {
			// Get initial metrics
			initialSeedDropped := testutil.ToFloat64(metrics.DroppedLogs.WithLabelValues("seed-endpoint:4317", "noop"))

			var wg sync.WaitGroup
			numGoroutines := 5
			wg.Add(numGoroutines + 1) // Writers + stopper

			// Goroutines sending logs
			for i := range numGoroutines {
				go func(id int) {
					defer wg.Done()
					for j := range 20 {
						entry := types.OutputEntry{
							Timestamp: time.Now(),
							Record:    map[string]any{"msg": fmt.Sprintf("concurrent log before stop %d-%d", id, j)},
						}
						_ = ctlClient.Handle(entry)
						time.Sleep(1 * time.Millisecond)
					}
				}(i)
			}

			// Goroutine that stops the client after a delay
			go func() {
				defer wg.Done()
				time.Sleep(10 * time.Millisecond)
				ctlClient.Stop()
			}()

			wg.Wait()

			// Verify seed client still processed logs (only shoot was stopped)
			finalSeedDropped := testutil.ToFloat64(metrics.DroppedLogs.WithLabelValues("seed-endpoint:4317", "noop"))
			seedCount := int(finalSeedDropped - initialSeedDropped)
			Expect(seedCount).To(BeNumerically(">", 0), "Seed client should have processed logs")
		})
	})

	Describe("#GetState", func() {
		It("Should get the state", func() {
			ctlClient.seedTarget.conf = &config.SeedControllerClientConfig
			ctlClient.shootTarget.conf = &config.ShootControllerClientConfig
			ctlClient.SetState(clusterStateReady)
			currentState := ctlClient.GetState()
			Expect(currentState).To(Equal(clusterStateReady))
		})
	})

	Describe("#GetClient", func() {
		var (
			reconciler           *ClusterReconciler
			clientName           = "test-client"
			testControllerClient *fakeControllerClient
		)

		BeforeEach(func() {
			noopClient, err := client.NewNoopClient(
				context.Background(),
				config.Config{
					OTLPConfig: config.OTLPConfig{
						Endpoint: "fake-client-endpoint:4317",
					},
				}, logger)
			Expect(err).ToNot(HaveOccurred())

			testControllerClient = &fakeControllerClient{
				OutputClient: noopClient,
				name:         clientName,
				state:        clusterStateCreation,
			}

			ctx, cancel := context.WithCancel(context.Background())
			reconciler = &ClusterReconciler{
				clients: map[string]Client{
					clientName: testControllerClient,
				},
				logger: logger,
				ctx:    ctx,
				cancel: cancel,
			}
		})

		It("Should return the right client", func() {
			c, closed := reconciler.GetClient(clientName)
			Expect(closed).To(BeFalse())
			Expect(c).To(Equal(testControllerClient))
		})

		It("Should not return the right client", func() {
			c, closed := reconciler.GetClient("some-fake-name")
			Expect(closed).To(BeFalse())
			Expect(c).To(BeNil())
		})

		It("Should not return client when controller is stopped", func() {
			reconciler.Stop()
			c, closed := reconciler.GetClient(clientName)
			Expect(closed).To(BeTrue())
			Expect(c).To(BeNil())
		})
	})
})

type fakeControllerClient struct {
	client.OutputClient
	state clusterState
	name  string
}

func (c *fakeControllerClient) SetState(state clusterState) {
	c.state = state
}

func (c *fakeControllerClient) GetState() clusterState {
	return c.state
}
