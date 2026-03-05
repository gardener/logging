// Copyright 2025 SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"context"
	"regexp"
	"time"

	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	otelcolv1beta1 "github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	pkgclient "github.com/gardener/logging/v1/pkg/client"
	"github.com/gardener/logging/v1/pkg/config"
)

var _ = Describe("OpenTelemetryCollectorReconciler", func() {
	var (
		reconciler        *OpenTelemetryCollectorReconciler
		ctx               context.Context
		cancel            context.CancelFunc
		otelcolScheme     *runtime.Scheme
		dynamicHostPrefix = "http://logging."
		dynamicHostSuffix = ".svc:4318/v1/logs"
		namespace         = "shoot--dev--logging"
		otelcolName       = "test-collector"
		labelKey          = "app.kubernetes.io/managed-by"
		labelValue        = "gardener"
		nsLabelKey        = "gardener.cloud/role"
		nsLabelValue      = "shoot"
	)

	BeforeEach(func() {
		otelcolScheme = runtime.NewScheme()
		Expect(otelcolv1beta1.AddToScheme(otelcolScheme)).To(Succeed())
		Expect(corev1.AddToScheme(otelcolScheme)).To(Succeed())

		ctx, cancel = context.WithCancel(context.Background())

		labelSelector, err := labels.Parse(labelKey + "=" + labelValue)
		Expect(err).ToNot(HaveOccurred())
		namespaceLabelSelector, err := labels.Parse(nsLabelKey + "=" + nsLabelValue)
		Expect(err).ToNot(HaveOccurred())

		reconciler = &OpenTelemetryCollectorReconciler{
			conf: &config.Config{
				OTLPConfig: config.OTLPConfig{
					DQueConfig: config.DefaultDQueConfig,
				},
				ControllerConfig: config.ControllerConfig{
					DynamicHostPrefix: dynamicHostPrefix,
					DynamicHostSuffix: dynamicHostSuffix,
					DynamicHostRegex:  "shoot--.*",
					OpenTelemetryCollectorNamespaceLabelSelector: nsLabelKey + "=" + nsLabelValue,
					OpenTelemetryCollectorLabelSelector:          labelKey + "=" + labelValue,
				},
			},
			clients:                make(map[string]pkgclient.OutputClient),
			logger:                 logr.Discard(),
			ctx:                    ctx,
			cancel:                 cancel,
			labelSelector:          labelSelector,
			namespaceLabelSelector: namespaceLabelSelector,
			dynamicHostRegex:       regexp.MustCompile("shoot--.*"),
		}
	})

	AfterEach(func() {
		cancel()
	})

	Describe("#GetClient", func() {
		It("should return existing client", func() {
			reconciler.clients[namespace] = &fakeOutputClient{}
			c, closed := reconciler.GetClient(namespace)
			Expect(closed).To(BeFalse())
			Expect(c).ToNot(BeNil())
		})

		It("should return nil for non-existing client", func() {
			c, closed := reconciler.GetClient("non-existing")
			Expect(closed).To(BeFalse())
			Expect(c).To(BeNil())
		})

		It("should return nil and closed=true when stopped", func() {
			reconciler.clients = nil
			c, closed := reconciler.GetClient(namespace)
			Expect(closed).To(BeTrue())
			Expect(c).To(BeNil())
		})
	})

	Describe("#Stop", func() {
		It("should stop all clients and set clients to nil", func() {
			client1 := &fakeOutputClient{}
			client2 := &fakeOutputClient{}
			reconciler.clients[namespace] = client1
			reconciler.clients["shoot--dev--other"] = client2

			reconciler.Stop()

			Expect(reconciler.clients).To(BeNil())
			Expect(client1.isStopped).To(BeTrue())
			Expect(client2.isStopped).To(BeTrue())
		})
	})

	Describe("#Reconcile", func() {
		var (
			otelcol *otelcolv1beta1.OpenTelemetryCollector
			ns      *corev1.Namespace
		)

		BeforeEach(func() {
			otelcol = &otelcolv1beta1.OpenTelemetryCollector{
				ObjectMeta: metav1.ObjectMeta{
					Name:      otelcolName,
					Namespace: namespace,
					Labels: map[string]string{
						labelKey: labelValue,
					},
				},
			}

			ns = &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: namespace,
					Labels: map[string]string{
						nsLabelKey: nsLabelValue,
					},
				},
			}
		})

		It("should create a client for an allowed OpenTelemetryCollector", func() {
			reconciler.Client = fake.NewClientBuilder().
				WithScheme(otelcolScheme).
				WithObjects(otelcol, ns).
				Build()

			result, err := reconciler.Reconcile(ctx, ctrl.Request{
				NamespacedName: types.NamespacedName{
					Name:      otelcolName,
					Namespace: namespace,
				},
			})

			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal(ctrl.Result{}))
			Expect(reconciler.clients).To(HaveKey(namespace))
		})

		It("should delete client when OpenTelemetryCollector is not found", func() {
			reconciler.clients[namespace] = &fakeOutputClient{}
			reconciler.Client = fake.NewClientBuilder().
				WithScheme(otelcolScheme).
				Build()

			result, err := reconciler.Reconcile(ctx, ctrl.Request{
				NamespacedName: types.NamespacedName{
					Name:      otelcolName,
					Namespace: namespace,
				},
			})

			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal(ctrl.Result{}))
			Expect(reconciler.clients).ToNot(HaveKey(namespace))
		})

		It("should delete client when OpenTelemetryCollector is being deleted", func() {
			now := metav1.NewTime(time.Now())
			otelcol.DeletionTimestamp = &now
			otelcol.Finalizers = []string{"test-finalizer"}

			reconciler.clients[namespace] = &fakeOutputClient{}
			reconciler.Client = fake.NewClientBuilder().
				WithScheme(otelcolScheme).
				WithObjects(otelcol, ns).
				Build()

			result, err := reconciler.Reconcile(ctx, ctrl.Request{
				NamespacedName: types.NamespacedName{
					Name:      otelcolName,
					Namespace: namespace,
				},
			})

			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal(ctrl.Result{}))
			Expect(reconciler.clients).ToNot(HaveKey(namespace))
		})

		It("should delete client when labels don't match", func() {
			otelcol.Labels = map[string]string{"other-key": "other-value"}

			reconciler.clients[namespace] = &fakeOutputClient{}
			reconciler.Client = fake.NewClientBuilder().
				WithScheme(otelcolScheme).
				WithObjects(otelcol, ns).
				Build()

			result, err := reconciler.Reconcile(ctx, ctrl.Request{
				NamespacedName: types.NamespacedName{
					Name:      otelcolName,
					Namespace: namespace,
				},
			})

			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal(ctrl.Result{}))
			Expect(reconciler.clients).ToNot(HaveKey(namespace))
		})

		It("should delete client when namespace labels don't match", func() {
			ns.Labels = map[string]string{"other-key": "other-value"}

			reconciler.clients[namespace] = &fakeOutputClient{}
			reconciler.Client = fake.NewClientBuilder().
				WithScheme(otelcolScheme).
				WithObjects(otelcol, ns).
				Build()

			result, err := reconciler.Reconcile(ctx, ctrl.Request{
				NamespacedName: types.NamespacedName{
					Name:      otelcolName,
					Namespace: namespace,
				},
			})

			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal(ctrl.Result{}))
			Expect(reconciler.clients).ToNot(HaveKey(namespace))
		})

		It("should delete client when namespace name doesn't match DynamicHostRegex", func() {
			nonMatchingNS := "kube-system"
			otelcol.Namespace = nonMatchingNS
			ns.Name = nonMatchingNS

			reconciler.clients[nonMatchingNS] = &fakeOutputClient{}
			reconciler.Client = fake.NewClientBuilder().
				WithScheme(otelcolScheme).
				WithObjects(otelcol, ns).
				Build()

			result, err := reconciler.Reconcile(ctx, ctrl.Request{
				NamespacedName: types.NamespacedName{
					Name:      otelcolName,
					Namespace: nonMatchingNS,
				},
			})

			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal(ctrl.Result{}))
			Expect(reconciler.clients).ToNot(HaveKey(nonMatchingNS))
		})

		It("should not overwrite existing client", func() {
			existingClient := &fakeOutputClient{}
			reconciler.clients[namespace] = existingClient
			reconciler.Client = fake.NewClientBuilder().
				WithScheme(otelcolScheme).
				WithObjects(otelcol, ns).
				Build()

			result, err := reconciler.Reconcile(ctx, ctrl.Request{
				NamespacedName: types.NamespacedName{
					Name:      otelcolName,
					Namespace: namespace,
				},
			})

			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal(ctrl.Result{}))
			Expect(reconciler.clients[namespace]).To(BeIdenticalTo(existingClient))
		})
	})

	Describe("#buildClientConfig", func() {
		It("should build config with correct endpoint", func() {
			conf := reconciler.buildClientConfig(namespace)
			Expect(conf).ToNot(BeNil())
			Expect(conf.OTLPConfig.Endpoint).To(Equal(dynamicHostPrefix + namespace + dynamicHostSuffix))
			Expect(conf.OTLPConfig.DQueConfig.DQueName).To(Equal(namespace))
		})
	})

	Describe("#deleteClient", func() {
		It("should delete an existing client", func() {
			reconciler.clients[namespace] = &fakeOutputClient{}
			reconciler.deleteClient(namespace)
			Expect(reconciler.clients).ToNot(HaveKey(namespace))
		})

		It("should be a no-op for non-existing client", func() {
			reconciler.deleteClient("non-existing")
			Expect(reconciler.clients).ToNot(HaveKey("non-existing"))
		})

		It("should be a no-op when stopped", func() {
			reconciler.clients = nil
			reconciler.deleteClient(namespace)
			Expect(reconciler.clients).To(BeNil())
		})
	})

	Describe("#isNamespaceAllowed", func() {
		It("should allow namespace matching regex and label selector", func() {
			ns := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: namespace,
					Labels: map[string]string{
						nsLabelKey: nsLabelValue,
					},
				},
			}
			reconciler.Client = fake.NewClientBuilder().
				WithScheme(otelcolScheme).
				WithObjects(ns).
				Build()

			allowed, err := reconciler.isNamespaceAllowed(ctx, namespace)
			Expect(err).ToNot(HaveOccurred())
			Expect(allowed).To(BeTrue())
		})

		It("should reject namespace not matching regex", func() {
			reconciler.Client = fake.NewClientBuilder().
				WithScheme(otelcolScheme).
				Build()

			allowed, err := reconciler.isNamespaceAllowed(ctx, "kube-system")
			Expect(err).ToNot(HaveOccurred())
			Expect(allowed).To(BeFalse())
		})

		It("should reject namespace not matching label selector", func() {
			ns := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name:   namespace,
					Labels: map[string]string{"other": "label"},
				},
			}
			reconciler.Client = fake.NewClientBuilder().
				WithScheme(otelcolScheme).
				WithObjects(ns).
				Build()

			allowed, err := reconciler.isNamespaceAllowed(ctx, namespace)
			Expect(err).ToNot(HaveOccurred())
			Expect(allowed).To(BeFalse())
		})

		It("should reject namespace that does not exist", func() {
			reconciler.Client = fake.NewClientBuilder().
				WithScheme(otelcolScheme).
				Build()

			allowed, err := reconciler.isNamespaceAllowed(ctx, namespace)
			Expect(err).ToNot(HaveOccurred())
			Expect(allowed).To(BeFalse())
		})
	})
})
