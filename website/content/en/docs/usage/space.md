---
title: "Space resources"
linkTitle: "Space resources"
weight: 10
type: "docs"
description: >
  Define Cloud Foundry spaces on namespace level
---

Objects of type `spaces.cf.cs.sap.com` represent Cloud Foundry spaces at the scope of a Kubernetes namespace.
Spaces can be defined as managed or unmanaged, and can be referenced by `ServiceInstance` objects deployed into the same namespace.

## Unmanaged spaces

A `Space` object is called unmanaged if it just references an already existing Cloud Foundry space by its GUID via `spec.guid`.
The Cloud Foundry space will not be touched at all by the controller.
It just serves as a reference for `ServiceInstance` objects to be linked with the underlying Cloud Foundry space. For example:

```yaml
apiVersion: cf.cs.sap.com/v1alpha1
kind: Space
metadata:
  name: k8s
  namespace: demo
spec:
  # Cloud Foundry space guid
  guid: 0a61a2ea-0326-43b6-bc08-3510bd32c5e8
  # Secret containing authentication details (Cloud Foundry API address, username, password)
  authSecretName: k8s-space
```

The referenced secret (expected in the same namespace) contains connection details for the space:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: k8s-space
  namespace: demo
stringData:
  url: https://api.cf.sap.hana.ondemand.com
  username: "<email>"
  password: "<password>"
```

Here the user specified in `username` should have at least the space developer role in Cloud Foundry.

## Managed spaces

A managed `Space` is not linked with an existing Cloud Foundry space. Instead it contains a reference to the target Cloud Foundry
organization as `spec.organizationName`.
The controller will then try to maintain a Cloud Foundry space called `metadata.name` in that organization, and also delete it once the `Space` object is deleted.
For example:

```yaml
apiVersion: cf.cs.sap.com/v1alpha1
kind: Space
metadata:
  name: k8s
  namespace: demo
spec:
  # Cloud Foundry organization
  organizationName: my-org 
  # Secret containing authentication details (Cloud Foundry API address, username, password)
  authSecretName: k8s-space
```

The name of the Cloud Foundry space can be overridden by specifying `spec.name`.
The referenced secret looks the same as in the unmanaged case, but in addition,
it is possible to provide different credentials for the space management, such as:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: k8s-space
  namespace: demo
stringData:
  url: https://api.cf.sap.hana.ondemand.com
  username: "<email>"
  password: "<password>"
  org_username: "<email>"
  org_password: "<password>"
```

The user specified as `org_username` is then expected to have the organization manager role in Cloud Foundry.
If omitted, the user specified in `username` will be used to create/update/delete the space (and that user of course must be an organization manager in that case).

Finally, the user specified in `username` will be added as a space manager to the space.
