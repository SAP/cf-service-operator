---
title: "Adopt existing resources"
linkTitle: "Adopt existing resources"
weight: 10
type: "docs"
description: >
  How to adopt existing Cloud Foundry instances and bindings
---

## Adopt service instances

To adopt an existing Cloud Foundry instance, create a Kubernetes ServiceInstance object that specifies the `spec.name` with the name of the existing Cloud Foundry instance and provides the offering, plans, parameters, and tags matching the current state.

Such as:

```yaml
apiVersion: cf.cs.sap.com/v1alpha1
kind: ServiceInstance
metadata:
  name: example-instance
  namespace: demo
spec:
  # Name of a Space object in the same namespace;
  # this defines the Cloud Foundry space where the instance will be created
  spaceName: k8s
  # Name of service offering to be used
  serviceOfferingName: xsuaa
  # Name of service plan to be used
  servicePlanName: standard
  # Explicitly specify the name of the Cloud Foundry instance
  name: <cf instance name>
  # Current paremeters (if any)
  parameters: 
    ...
  # Current tags (if any)
  tags:
    ...
```

After deploying this object, it will enter an error state, complaining that an instance with the same name already exists in Cloud Foundry, but is not managed by the controller.

Check the status of the Instance. The following error is expected:
`cfclient error (CF-UnprocessableEntity|10008): The service instance name is taken`

To solve this, the Cloud Foundry metadata of the existing instance must be updated.

>More information about how this controller leverages Cloud Foundry metadata can be found [here](../../concepts/cfmetadata).

The CF Service Operator provides a way to adopt orphan instances via a Kubernetes Annotation.

### Using the annotations adopt-cf-resources

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
  spaceName: k8s
  serviceOfferingName: xsuaa
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

After some time the controller will consider the ServiceInstance and ServiceBinding as managed.
