---
title: "Adopt existing resources"
linkTitle: "Adopt existing resources"
weight: 10
type: "docs"
description: >
  How to adopt existing Cloud Foundry instances and bindings
---

## Adopt service instances

In order to adopt an existing Cloud Foundry instance, create a Kubernetes ServiceInstance object which specifies `spec.name`
with the name of the existing Cloud Foundry instance, and provide offering, plan, parameters and tags matching the current state.
Such as:

```yaml
apiVersion: cf.cs.sap.com/v1alpha1
kind: ServiceInstance
metadata:
  name: uaa
  namespace: demo
spec:
  # Name of a Space object in the same namespace;
  # this defines the Cloud Foundry space where the instance will be created
  spaceName: k8s
  # Name of service offering to be used
  serviceOfferingName: xsuaa
  # Name of service plan to be used
  servicePlanName: application
  # Explicitly specify the name of the Cloud Foundry instance
  name: <cf instance name>
  # Current paremeters (if any)
  parameters: 
    ...
  # Current tags (if any)
  tags:
    ...
```

After deploying this object, it will enter an error state, complaining that an instance with the same name already exists in Cloud Foundry,
but is not managed by the controller. To solve this, update the Cloud Foundry metadata of the existing instance; more information about how this
controller leverages Cloud Foundry metadata can be found [here](../../concepts/cfmetadata). The update can e.g. be done with the cf command line client:

```bash
cat > /tmp/patch <<END
{
  "metadata": {
    "labels": {
      "service-operator.cf.cs.sap.com/owner": "<ObjectMeta.uid of the Kubernetes ServiceInstance>"
    },
    "annotations": {
      "service-operator.cf.cs.sap.com/generation": "0",
      "service-operator.cf.cs.sap.com/parameter-hash": "0"
    }
  }
}
END
cf curl -X PATCH -H "Content-Type: application/json" /v3/service_instances/<cf instance guid> -d @/tmp/patch
```

More information about this Cloud Foundry API call can be found [here](https://v3-apidocs.cloudfoundry.org/version/3.113.0/index.html#update-a-service-instance).
After some time the controller will consider the instance as managed.

## Adopt service bindings

Works analogously.

The according Cloud Foundry API endpoint is `/v3/service_credential_bindings`.
More information can be found [here](https://v3-apidocs.cloudfoundry.org/version/3.113.0/index.html#update-a-service-credential-binding).

## Using Annotations

An automated way of adopting Cloud Foundry instance is via the Kuberneste annotation `service-operator.cf.cs.sap.com/adopt-cf-resources`.

During the reconciliation of an orphan ServiceInstance and ServiceBinding custom resource, the controller will check if this annotation is present. If the annotation is found then the controller will try to update the Cloud Foundry instance with label `service-operator.cf.cs.sap.com/owner`, and the annotations `service-operator.cf.cs.sap.com/generation` and `service-operator.cf.cs.sap.com/parameter-hash`

Here's an example of how to use this annotation in a ServiceInstance and ServiceBinding:

```yaml
apiVersion: cf.cs.sap.com/v1alpha1
kind: ServiceInstance
metadata:
  name: example-instance
  namespace: demo
  annotations:
    service-operator.cf.cs.sap.com/adopt-cf-resources: "adopt"
spec:
  spaceName: development
  serviceOfferingName: my-service
  servicePlanName: standard
```

```yaml
apiVersion: cf.cs.sap.com/v1alpha1
kind: ServiceBinding
metadata:
  name: example-binding-instance
  namespace: demo
  annotations:
    service-operator.cf.cs.sap.com/adopt-cf-resources: "adopt"  
spec:
  serviceInstanceName: example-instance
```
