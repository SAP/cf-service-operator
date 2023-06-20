---
title: "Helm"
linkTitle: "Helm"
weight: 10
type: "docs"
description: >
  Installation by Helm
---

## Helm deployment

The recommended way to deploy cf-service-operator is to use the [Helm chart](https://github.com/sap/cf-service-operator-helm),
also available in packaged form:
- as helm package: [https://sap.github.io/cf-service-operator-helm](https://sap.github.io/cf-service-operator-helm)
- as OCI package: [oci://ghcr.io/sap/cf-service-operator-helm](oci://ghcr.io/sap/cf-service-operator-helm)

The chart does not require any mandatory parameters, so deploying cf-service-operator is as easy as

```bash
helm repo add cf-service-operator https://sap.github.io/cf-service-operator-helm
helm -n cf-service-operator upgrade -i cf-service-operator cf-service-operator/cf-service-operator
```
