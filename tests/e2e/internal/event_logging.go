// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/gardener/gardener/pkg/client/kubernetes"
	"github.com/gardener/gardener/pkg/utils"
	"github.com/gardener/gardener/pkg/utils/retry"
	. "github.com/gardener/gardener/pkg/utils/test/matchers"
	"github.com/gardener/gardener/test/framework"
	"github.com/gardener/gardener/test/utils/access"

	fluentbitv1alpha2 "github.com/fluent/fluent-operator/v2/apis/fluentbit/v1alpha2"
	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	errorsutil "github.com/onsi/gomega/gstruct/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	eventMessageForSeed  = "Event from Event Logger integration test related to the Seed"
	eventMessageForShoot = "Event from Event Logger integration test related to the Shoot"
)

// EventLoggingVerifier verifies the event-logger.
type EventLoggingVerifier struct {
	*framework.ShootFramework
	shootNamespace  string
	testId          string
	seedEvent       *corev1.Event
	shootEvent      *corev1.Event
	adminKubeconfig kubernetes.Interface
}

// Verify verifies that the event logging is working properly
func (v *EventLoggingVerifier) Verify(ctx context.Context) {
	v.before(ctx)
	v.prepare(ctx)
	v.expect(ctx)
}

// Cleanup cleans all resources created in prepare function
func (v *EventLoggingVerifier) Cleanup(ctx context.Context) {
	if v.Config.GardenerConfig.ExistingShootName == "" {
		// we only have to clean up if we are using an existing shoot, otherwise the shoot will be deleted
		return
	}

	if v.adminKubeconfig == nil {
		// shoot was never successfully created or accessed, nothing to delete
		return
	}

	By("Update admin kubeconfig to delete the seed and shoot events")
	Eventually(func(g Gomega) {
		shootClient, err := access.CreateShootClientFromAdminKubeconfig(ctx, v.GardenClient, v.Shoot)
		g.Expect(err).NotTo(HaveOccurred())

		g.Expect(shootClient.Client().List(ctx, &corev1.NamespaceList{})).To(Succeed())

		v.adminKubeconfig = shootClient
	}).Should(Succeed())

	By("Delete seed and shoot events")
	Eventually(func(g Gomega) {
		By("Delete seed event")
		g.Expect(v.ShootFramework.SeedClient.Client().Delete(ctx, v.seedEvent)).To(Or(Succeed(), BeNotFoundError()))
		By("Delete shoot event")
		g.Expect(v.adminKubeconfig.Client().Delete(ctx, v.shootEvent)).To(Or(Succeed(), BeNotFoundError()))
	}).Should(Succeed())
}

// before is called before the test is started and checks for required logging components.
func (v *EventLoggingVerifier) before(ctx context.Context) {
	var err error
	v.testId, err = utils.GenerateRandomString(64)
	Expect(err).ToNot(HaveOccurred())

	v.shootNamespace = v.Shoot.Status.TechnicalID
	v.seedEvent = v.getEventFor("seed", v.shootNamespace)
	v.shootEvent = v.getEventFor("shoot", "kube-system")

	Eventually(func(g Gomega) {
		shootClient, err := access.CreateShootClientFromAdminKubeconfig(ctx, v.GardenClient, v.Shoot)
		g.Expect(err).NotTo(HaveOccurred())

		g.Expect(shootClient.Client().List(ctx, &corev1.NamespaceList{})).To(Succeed())

		v.adminKubeconfig = shootClient
	}).Should(Succeed())

	Eventually(func(g Gomega) {
		flbList := &fluentbitv1alpha2.FluentBitList{}
		g.Expect(v.ShootFramework.SeedClient.Client().List(ctx, flbList,
			client.InNamespace("garden"),
			client.MatchingLabels{"app": "fluent-bit"})).To(Succeed())

		for _, flb := range flbList.Items {
			g.Expect(v.ShootFramework.SeedClient.Client().Get(ctx, client.ObjectKey{Namespace: "garden", Name: flb.Name}, &appsv1.DaemonSet{})).To(Succeed())
		}
		g.Expect(v.ShootFramework.SeedClient.Client().Get(ctx, client.ObjectKey{Namespace: v.shootNamespace,
			Name: "vali"}, &appsv1.StatefulSet{})).To(Succeed())
		g.Expect(v.ShootFramework.SeedClient.Client().Get(ctx, client.ObjectKey{Namespace: v.shootNamespace, Name: "event-logger"}, &appsv1.Deployment{})).To(Succeed())
	}).Should(Succeed())

	By("Wait for the shoot Vali and Event-Logger to become ready")
	Expect(v.WaitUntilStatefulSetIsRunning(ctx, "vali", v.shootNamespace, v.ShootFramework.SeedClient)).To(Succeed())
	Expect(v.WaitUntilDeploymentIsReady(ctx, "event-logger", v.shootNamespace, v.ShootFramework.SeedClient)).To(Succeed())
}

// prepare is called after Before and it makes all preparation for the test.
func (v *EventLoggingVerifier) prepare(ctx context.Context) {
	By("Create seed and shoot events")
	Eventually(func(g Gomega) {
		g.Expect(v.ShootFramework.SeedClient.Client().Create(ctx, v.seedEvent)).To(Succeed())
		g.Expect(v.adminKubeconfig.Client().Create(ctx, v.shootEvent)).To(Succeed())
	}).Should(Succeed())
}

