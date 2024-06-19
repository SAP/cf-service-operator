---
title: "Annotations"
linkTitle: "Annotations"
weight: 20
type: "docs"
description: >
  How to control and optimize the CF Service Operator behavior via annotations.
---

## Annotation Polling Interval Ready

The AnnotationPollingIntervalReady annotation is used to specify the duration of the requeue interval at which the operator polls the status of a Custom Resource after final state ready. It is possible to apply this annotations to Space, ServiceInstance and ServiceBiding CRs. 

By using this annotation, the code allows for flexible configuration of the polling interval, making it easier to adjust the readiness checking frequency based on specific requirements or conditions.

The value of the annotation is a string representing a duration, such as "100m" or "5h".

Usage:

```yaml
apiVersion: cf.cs.sap.com/v1alpha1
kind: ServiceInstance
  metadata:
    annotations:
      service-operator.cf.cs.sap.com/polling-interval-ready: "3h"
```

In the example above the custom resource will be reconcile every three hours after reaching the state Ready.

## Annotation Polling Interval Fail

The AnnotationPollingIntervalFail annotation is used to specify the duration of the requeue interval at which the operator polls the status of a Custom Resource after the final states Creation Failed and Deletion Failed. Currently it is possible to apply this annotations to ServiceInstance custom resource only.

By using this annotation, the code allows for flexible configuration of the polling interval, making it easier to adjust the re-queue frequency after the failure based on specific requirements or conditions.

The value of the annotation is a string representing a duration, such as "20s" or "10m".

Usage:

```yaml
apiVersion: cf.cs.sap.com/v1alpha1
kind: ServiceInstance
  metadata:
    annotations:
      service-operator.cf.cs.sap.com/polling-interval-fail: "5m"
```

In the example above the custom resource will be reconcile every three hours after reaching the final state Failed.
