/*
SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and cf-service-operator contributors
SPDX-License-Identifier: Apache-2.0
*/
package controllers

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/sap/cf-service-operator/api/v1alpha1"
	"github.com/sap/cf-service-operator/internal/facade"
	"github.com/sap/cf-service-operator/internal/facade/facadefakes"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
)

const (
	testNamespace    = "cf-env-test"
	testSpace        = "test-space"
	testSecret       = "test-secret"
	testOrganization = "test-organization"
	testGuid         = "test-guid"
)

// timeout used for waiting on changes of custom resource (overridden by environment variable TEST_TIMEOUT)
var timeout = 3 * time.Second

// interval used for polling custom resource
var interval = 500 * time.Millisecond

var (
	k8sConfig              *rest.Config
	k8sClient              client.Client
	testCluster            *envtest.Environment
	k8sManager             ctrl.Manager
	cancelManager          context.CancelFunc
	fakeOrgClient          *facadefakes.FakeOrganizationClient
	fakeSpaceHealthChecker *facadefakes.FakeSpaceHealthChecker
)

func TestSpaceController(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Space Controller Test Suite")
}

// -----------------------------------------------------------------------------------------------
// Suite
// -----------------------------------------------------------------------------------------------

var _ = BeforeSuite(func() {
	By("bootstrapping test environment")

	var err error

	// enable logging of operator
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	By("spin-up new cluster")
	testCluster = &envtest.Environment{
		CRDDirectoryPaths: []string{filepath.Join("..", "..", "crds")},
	}
	k8sConfig, err = testCluster.Start()
	Expect(err).ToNot(HaveOccurred())
	Expect(k8sConfig).ToNot(BeNil())

	By("creating K8s client")
	Expect(v1alpha1.AddToScheme(scheme.Scheme)).To(Succeed())
	k8sClient, err = client.New(k8sConfig, client.Options{Scheme: scheme.Scheme})
	Expect(err).ToNot(HaveOccurred())
	Expect(k8sClient).ToNot(BeNil())

	By("creating K8s manager")
	k8sManager, err = ctrl.NewManager(k8sConfig, ctrl.Options{
		Scheme: scheme.Scheme,
		Metrics: metricsserver.Options{
			BindAddress: "0",
		},
		HealthProbeBindAddress: "0",
	})
	Expect(err).ToNot(HaveOccurred())

	By("adding controllers")
	fakeOrgClient = &facadefakes.FakeOrganizationClient{}
	fakeSpaceHealthChecker = &facadefakes.FakeSpaceHealthChecker{}
	spaceReconciler := &SpaceReconciler{
		Kind:   "Space",
		Client: k8sManager.GetClient(),
		Scheme: k8sManager.GetScheme(),
		ClientBuilder: func(organizationName string, url string, username string, password string) (facade.OrganizationClient, error) {
			return fakeOrgClient, nil
		},
		HealthCheckerBuilder: func(spaceGuid string, url string, username string, password string) (facade.SpaceHealthChecker, error) {
			return fakeSpaceHealthChecker, nil
		},
		// see main.go for more parameters
	}
	Expect(spaceReconciler.SetupWithManager(k8sManager)).To(Succeed())

	By("starting manager")
	ctx, cancel := context.WithCancel(context.Background())
	cancelManager = cancel
	go func() {
		defer GinkgoRecover()
		Expect(k8sManager.Start(ctx)).To(Succeed())
	}()
})

// -----------------------------------------------------------------------------------------------

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	if cancelManager != nil {
		cancelManager()
	}
	if testCluster != nil {
		Expect(testCluster.Stop()).To(Succeed())
	}
})

// -----------------------------------------------------------------------------------------------
// Tests
// -----------------------------------------------------------------------------------------------

var _ = Describe("Space Controller Integration Tests", func() {
	Describe("Reconcile", func() {
		ctx := context.Background()

		BeforeEach(func() {
			Expect(k8sClient).ToNot(BeNil())

			By("creating namespace")
			namespace := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: testNamespace,
				},
			}
			err := k8sClient.Create(ctx, namespace)
			if err != nil && !apierrors.IsAlreadyExists(err) {
				Fail(err.Error())
				Fail("failed to create namespace")
			}

			// Create secret (that will be read during reconcile)
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testSecret,
					Namespace: testNamespace,
				},
				Data: map[string][]byte{
					"username": []byte("test-username"),
					"password": []byte("test-password"),
					"url":      []byte("https://api.cf.sap.hana.ondemand.com"),
				},
			}
			Expect(k8sClient.Create(ctx, secret)).To(Succeed())
		})

		It("should create space", func() {
			By("creating space CR")
			var dummySpace = &facade.Space{Guid: testGuid, Name: testSpace, Owner: "me", Generation: 1}
			fakeOrgClient.CreateSpaceReturns(nil)
			fakeOrgClient.GetSpaceReturns(dummySpace, nil)
			fakeOrgClient.GetSpaceReturnsOnCall(0, nil, nil) // only the 1st call returns no space, that controller will create the space.
			fakeSpaceHealthChecker.CheckReturns(nil)
			var sleep = 0
			// sleep = 500 // with Debugger
			if sleep > 0 {
				createSpaceCR(ctx)
				time.Sleep(time.Duration(sleep) * time.Second)
			} else {
				spaceCR := createSpaceCR(ctx)
				waitForSpaceCR(ctx, client.ObjectKeyFromObject(spaceCR))
			}
			Expect(fakeOrgClient.CreateSpaceCallCount()).To(Equal(1))
		})
	})

})

// -----------------------------------------------------------------------------------------------
// Helper Functions
// -----------------------------------------------------------------------------------------------

func createSpaceCR(ctx context.Context) *v1alpha1.Space {
	spaceCR := &v1alpha1.Space{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testSpace,
			Namespace: testNamespace,
		},
		Spec: v1alpha1.SpaceSpec{
			AuthSecretName:   testSecret,
			OrganizationName: testOrganization,
		},
	}
	Expect(k8sClient.Create(ctx, spaceCR)).To(Succeed())

	return spaceCR
}

// -----------------------------------------------------------------------------------------------

func waitForSpaceCR(ctx context.Context, spaceKey client.ObjectKey) *v1alpha1.Space {
	spaceCR := &v1alpha1.Space{}
	expState := v1alpha1.SpaceStateReady

	Eventually(func() error {
		By("waiting for status 'Ready' of space CR")
		err := k8sClient.Get(ctx, spaceKey, spaceCR)
		fmt.Println("*** Status is ", spaceCR.Status.State)
		if err != nil {
			return err
		}
		if spaceCR.Status.State != expState {
			return fmt.Errorf("expected state %s but got %s", expState, spaceCR.Status.State)
		}
		return nil // success
	}, timeout, interval).Should(Succeed(), "space CR should have been started")

	return spaceCR
}
