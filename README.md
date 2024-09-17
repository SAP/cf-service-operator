# Kubernetes Operator For Cloud Foundry™ Services

[![REUSE status](https://api.reuse.software/badge/github.com/SAP/cf-service-operator)](https://api.reuse.software/info/github.com/SAP/cf-service-operator)

Disclaimer: Cloud Foundry is a registered trademark of THE LINUX FOUNDATION. Any rights therein are reserved to THE LINUX FOUNDATION. Any use by this project is for referential purposes only and does not indicate any sponsorship, endorsement or affiliation between CloudFoundry and this project.

## About this project

This repository adds native support for Cloud Foundry spaces, service instances and service bindings to Kubernetes clusters.
It contains custom resource definitions

- `spaces.cf.cs.sap.com` (kind `Space`)
- `clusterspaces.cf.cs.sap.com` (kind `ClusterSpace`)
- `serviceinstances.cf.cs.sap.com` (kind `ServiceInstance`)
- `servicebindings.cf.cs.sap.com` (kind `ServiceBinding`)

and an according operator reconciling resources of these types.

A typical usage could look as follows:

```yaml
---
apiVersion: cf.cs.sap.com/v1alpha1
kind: Space
metadata:
  name: k8s
spec:
  organizationName: my-org 
  authSecretName: k8s-space
---
apiVersion: v1
kind: Secret
metadata:
  name: k8s-space
stringData:
  url: "<cf api url>"
  username: "<username>"
  password: "<password>"
---
apiVersion: cf.cs.sap.com/v1alpha1
kind: ServiceInstance
metadata:
  name: uaa
spec:
  spaceName: k8s
  serviceOfferingName: xsuaa
  servicePlanName: application
---
apiVersion: cf.cs.sap.com/v1alpha1
kind: ServiceBinding
metadata:
  name: uaa
spec:
  serviceInstanceName: uaa
```

When reconciling these objects, the operator will ensure that according API entities (space, instance, binding) are maintained in the
targeted Cloud Foundry environment.

The scope of this project is comparable with the [SAP BTP Service Operator](https://github.com/SAP/sap-btp-service-operator).
But other than that operator, the Cloud Foundry Service Operator provided by this repository manages services and bindings through
Cloud Foundry APIs, instead of using native BTP APIs.

## Requirements and Setup

The recommended deployment method is to use the [Helm chart](https://github.com/sap/cf-service-operator-helm):

```bash
helm upgrade -i cf-service-operator oci://ghcr.io/sap/cf-service-operator-helm/cf-service-operator
```

## Documentation

The project's documentation can be found here: [https://sap.github.io/cf-service-operator](https://sap.github.io/cf-service-operator).  
The API reference is here: [https://pkg.go.dev/github.com/sap/cf-service-operator](https://pkg.go.dev/github.com/sap/cf-service-operator).

## Support, Feedback, Contributing

This project is open to feature requests/suggestions, bug reports etc. via [GitHub issues](https://github.com/SAP/cf-service-operator/issues). Contribution and feedback are encouraged and always welcome. For more information about how to contribute, the project structure, as well as additional contribution information, see our [Contribution Guidelines](CONTRIBUTING.md).

As artificial intelligence evolves, AI-generated code is becoming valuable for many software projects, including open-source initiatives. While we recognize the potential benefits of incorporating AI-generated content into our open-source projects there a certain requirements that need to be reflected and adhered to when making contributions.

Please see our [guideline for AI-generated code contributions to SAP Open Source Software Projects](CONTRIBUTING_USING_GENAI.md) for these requirements.

## Code of Conduct

We as members, contributors, and leaders pledge to make participation in our community a harassment-free experience for everyone. By participating in this project, you agree to abide by its [Code of Conduct](https://github.com/SAP/.github/blob/main/CODE_OF_CONDUCT.md) at all times.

## Licensing

Copyright 2024 SAP SE or an SAP affiliate company and cf-service-operator contributors. Please see our [LICENSE](LICENSE) for copyright and license information. Detailed information including third-party components and their licensing/copyright information is available [via the REUSE tool](https://api.reuse.software/info/github.com/SAP/cf-service-operator).
