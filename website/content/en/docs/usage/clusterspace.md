---
title: "ClusterSpace resources"
linkTitle: "ClusterSpace resources"
weight: 20
type: "docs"
description: >
  Define Cloud Foundry spaces on cluster level
---

Objects of type `clusterspaces.cf.cs.sap.com` represent a Cloud Foundry spaces at Kubernetes cluster scope.
Cluster Spaces can be defined as managed or unmanaged, and can be referenced by `ServiceInstance` objects across the whole cluster.

## Unmanaged cluster spaces

A `ClusterSpace` object is called unmanaged if it just references an already existing Cloud Foundry space by its GUID via `spec.guid`.
The Cloud Foundry space will not be touched at all by the controller.
It just serves as a reference for `ServiceInstance` objects to be linked with the underlying Cloud Foundry space. For example:

```yaml
apiVersion: cf.cs.sap.com/v1alpha1
kind: ClusterSpace
metadata:
  name: k8s
spec:
  # Cloud Foundry space guid
  guid: 0a61a2ea-0326-43b6-bc08-3510bd32c5e8
  # Secret containing authentication details (Cloud Foundry API address, username, password)
  authSecretName: k8s-space
```

The referenced secret must have the same structure used for [Space](../space/#unmanaged-spaces) objects, and by default is looked up in the namespace where the operator is deployed (but this can be overridden by command line flag, see the [configuration section](../../configuration) for more details).

## Managed cluster spaces

A managed `ClusterSpace` is not linked with an existing Cloud Foundry space. Instead it contains a reference to the target Cloud Foundry
organization as `spec.organizationName`.
The controller will then try to maintain a Cloud Foundry space called `metadata.name` in that organization, and also delete it once the `Space` object is deleted.
For example:

```yaml
apiVersion: cf.cs.sap.com/v1alpha1
kind: ClusterSpace
metadata:
  name: k8s
spec:
  # Cloud Foundry organization
  organizationName: my-org 
  # Secret containing authentication details (Cloud Foundry API address, username, password)
  authSecretName: k8s-space
```

The name of the Cloud Foundry space can be overridden by specifying `spec.name`.
The referenced secret must have the same structure used for [Space](../space/#managed-spaces) objects, and by default is looked up in the namespace where the operator is deployed (but this can be overridden by command line flag, see the [configuration section](../../configuration) for more details).
