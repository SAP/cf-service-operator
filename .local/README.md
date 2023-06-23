# Instructions for local development

Prerequisite: K8s cluster (kind, minikube) with cert-manager installed.

1. Deploy custom resource definnitions:
   ```bash
   kubectl apply -f crds
   ```

2. Deploy webhook definitions and according objects:
   ```bash
   # replace HOST_IP below with a non-loopback interface address of your desktop
   HOST_IP=1.2.3.4 envsubst < .local/k8s-resources.yaml | kubectl apply -f -
   ```

3. Extract the TLS server keypair:
   ```bash
   .local/getcerts.sh
   ```

4. Copy a sufficently authorized kubeconfig to `.kubeconfig` in the root folder of this repository.

Afterwards (if using vscode) it should be possible to start the operator with the included launch configuration.
