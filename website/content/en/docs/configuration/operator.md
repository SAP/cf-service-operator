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

**Kubecibfig Configuration**
cf-service-operator honors the following environment variables:

- `$KUBECONFIG` the path to the kubeconfig used by the operator executable; note that this has lower precedence than the command line flag `-kubeconfig`.

**Cache Configuration**
To optimize the usage of CF resources and reduce the number of API calls, the CF service operator supports an optional caching mechanism. This feature allows resources to be stored in memory and refreshed based on a configurable timeout.
By storing the CF resources in memory, we aim to reduce the number of requests to the CF API and avoid reaching the rate limit.

The cache feature is optional and can be enabled via the environment variable RESOURCE_CACHE_ENABLED, which can have values of true or false by default. Below are the environment variables that control the caching behavior:

- **`RESOURCE_CACHE_ENABLED`**  
  - Description: Determines whether the caching mechanism is enabled or disabled.  
  - Type: Boolean  
  - Default: `false`  
  - Values:  
    - `true`: Enables caching of CF resources.  
    - `false`: Disables the cache, and the operator will fetch CF resources directly from the CF API on each request.

- **`CACHE_TIMEOUT`**  
  Description: This defines the duration after which the cache is refreshed. The cache is refreshed based on the last time it was refreshed.  
  - Type: String  
  - Default: `1m` (1 minute)  
  - Values:  
    - The timeout can be specified in seconds (`s`), minutes (`m`), or hours (`h`). For example:
      - `30s` for 30 seconds  
      - `10m` for 10 minutes  
      - `1h` for 1 hour.

These environment variables can be configured in your `deployment.yaml` file as follows:

```yaml
    env:
    - name: CACHE_TIMEOUT
      value: "{{ .Values.cache.timeout }}"
    - name: RESOURCE_CACHE_ENABLED
      value: "{{ .Values.cache.enabled }}"    
```

Additionally, the corresponding values can be set in the `values.yaml` file of the [helm chart](https://github.com/SAP/cf-service-operator-helm/blob/main/chart/values.yaml)![image](https://github.com/user-attachments/assets/0426656d-fd55-4276-a7c3-e4483ba74218), allowing the operator to be easily configured:

```yaml
# -- Enable Resources Cache
cache:
  # -- Whether to enable the cache
  enabled: false    # default: false
  # -- Cache expiration time
  timeout: 1m     # default: 1m   
```

## Logging

cf-service-operator uses [logr](https://github.com/go-logr) with [zap](https://github.com/uber-go/zap) for logging.
Please check the according documentation for details about how to configure logging.
