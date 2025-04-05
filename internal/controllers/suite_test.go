/*
SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and cf-service-operator contributors
SPDX-License-Identifier: Apache-2.0
*/
package controllers

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/sap/cf-service-operator/api/v1alpha1"
	"github.com/sap/cf-service-operator/internal/config"
	"github.com/sap/cf-service-operator/internal/facade"
	"github.com/sap/cf-service-operator/internal/facade/facadefakes"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

var kNoSpace *facade.Space = nil
var kNoInstance *facade.Instance = nil

var kNoError error = nil
var errNotExpected = fmt.Errorf("not expected")
var errCreateInstanceFail = fmt.Errorf("create instance failed")
var errDeleteInstanceFail = fmt.Errorf("delete instance failed")

// constants useful for all tests
const (
	testK8sNamespace  = "test-namespace"
	testK8sSecretName = "test-secret"
	testCfOrgName     = "test-organization"
	testCfSpaceGuid   = "test-space-guid"
	testCfPlanGuid    = "test-plan-guid"
	testCfOwner       = "test-owner"

	// credentials for CF client
	testCfUsername = "test-username"
	testCfPassword = "test-password"
	testCfUrl      = "https://api.cf.example.com"

	testServiceInstanceDefaultReconcileInterval     = 1 * time.Second
	testServiceInstanceDefaultMaxRetries        int = 5
	testServiceInstanceDefaultRetryInterval         = 1 * time.Second
	testServiceInstanceDefaultMaxRetryInterval      = 1 * time.Minute
)

// timeout used for waiting on changes of custom resource
// (overridden by environment variable TEST_TIMEOUT)
var timeout = 5 * time.Minute

// interval used for polling custom resource
var interval = 500 * time.Millisecond

var (
	k8sConfig              *rest.Config
	k8sClient              client.Client
	testCluster            *envtest.Environment
	cancelManager          context.CancelFunc
	fakeOrgClient          *facadefakes.FakeOrganizationClient
	fakeSpaceClient        *facadefakes.FakeSpaceClient
	fakeSpaceHealthChecker *facadefakes.FakeSpaceHealthChecker
)

// -----------------------------------------------------------------------------------------------

func TestController(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Controller Test Suite")
}

// -----------------------------------------------------------------------------------------------

var _ = BeforeSuite(func() {
	// enable logging of operator
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	// for debugging of integration tests, allow setting of timeout by environment variable
	if testTimeout, err := strconv.Atoi(os.Getenv("TEST_TIMEOUT")); err == nil {
		timeout = time.Duration(testTimeout) * time.Second
	}

	prepareKubernetesEnvironment()
	prepareOperatorResources()

	// create operator with some of the controllers required for testing
	// the "real" operator is constructed in main.go
	// disabling metrics server and health probe (not required for tests, but if activate might
	// cause problems with already used ports in the GitHub action runner)

	By("creating K8s manager")
	k8sManager, err := ctrl.NewManager(k8sConfig, ctrl.Options{
		Scheme: scheme.Scheme,
		Metrics: metricsserver.Options{
			BindAddress: "0",
		},
		HealthProbeBindAddress: "0",
	})
	Expect(err).ToNot(HaveOccurred())

	addControllers(k8sManager)

	By("starting K8s manager")
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
// Helper Functions
// -----------------------------------------------------------------------------------------------

// prepare test K8s cluster and K8s client
func prepareKubernetesEnvironment() {
	var err error

	By("spinning up new K8s cluster")
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
}

// -----------------------------------------------------------------------------------------------

// prepare K8s resources for cf-service-operator
func prepareOperatorResources() {
	var err error

	By("creating K8s namespace")
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: testK8sNamespace,
		},
	}
	ctx := context.Background()
	err = k8sClient.Create(ctx, namespace)
	if err != nil && !apierrors.IsAlreadyExists(err) {
		Fail(err.Error())
		Fail("failed to create K8s namespace")
	}

	// Create secret (that will be read during reconcile)
	By("creating K8s secret")
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testK8sSecretName,
			Namespace: testK8sNamespace,
		},
		Data: map[string][]byte{
			"username": []byte(testCfUsername),
			"password": []byte(testCfPassword),
			"url":      []byte(testCfUrl),
		},
	}
	Expect(k8sClient.Create(ctx, secret)).To(Succeed())
}

// -----------------------------------------------------------------------------------------------

