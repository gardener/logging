// Copyright 2026 SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"context"
	"errors"

	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	otelcolv1beta1 "github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/gardener/logging/v1/pkg/client/api"
)

// fakeController is the minimal Controller implementation `build` callbacks
// can return; it tracks how many times Stop has been invoked so we can assert
// that successful builds make it through to the channel.
type fakeController struct {
	stopped int
}

func (*fakeController) GetClient(_ string) (api.Output, bool) { return nil, false }
func (*fakeController) Reconcile(_ context.Context, _ ctrl.Request) (ctrl.Result, error) {
	return ctrl.Result{}, nil
}
func (f *fakeController) Stop() { f.stopped++ }

// fakeDynamicScheme builds a runtime.Scheme that knows about
// CustomResourceDefinition. NewSimpleDynamicClient needs the scheme to derive
// the list-kind for the resources it serves.
func fakeDynamicScheme() *runtime.Scheme {
	s := runtime.NewScheme()
	utilruntime.Must(apiextensionsv1.AddToScheme(s))

	return s
}

// establishedCRD returns a CRD whose Established/NamesAccepted conditions are
// True. pendingCRD returns one whose conditions are False. Splitting these
// keeps the call sites self-describing and avoids a control-flag parameter.
func establishedCRD(name, group string) *unstructured.Unstructured {
	return crdUnstructuredWithStatus(name, group, apiextensionsv1.ConditionTrue)
}

func pendingCRD(name, group string) *unstructured.Unstructured {
	return crdUnstructuredWithStatus(name, group, apiextensionsv1.ConditionFalse)
}

func crdUnstructuredWithStatus(name, group string, status apiextensionsv1.ConditionStatus) *unstructured.Unstructured {
	conds := []apiextensionsv1.CustomResourceDefinitionCondition{
		{Type: apiextensionsv1.Established, Status: status},
		{Type: apiextensionsv1.NamesAccepted, Status: status},
	}

	crd := &apiextensionsv1.CustomResourceDefinition{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apiextensions.k8s.io/v1",
			Kind:       "CustomResourceDefinition",
		},
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec:       apiextensionsv1.CustomResourceDefinitionSpec{Group: group},
		Status:     apiextensionsv1.CustomResourceDefinitionStatus{Conditions: conds},
	}
	out, err := runtime.DefaultUnstructuredConverter.ToUnstructured(crd)
	Expect(err).NotTo(HaveOccurred())

	return &unstructured.Unstructured{Object: out}
}

var _ = Describe("awaitController", func() {
	var (
		ctx    context.Context
		cancel context.CancelFunc
		l      logr.Logger
	)

	BeforeEach(func() {
		ctx, cancel = context.WithCancel(context.Background())
		l = logr.Discard()
	})

	AfterEach(func() { cancel() })

	It("delivers the controller for Cluster once that CRD is Established", func() {
		fc := &fakeController{}
		dyn := dynamicfake.NewSimpleDynamicClient(fakeDynamicScheme(),
			establishedCRD("clusters.extensions.gardener.cloud", "extensions.gardener.cloud"))

		out, err := awaitController(ctx, l, scheme, &extensionsv1alpha1.Cluster{}, dyn,
			func(_ context.Context) (Controller, error) { return fc, nil },
		)
		Expect(err).NotTo(HaveOccurred())

		var got Controller
		Eventually(out, "5s", "20ms").Should(Receive(&got))
		Expect(got).To(BeIdenticalTo(Controller(fc)))
	})

	It("delivers the controller for OpenTelemetryCollector once that CRD is Established", func() {
		fc := &fakeController{}
		dyn := dynamicfake.NewSimpleDynamicClient(fakeDynamicScheme(),
			establishedCRD("opentelemetrycollectors.opentelemetry.io", "opentelemetry.io"))

		out, err := awaitController(ctx, l, otelcolScheme, &otelcolv1beta1.OpenTelemetryCollector{}, dyn,
			func(_ context.Context) (Controller, error) { return fc, nil },
		)
		Expect(err).NotTo(HaveOccurred())

		Eventually(out, "5s", "20ms").Should(Receive(BeIdenticalTo(Controller(fc))))
	})

	It("does not deliver while the CRD exists but is not yet Established", func() {
		dyn := dynamicfake.NewSimpleDynamicClient(fakeDynamicScheme(),
			pendingCRD("clusters.extensions.gardener.cloud", "extensions.gardener.cloud"))

		built := false
		out, err := awaitController(ctx, l, scheme, &extensionsv1alpha1.Cluster{}, dyn,
			func(_ context.Context) (Controller, error) {
				built = true

				return &fakeController{}, nil
			},
		)
		Expect(err).NotTo(HaveOccurred())

		Consistently(out, "300ms", "20ms").ShouldNot(Receive())
		Expect(built).To(BeFalse())
	})

	It("ignores CRDs in a different group even if their name matches", func() {
		// Same plural+name shape but a different group — must not match.
		dyn := dynamicfake.NewSimpleDynamicClient(fakeDynamicScheme(),
			establishedCRD("clusters.extensions.gardener.cloud", "wrong.group.example.com"))

		out, err := awaitController(ctx, l, scheme, &extensionsv1alpha1.Cluster{}, dyn,
			func(_ context.Context) (Controller, error) { return &fakeController{}, nil },
		)
		Expect(err).NotTo(HaveOccurred())

		Consistently(out, "300ms", "20ms").ShouldNot(Receive())
	})

	It("closes the channel without a value when ctx is cancelled before the CRD shows up", func() {
		dyn := dynamicfake.NewSimpleDynamicClient(fakeDynamicScheme())

		out, err := awaitController(ctx, l, scheme, &extensionsv1alpha1.Cluster{}, dyn,
			func(_ context.Context) (Controller, error) {
				Fail("build must not run when ctx is cancelled before the CRD is established")

				return nil, nil
			},
		)
		Expect(err).NotTo(HaveOccurred())

		cancel()
		Eventually(out, "2s", "20ms").Should(BeClosed())
	})

	It("closes the channel without a value when `build` returns an error", func() {
		dyn := dynamicfake.NewSimpleDynamicClient(fakeDynamicScheme(),
			establishedCRD("clusters.extensions.gardener.cloud", "extensions.gardener.cloud"))

		out, err := awaitController(ctx, l, scheme, &extensionsv1alpha1.Cluster{}, dyn,
			func(_ context.Context) (Controller, error) {
				return nil, errors.New("boom")
			},
		)
		Expect(err).NotTo(HaveOccurred())

		Eventually(out, "5s", "20ms").Should(BeClosed())
	})

	It("returns a synchronous error when the typed object is not registered in the scheme", func() {
		// extensionsv1alpha1.Cluster against otelcolScheme — not registered there.
		dyn := dynamicfake.NewSimpleDynamicClient(fakeDynamicScheme())

		_, err := awaitController(ctx, l, otelcolScheme, &extensionsv1alpha1.Cluster{}, dyn,
			func(_ context.Context) (Controller, error) { return &fakeController{}, nil },
		)
		Expect(err).To(HaveOccurred())
	})
})