// expect is the the process were we expect to get the correct result from the logging test.
func (v *EventLoggingVerifier) expect(ctx context.Context) {
	By("Wait until Vali receive seed event")
	Expect(v.waitUntilValiReceivesEvent(ctx, `origin_extracted="seed",source="event-logger-test",reason="`+v.testId+`"`, []string{eventMessageForSeed}, v.seedEvent.FirstTimestamp.Time)).To(Succeed())

	By("Wait until Vali receive shoot event")
	Expect(v.waitUntilValiReceivesEvent(ctx, `origin_extracted="shoot",source="event-logger-test",reason="`+v.testId+`"`, []string{eventMessageForShoot}, v.shootEvent.FirstTimestamp.Time)).To(Succeed())
}

func (v *EventLoggingVerifier) getEventFor(clusterType, namespace string) *corev1.Event {
	event := &corev1.Event{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
		},
		InvolvedObject: corev1.ObjectReference{
			Kind:      clusterType,
			Namespace: namespace,
			Name:      clusterType + "-event-logger-testing",
		},
		Reason: v.testId,
		Type:   "Normal",
		Source: corev1.EventSource{
			Component: "event-logger-test",
		},
		FirstTimestamp: metav1.Time{
			Time: time.Now(),
		},
	}

	switch clusterType {
	case "seed":
		event.Name = "seed-event"
		event.Message = eventMessageForSeed
	case "shoot":
		event.Name = "shoot-event"
		event.Message = eventMessageForShoot
	default:
		event.Name = "unknown-event"
		event.Message = "Event from Event Logger integration test related to unknown cluster type"
	}

	return event
}

// waitUntilValiReceivesEvent waits until the vali instance in <valiNamespace> receives <wantedEventMessages> for
// filtered events by <queryAfterUnpack>.
// If wantedEventMessages is empty then the function will return nil and will not query the Vali for any events.
func (v *EventLoggingVerifier) waitUntilValiReceivesEvent(ctx context.Context, queryAfterUnpack string, wantedEventMessages []string, startTime time.Time) error {
	var (
		valiLabels = map[string]string{
			"app":  "vali",
			"role": "logging",
		}
		query    = `{job="event-logging"} | unpack`
		interval = 30 * time.Second
		tenant   = "operator"
	)

	if len(wantedEventMessages) < 1 {
		return nil
	}

	if queryAfterUnpack != "" {
		query = query + ` | ` + queryAfterUnpack
	}
	log := v.Logger.WithValues("query", query)

	return retry.Until(ctx, interval, func(ctx context.Context) (done bool, err error) {
		search, err := getValiLogsWithCMD(ctx, log, valiLabels, tenant, v.shootNamespace, query, startTime, v.ShootFramework.SeedClient)
		if err != nil {
			return retry.SevereError(err)
		}

		if len(search.Data.Result) < len(wantedEventMessages) {
			log.Info("Waiting to receive all expected logs", "events", len(search.Data.Result), "wanted", len(wantedEventMessages))
			return retry.MinorError(fmt.Errorf("received only %d/%d events", len(search.Data.Result), len(wantedEventMessages)))
		} else if len(search.Data.Result) > len(wantedEventMessages) {
			return retry.SevereError(fmt.Errorf("expected to receive %d events but was %d", len(wantedEventMessages), len(search.Data.Result)))
		}

		var aggErr errorsutil.AggregateError
		for _, wantedEventMessage := range wantedEventMessages {
			found := false
			for _, result := range search.Data.Result {
				currentMessages := getAllStringsFromRangeSearchResponse(result.Values)
				for _, currentMessage := range currentMessages {
					if currentMessage == wantedEventMessage {
						found = true
						break
					}
				}
				if found {
					break
				}
			}
			if !found {
				aggErr = append(aggErr, fmt.Errorf("event with message \"%s\" is missing", wantedEventMessage))
			}
		}

		if aggErr != nil {
			return retry.SevereError(aggErr)
		}

		return retry.Ok()
	})
}

func getAllStringsFromRangeSearchResponse(values [][]interface{}) []string {
	var result []string
	for _, interfaceEntry := range values {
		for _, subInterfaceEntry := range interfaceEntry {
			entry, ok := subInterfaceEntry.(string)
			if !ok {
				continue
			}
			result = append(result, entry)
		}
	}
	return result
}

// rangeSearchResponse represents the response from a search query_range to vali
type rangeSearchResponse struct {
	Data struct {
		Result []struct {
			Stream map[string]interface{} `json:"stream"`
			Values [][]interface{}        `json:"values"`
		} `json:"result"`
	} `json:"data"`
}

// getValiLogsWithCMD gets logs from startTime until now for query to vali instance in <valiNamespace>
func getValiLogsWithCMD(ctx context.Context, logger logr.Logger, valiLabels map[string]string, tenant, valiNamespace,
	query string, startTime time.Time, client kubernetes.Interface) (*rangeSearchResponse, error) {
	valiLabelsSelector := labels.SelectorFromSet(labels.Set(valiLabels))

	if tenant == "" {
		tenant = "fake"
	}

	command := fmt.Sprintf("wget 'http://localhost:%d/vali/api/v1/query_range?start=%d' -O- '--header=X-Scope-OrgID"+
		": %s' --post-data='query=%s'", 3100, startTime.UnixNano(), tenant, query)

	var reader io.Reader
	err := retry.Until(ctx, 5*time.Second, func(ctx context.Context) (bool, error) {
		var err error
		reader, err = framework.PodExecByLabel(ctx, valiLabelsSelector, "vali", command, valiNamespace, client)
		if err != nil {
			logger.Error(err, "Error exec'ing into pod")
			return retry.MinorError(err)
		}
		return retry.Ok()
	})
	if err != nil {
		return nil, err
	}

	search := &rangeSearchResponse{}

	if err = json.NewDecoder(reader).Decode(search); err != nil {
		return nil, err
	}

	return search, nil
}