func addControllers(k8sManager ctrl.Manager) {
	// add space controller
	spaceReconciler := &SpaceReconciler{
		Kind:                     "Space",
		Client:                   k8sManager.GetClient(),
		Scheme:                   k8sManager.GetScheme(),
		ClusterResourceNamespace: testK8sNamespace,
		ClientBuilder: func(organizationName string, url string, username string, password string, config *config.Config) (facade.OrganizationClient, error) {
			return fakeOrgClient, nil
		},
		HealthCheckerBuilder: func(spaceGuid string, url string, username string, password string, config *config.Config) (facade.SpaceHealthChecker, error) {
			return fakeSpaceHealthChecker, nil
		},
	}
	Expect(spaceReconciler.SetupWithManager(k8sManager)).To(Succeed())

	// add service instance controller
	instanceReconciler := &ServiceInstanceReconciler{
		Client:                   k8sManager.GetClient(),
		Scheme:                   k8sManager.GetScheme(),
		ClusterResourceNamespace: testK8sNamespace,
		ClientBuilder: func(organizationName string, url string, username string, password string, config *config.Config) (facade.SpaceClient, error) {
			return fakeSpaceClient, nil
		},
	}
	Expect(instanceReconciler.SetupWithManager(k8sManager)).To(Succeed())

	// TODO: add service binding controller if required for tests
	// TODO: add another space controller for ClusterSpace resources if required for tests
}

// -----------------------------------------------------------------------------------------------

func createSpaceCR(ctx context.Context, spaceName string) *v1alpha1.Space {
	spaceCR := &v1alpha1.Space{
		ObjectMeta: metav1.ObjectMeta{
			Name:      spaceName,
			Namespace: testK8sNamespace,
		},
		Spec: v1alpha1.SpaceSpec{
			AuthSecretName:   testK8sSecretName,
			OrganizationName: testCfOrgName,
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
		By(fmt.Sprintf("waiting for state '%s' of space CR", expState))
		err := k8sClient.Get(ctx, spaceKey, spaceCR)
		fmt.Println("*** State is ", spaceCR.Status.State)
		if err != nil {
			return err
		}
		if spaceCR.Status.State != expState {
			return fmt.Errorf("expected state '%s' but got '%s'", expState, spaceCR.Status.State)
		}
		return kNoError
	}, timeout, interval).Should(Succeed(), "space CR should have been started")

	return spaceCR
}

// -----------------------------------------------------------------------------------------------

func createInstanceCR(ctx context.Context, instanceName, spaceName string, infinite bool, recreate ...bool) *v1alpha1.ServiceInstance {
	instanceCR := &v1alpha1.ServiceInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      instanceName,
			Namespace: testK8sNamespace,
		},
		Spec: v1alpha1.ServiceInstanceSpec{
			Name:                instanceName,
			ServiceOfferingName: "test-service",
			ServicePlanName:     "test-plan",
			SpaceName:           spaceName,
		},
	}

	if len(recreate) > 0 && recreate[0] {
		annotations := make(map[string]string)
		annotations[v1alpha1.AnnotationRecreate] = "true"
		if !infinite {
			annotations[v1alpha1.AnnotationMaxRetries] = fmt.Sprint(testServiceInstanceDefaultMaxRetries)
		}
		annotations[v1alpha1.AnnotationReconcileTimeout] = testServiceInstanceDefaultReconcileInterval.String()
		instanceCR.SetAnnotations(annotations)
	}

	Expect(k8sClient.Create(ctx, instanceCR)).To(Succeed())

	return instanceCR
}

// -----------------------------------------------------------------------------------------------

func waitForInstanceCR(ctx context.Context, instanceKey client.ObjectKey) *v1alpha1.ServiceInstance {
	instanceCR := &v1alpha1.ServiceInstance{}
	expState := v1alpha1.ServiceInstanceStateReady

	Eventually(func() error {
		By(fmt.Sprintf("waiting for state '%s' of instance CR", expState))
		err := k8sClient.Get(ctx, instanceKey, instanceCR)
		fmt.Println("*** State is ", instanceCR.Status.State)
		if err != nil {
			return err
		}
		if instanceCR.Status.State != expState {
			return fmt.Errorf("expected state %s but got %s", expState, instanceCR.Status.State)
		}
		return kNoError
	}, timeout, interval).Should(Succeed(), "instance CR should have been started")

	return instanceCR
}

func waitForInstanceCRToFail(ctx context.Context, instanceKey client.ObjectKey) *v1alpha1.ServiceInstance {
	instanceCR := &v1alpha1.ServiceInstance{}
	expState := v1alpha1.ServiceInstanceStateError

	Eventually(func() error {
		By(fmt.Sprintf("waiting for state '%s' of instance CR", expState))
		err := k8sClient.Get(ctx, instanceKey, instanceCR)
		fmt.Println("*** State is ", instanceCR.Status.State)
		if err != nil {
			return err
		}
		if instanceCR.Status.State != expState {
			return kNoError
		}
		return fmt.Errorf("expected state %s but got %s", expState, instanceCR.Status.State)
	}, timeout, interval).ShouldNot(Succeed(), "instance CR should be in state Error")

	return instanceCR
}
