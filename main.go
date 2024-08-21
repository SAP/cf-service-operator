/*
SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and cf-service-operator contributors
SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"strconv"

	"github.com/pkg/errors"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	cfv1alpha1 "github.com/sap/cf-service-operator/api/v1alpha1"
	"github.com/sap/cf-service-operator/internal/cf"
	"github.com/sap/cf-service-operator/internal/config"
	"github.com/sap/cf-service-operator/internal/controllers"
	// +kubebuilder:scaffold:imports
)

const (
	LeaderElectionID = "service-operator.cf.cs.sap.com"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(cfv1alpha1.AddToScheme(scheme))
	// +kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	var probeAddr string
	var webhookAddr string
	var webhookCertDir string
	var enableLeaderElection bool
	var enableWebhooks bool
	var clusterResourceNamespace string
	var enableBindingMetadata bool
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.StringVar(&webhookAddr, "webhook-bind-address", ":9443", "The address the webhook endpoint binds to.")
	flag.StringVar(&webhookCertDir, "webhook-tls-directory", "", "The directory containing TLS server key and certificate, as tls.key and tls.crt; defaults to $TMPDIR/k8s-webhook-server/serving-certs.")
	flag.BoolVar(&enableWebhooks, "enableWebhooks", true, "Enable webhooks in controller. May be disabled for local development.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false, "Enable leader election for controller manager. Enabling this will ensure there is only one active controller manager.")
	flag.StringVar(&clusterResourceNamespace, "cluster-resource-namespace", "", "The namespace for secrets in which cluster-scoped resources are found.")
	flag.BoolVar(&enableBindingMetadata, "sap-binding-metadata", false, "Enhance binding secrets by SAP binding metadata by default.")

	opts := zap.Options{
		Development: false,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	if clusterResourceNamespace == "" {
		var err error
		clusterResourceNamespace, err = getInClusterNamespace()
		if err != nil {
			if errors.Is(err, errNotInCluster) {
				setupLog.Error(err, "please supply --cluster-resource-namespace")
			} else {
				setupLog.Error(err, "unexpected error while getting in-cluster namespace")
			}
			os.Exit(1)
		}
	}

	setupLog.Info(
		"Starting",
		"enable-leader-election", enableLeaderElection,
		"metrics-addr", metricsAddr,
		"cluster-resource-namespace", clusterResourceNamespace,
	)

	webhookHost, webhookPort, err := parseAddress(webhookAddr)
	if err != nil {
		setupLog.Error(err, "unable to parse webhook bind address", "controller", "Space")
		os.Exit(1)
	}
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("failed to load config %v\n", err)
		os.Exit(1)
	}
	options := ctrl.Options{
		Scheme: scheme,
		// TODO: disable cache for further resources (e.g. secrets) ?
		Client: client.Options{
			Cache: &client.CacheOptions{
				DisableFor: []client.Object{
					&cfv1alpha1.Space{},
					&cfv1alpha1.ClusterSpace{},
					&cfv1alpha1.ServiceInstance{},
					&cfv1alpha1.ServiceBinding{},
				},
			},
		},
		LeaderElection:                enableLeaderElection,
		LeaderElectionID:              LeaderElectionID,
		LeaderElectionReleaseOnCancel: true,
		Metrics: metricsserver.Options{
			BindAddress: metricsAddr,
		},
		HealthProbeBindAddress: probeAddr,
	}
	if enableWebhooks {
		options.WebhookServer = webhook.NewServer(webhook.Options{
			Host:    webhookHost,
			Port:    webhookPort,
			CertDir: webhookCertDir,
		})
	}
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), options)
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	if err = (&controllers.SpaceReconciler{
		Kind:                     "Space",
		Client:                   mgr.GetClient(),
		Scheme:                   mgr.GetScheme(),
		ClusterResourceNamespace: clusterResourceNamespace,
		ClientBuilder:            cf.NewOrganizationClient,
		HealthCheckerBuilder:     cf.NewSpaceHealthChecker,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Space")
		os.Exit(1)
	}
	if err = (&controllers.SpaceReconciler{
		Kind:                     "ClusterSpace",
		Client:                   mgr.GetClient(),
		Scheme:                   mgr.GetScheme(),
		ClusterResourceNamespace: clusterResourceNamespace,
		ClientBuilder:            cf.NewOrganizationClient,
		HealthCheckerBuilder:     cf.NewSpaceHealthChecker,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ClusterSpace")
		os.Exit(1)
	}
	if err = (&controllers.ServiceInstanceReconciler{
		Client:                   mgr.GetClient(),
		Scheme:                   mgr.GetScheme(),
		ClusterResourceNamespace: clusterResourceNamespace,
		ClientBuilder:            cf.NewSpaceClient,
		Config:                   cfg,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ServiceInstance")
		os.Exit(1)
	}
	if err = (&controllers.ServiceBindingReconciler{
		Client:                   mgr.GetClient(),
		Scheme:                   mgr.GetScheme(),
		ClusterResourceNamespace: clusterResourceNamespace,
		EnableBindingMetadata:    enableBindingMetadata,
		ClientBuilder:            cf.NewSpaceClient,
		Config:                   cfg,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ServiceBinding")
		os.Exit(1)
	}
	if enableWebhooks {
		if err = (&cfv1alpha1.Space{}).SetupWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "Space")
			os.Exit(1)
		}
		if err = (&cfv1alpha1.ClusterSpace{}).SetupWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "ClusterSpace")
			os.Exit(1)
		}
		if err = (&cfv1alpha1.ServiceInstance{}).SetupWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "ServiceInstance")
			os.Exit(1)
		}
		if err = (&cfv1alpha1.ServiceBinding{}).SetupWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "ServiceBinding")
			os.Exit(1)
		}
	}
	// +kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("Starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

const inClusterNamespacePath = "/var/run/secrets/kubernetes.io/serviceaccount/namespace"

var errNotInCluster = errors.New("not running in-cluster")

// Copied from controller-runtime/pkg/leaderelection
func getInClusterNamespace() (string, error) {
	// Check whether the namespace file exists.
	// If not, we are not running in cluster so can't guess the namespace.
	_, err := os.Stat(inClusterNamespacePath)
	if os.IsNotExist(err) {
		return "", errNotInCluster
	} else if err != nil {
		return "", errors.Wrap(err, "error checking namespace file")
	}

	// Load the namespace file and return its content
	namespace, err := os.ReadFile(inClusterNamespacePath)
	if err != nil {
		return "", errors.Wrap(err, "error reading namespace file")
	}
	return string(namespace), nil
}

func parseAddress(address string) (string, int, error) {
	host, p, err := net.SplitHostPort(address)
	if err != nil {
		return "", -1, err
	}
	port, err := strconv.Atoi(p)
	if err != nil {
		return "", -1, err
	}
	return host, port, nil
}
