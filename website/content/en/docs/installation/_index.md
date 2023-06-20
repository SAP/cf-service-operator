---
title: "Installation"
linkTitle: "Installation"
weight: 10
type: "docs"
description: >
  Overview on available installation methods
---

cf-service-operator relies on four custom resource types, the according definitions can be found 
[here](https://github.com/sap/cf-service-operator/tree/main/crds).
These definitions must be deployed before the operator can be started, otherwise it will fail.
The core of the cf-service-operator installation is the executable built from this repository
(Docker image `cs-devops.common.repositories.cloud.sap/cf-service-operator`).
It implements both the controller logic and validating/mutating webhooks for the above custom resource types.

A valid deployment consists of:
- the custom resource definitions
- the controller deployment
- rbac objects for the controller (service account, (cluster) roles, according (cluster) role bindings)
- a service for the webhooks
- webhook configurations.

Note that it is highly recommended to always activate the webhooks, as they are not only validating, but
also adding default values to our custom resource objects. Running without this mutating functionality
might lead to unexpected behavior.

The following deployment methods are available (recommended is Helm).
