---
title: "Local environment"
linkTitle: "Local environment"
weight: 10
type: "docs"
description: >
  Run cf-service-operator in a local environment
---

**Prerequisites**
- A Kubernetes cluster running locally, for example provided by [kind](https://kind.sigs.k8s.io) or [minikube](https://minikube.sigs.k8s.io/)
- Jetstack's [cert-manager](https://cert-manager.io) installed in that cluster
- A stable non-loopback address of this host, exported as environment variable `HOST_IP`
- The target cluster's kubeconfig made available as `.kubeconfig` at the root of this repository (can also be a symlink of course).

**Deployment**
- Deploy the [cf-service-operator custom resource definitions](https://github.com/sap/cf-service-operator/blob/main/crds/crd.yaml) to Kubernetes:
  ```
  kubectl apply -f crds/crd.yaml
  ```
- Deploy the [webhook related artifacts](https://github.com/sap/cf-service-operator/blob/main/.local/k8s-resources.yaml) to Kubernetes:
  ```
  envsubst < .local/k8s-resources.yaml | kubectl apply -f -
  ```
- Extract the webhook certificates to be consumed by the controller:
  ```
  mkdir -p .local/ssl
  kubectl get secret cf-service-operator-webhook -o jsonpath='{.data.tls\.key}' \
    | base64 -d > .local/ssl/tls.key
  kubectl get secret cf-service-operator-webhook -o jsonpath='{.data.tls\.crt}' \
    | base64 -d > .local/ssl/tls.crt
  ```
  (paths relative to the root of this repository)

**Launching**

Then the operator can be started by the include `.vscode/launch.json` file ('Launch Operator').

