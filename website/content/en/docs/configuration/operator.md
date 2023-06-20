---
title: "Operator startup options"
linkTitle: "Operator startup options"
weight: 10
type: "docs"
description: >
  Command line configuration options, environment variables
---

## Command line parameters

cf-service-operator accepts the following command line flags:

```
Usage of manager:
  -cluster-resource-namespace string
      The namespace for secrets in which cluster-scoped resources are found.
  -health-probe-bind-address string
      The address the probe endpoint binds to. (default ":8081")
  -kubeconfig string
      Paths to a kubeconfig. Only required if out-of-cluster.
  -leader-elect
      Enable leader election for controller manager.
      Enabling this will ensure there is only one active controller manager.
  -metrics-bind-address string
      The address the metric endpoint binds to. (default ":8080")
  -sap-binding-metadata
      Enhance binding secrets by SAP binding metadata by default.
  -webhook-bind-address string
      The address the webhook endpoint binds to. (default ":9443")
  -webhook-tls-directory string
      The directory containing tls server key and certificate, as tls.key and tls.crt;
      defaults to $TMPDIR/k8s-webhook-server/serving-certs
  -zap-devel
      Development Mode defaults(encoder=consoleEncoder,logLevel=Debug,stackTraceLevel=Warn);
      Production Mode defaults(encoder=jsonEncoder,logLevel=Info,stackTraceLevel=Error) (default: true)
  -zap-encoder value
      Zap log encoding (one of 'json' or 'console')
  -zap-log-level value
      Zap Level to configure the verbosity of logging. Can be one of 'debug', 'info', 'error',
      or any integer value > 0 which corresponds to custom debug levels of increasing verbosity
  -zap-stacktrace-level value
      Zap Level at and above which stacktraces are captured (one of 'info', 'error', 'panic').
  -zap-time-encoding value
      Zap time encoding (one of 'epoch', 'millis', 'nano', 'iso8601', 'rfc3339' or 'rfc3339nano').
      Defaults to 'epoch'.
```

Notes:
- When running in-cluster, then `-cluster-resource-namespace` defaults to the operator's namespace; otherwise this flag is mandatory.
- The logic for looking up the kubeconfig file is
  - path provided as `-kubeconfig` (if present)
  - value of environment variable `$KUBECONFIG`(if present)
  - in-cluster config (if running in cluster)
  - `$HOME/.kube/config` (if existing)
  Thus when running in-cluster, it is usually not necessary to specify the flag or environment variable, such that the operator just
  uses the according service account's kubeconfig.
- Enabling leader election is mandatory whenever there is a chance that more than one replica is running; because running multiple replicas
  without leader election will lead to concurrent active control loops handling the same set of resources, probably ending up with split brain situations and
  potential inconsistencies. Leader election is disabled by default, which is fine for development purposes, or situations where the connectivity to
  the API server is not reliable (in that case, still, only one replica must be running of course).

## Environment variables

cf-service-operator honors the following environment variables:

- `$KUBECONFIG` the path to the kubeconfig used by the operator executable; note that this has lower precedence than the command line flag `-kubeconfig`.

## Logging

cf-service-operator uses [logr](https://github.com/go-logr) with [zap](https://github.com/uber-go/zap) for logging.
Please check the according documentation for details about how to configure logging.
