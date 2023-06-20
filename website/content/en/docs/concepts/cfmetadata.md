---
title: "Cloud Foundry API usage"
linkTitle: "Cloud Foundry API usage"
weight: 10
type: "docs"
description: >
  Owner and version management through Cloud Foundry metadata
---

Cloud Foundry API version 3 allows storing custom labels and annotations as [metadata](https://docs.cloudfoundry.org/adminguide/metadata.html) on all Cloud Foundry objects.
Both labels and annotations are key-value string pairs. The difference is that labels can be used in list filters.

cf-service-operator persists the following metadata.labels on Cloud Foundry service instances and bindings:
- `service-operator.cf.cs.sap.com/owner`: the Kubernetes `ObjectMeta.uid` of the owning ServiceInstance or ServiceBinding

cf-service-operator persists the following metadata.annotations on Cloud Foundry service instances and bindings:
- `service-operator.cf.cs.sap.com/generation`: the last applied Kubernetes `ObjectMeta.generation`
- `service-operator.cf.cs.sap.com/parameter-hash`: a hash of the last applied instance or binding parameters (after merging).
