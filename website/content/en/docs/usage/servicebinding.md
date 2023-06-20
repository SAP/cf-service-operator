---
title: "ServiceBinding resources"
linkTitle: "ServiceBinding resources"
weight: 40
type: "docs"
description: >
  Manage Cloud Foundry service bindings
---

Objects of type `servicebindings.cf.cs.sap.com` represent Cloud Foundry service bindings. For example,
deploying the following descriptor will let the controller deploy a Cloud Foundry credentials binding for the
service instanced managed through the ServiceInstance object referenced by `spec.serviceInstanceName`:

```yaml
apiVersion: cf.cs.sap.com/v1alpha1
kind: ServiceBinding
metadata:
  name: uaa
  namespace: demo
spec:
  serviceInstanceName: uaa
```

If the binding is successful, the controller will store the retrieved binding credentials in a Kubernetes secret
in the namespace of the ServiceBinding object. By default, the secret will have the same name as the ServiceBinding,
and the top-level keys of the credentials object will become secret keys. In the above example, the returned secret would look like this:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: uaa
  namespace: demo
type: Opaque
stringData:
  apiurl: https://api.authentication.sap.hana.ondemand.com
  clientid: sb-myAppName!t39788
  clientsecret: ***
  credential-type: instance-secret
  identityzone: mysubaccount
  identityzoneid: a48fa6e4-df75-4128-abdd-9400d01f3a18
  sburl: https://internal-xsuaa.authentication.sap.hana.ondemand.com
  subaccountid: 56f1b3a2-dbc2-43b1-8bd9-61e0f8290c27
  tenantid: a48fa6e4-df75-4128-abdd-9400d01f3a18
  tenantmode: shared
  uaadomain: authentication.sap.hana.ondemand.com
  url: https://mysubaccount.authentication.sap.hana.ondemand.com
  verificationkey: '-----BEGIN PUBLIC KEY-----MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEArF3/FjAEJx3LTt+UgM65/5LwyHVYownXmOUriLcEO82PiEPFW2n4438VTj5JTvyk42VW5E97lPuXRuVaialRjVDGBmPC9PX8U4ljYYOL3Wgpkid/PkpNe4H/s/U51xJzGTd/XoyuPw64h4v9B71L7bSjOysD5WPzI32/dIHMI0QbZTX2foB8MZqHjhJmVGll2BlT+E7Q+fwQ6bFXL3Ge7fylPu2EgAhR8rnlvrO7hIGIsIGmbGhRmrp38vdIkFsIhanRgtjh2imPh9tBhsYGxUirgzQqnEWQTvE3QZtlfNJ5fK6rht1oO7orkALHzZ2/azBIAojPk4nNx9hEvoCaVwIDAQAB-----END
    PUBLIC KEY-----'
  xsappname: myAppName!t39788
  zoneid: a48fa6e4-df75-4128-abdd-9400d01f3a18
```

The name of the secret can be overridden by setting `spec.secretName`. 
Furthermore, it is possible to render the whole service credentials object into a single key of the target secret by specifying `spec.secretKey`.

Finally, if the binding requires parameters, those can be passed by setting `spec.parameters` and/or `spec.parametersFrom`; 
here the same logic applies as for [ServiceInstance objects](../serviceinstance).

Updating parameters on the ServiceBinding object has no effect by default (because the Cloud Foundry API does not support such updates). However it is possible to enforce a recreation of the Cloud Foundry binding in that situation by setting the annotation `service-operator.cf.cs.sap.com/rotate-on-parameter-change: "true"`.

In addition to this, setting the annotation `service-operator.cf.cs.sap.com/rotate-on-instance-change: "true"` triggers a recreation of the Cloud Foundry binding whenever the referenced service instance changes (due to plan or instance parameter changes).

Recently, SAP published a [specification](https://blogs.sap.com/2022/07/12/the-new-way-to-consume-service-bindings-on-kyma-runtime) to extend binding credentials by additional metadata, to leverage better Kubernetes support in the [xsenv](https://www.npmjs.com/package/@sap/xsenv) library. By default, cf-service-operator will not add these metadata (to remain backwards compatible), but there is a global controller flag `--sap-binding-metadata` that can be used to enhance all created binding secrets by default. In addition, the default behavior can be overridden on a per service binding basis by setting the annotation `service-operator.cf.cs.sap.com/with-sap-binding-metadata: "true"`, or `"false"`.