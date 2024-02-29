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