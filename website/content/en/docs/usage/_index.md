---
title: "Usage"
linkTitle: "Usage"
weight: 30
type: "docs"
description: >
  How to use cf-service-operator
---

cf-service-operator introduces the following resource types:

* [Space](./space) and [ClusterSpace](./clusterspace): used to represent or manage Cloud Foundry spaces at namespace or cluster level.
  A ServiceInstance object always references exactly one Space or ClusterSpace.

* [ServiceInstance](./serviceinstance): used to manage (create/update) a Cloud Foundry service instance. 
  A ServiceInstance references a Space or ClusterSpace, the Cloud Foundry service offering/plan, and optionally defines service parameters.

* [ServiceBinding](./servicebinding): used to manage (create/update) a Cloud Foundry service binding.
  A ServiceBinding references a ServiceInstance Object, and defines the Kubernetes secret used to store the retrieved service key.
  Optionally binding parameters can be specified.

