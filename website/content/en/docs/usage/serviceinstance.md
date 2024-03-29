---
title: "ServiceInstance resources"
linkTitle: "ServiceInstance resources"
weight: 30
type: "docs"
description: >
  Manage Cloud Foundry service instances
---

Objects of type `serviceinstances.cf.cs.sap.com` represent Cloud Foundry service instances. For example,
deploying the following descriptor will let the controller create or update a Cloud Foundry instance 
of the service offering 'xsuaa' with plan 'application',
in the Cloud Foundry space referenced through the Space object given in `spec.spaceName`:

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
```

In order to reference a ClusterSpace instead of a Space, the definition would change like this:

```yaml
apiVersion: cf.cs.sap.com/v1alpha1
kind: ServiceInstance
metadata:
  name: uaa
  namespace: demo
spec:
  # Name of a ClusterSpace object;
  # this defines the Cloud Foundry space where the instance will be created
  clusterSpaceName: k8s
  # Name of service offering to be used
  serviceOfferingName: xsuaa
  # Name of service plan to be used
  servicePlanName: application
```

Furthermore, instead of specifying service offering and plan by name, it is possible to directly provide
the guid of the service plan, such as:

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
  # Guid of service plan to be used
  servicePlanGuid: 432bd9db-20e2-4997-825f-e4a937705b87
```

Instance parameters can be passed like this:

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
  # Instance parameters (inline)
  # Caveat: never specify sensitive parameters here, but use parametersFrom instead!
  parameters:
    xsappname: myAppName
  # Instance parameters (by secret key reference)
  parametersFrom:
  - name: uaa-params
    key: parameters.json
```

Following the logic implemented by similar controllers (e.g. the K8s service catalog) it is allowed to specify both `parameters` and `parametersFrom`,
but it is considered an error if a top level key occurs in more than one of the sources.

In addition, it is possible to annotate custom instance tags, such as:

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
  # List of custom tags
  tags:
  - uaa
  - xsuaa
  - authentication
```

Finally there is a way to tweak the controller's behavior in case the creation of a service instance fails in Cloud Foundry.
By default, in all kinds of error situations, the controller will send an update request, in order to trigger a reconciliation of the instance.
However some service brokers do not really support this approach, specifically when the initial creation of the instance has failed.
To overcome this, the annotation `service-operator.cf.cs.sap.com/recreate-on-creation-failure: "true"` may be set on the service instance object.
If present, the controller will drop and recreate instances which are in a failed creation state (which precisely means that the `LastOperation` reported by the Cloud Foundry API is of type `create` and state `failed`).