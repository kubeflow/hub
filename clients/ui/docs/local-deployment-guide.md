# Local Deployment Guide

## Local kubernetes deployment of Kubeflow Hub

To test the BFF locally without mocking the k8s calls the Model Registry backend can be deployed locally using kind.

### Prerequisites

The following tools need to be installed in your local environment:

* Docker - [Docker Instructions](https://www.docker.com)
* kubectl - [Instructions](https://kubernetes.io/docs/tasks/tools/#kubectl)
* kind - [Instructions](https://kind.sigs.k8s.io/docs/user/quick-start/#installation)

Note: all of the above tools can be installed using your OS package manager, this is the preferred method.

### Setup

#### 1. Create a kind cluster

Create a local cluster for running the MR backend using the following command:

```shell
kind create cluster
```

Kind will start creating a new local cluster for you to deploy, once it has completed verify you can access the cluster
using kubectl by running:

```shell
kubectl cluster-info
```

If everything is working correctly you should see output similar to:

``` shell
Kubernetes control plane is running at https://127.0.0.1:58635
CoreDNS is running at https://127.0.0.1:58635/api/v1/namespaces/kube-system/services/kube-dns:dns/proxy
```

#### 2. Create kubeflow namespace and switch to it

Create a namespace for model registry to run in, by default this is kubeflow, run:

```shell
kubectl create namespace kubeflow
```
then, switch to that  kubeflow namespace


```shell
  kubectl config set-context --current --namespace=kubeflow
```

#### 3. Deploy Model Registry to cluster

You can now deploy the MR backend to your newly created cluster using the kustomize configs in the MR repository by
running:

```shell
kubectl apply -k "https://github.com/kubeflow/hub/manifests/kustomize/overlays/db"
```

Wait for the model registry deployment to spin up, alternatively run:

```shell
kubectl wait --for=condition=available -n kubeflow deployment/model-registry-deployment --timeout=1m
```

This command will return when the cluster is ready. To verify this now run:

```shell
kubectl get pods -n kubeflow
```

Two pods should be listed, `model-registry-db-xxx` and `model-registry-deployment-yyy` both should have a status of `Running`.

##### NOTE: Issues running on arm64 architecture

There is currently an issue deploying to an arm64 device such as a Mac with an M-series chip. This is because the MySql
image tag deployed by the manifests does not have an arm64 compatible image. To work around this you can use a modified
manifest deployed in a fork of the repo - you can use this by running the below command instead of the first command in
section 3 of this guide.

```shell
kubectl apply -k "https://github.com/alexcreasy/model-registry/manifests/kustomize/overlays/db?ref=kind"
```

Note: an issue has been filed regarding this ticket here:

* [#266 Cannot deploy to k8s on AArch64 nodes using manifests in repo](https://github.com/kubeflow/hub/issues/266)

#### 4. Setup a port forward to the service

In order to access the MR REST API locally you need to forward a local port to 8080 on the MR service. Run the following
command: (Note: this command starts a long-running service and will not exit the shell)

```shell
kubectl port-forward svc/model-registry-service -n kubeflow 8080:8080
```

Note: you can change the local forwarded port by changing the first port value, e.g. `4000:8080` will forward port 4000
to the MR service.

#### 5. Test the service

In a separate terminal window to the previous step, test the service by querying one of the rest endpoints, for example:

```shell
curl http://localhost:8080/api/model_registry/v1/registered_models
```

You should receive a 200 response if everything is working correctly, the body should look like:

```json
{"items":[],"nextPageToken":"","pageSize":0,"size":0}
```

#### 6. (Optional) Run BFF locally in Dev Mode

To access your local kind cluster when running the BFF locally, you can use the `DEV_MODE` option. This is useful for when
you want to test live changes on real cluster. To do so, simply run:

```shell
make run DEV_MODE=true
```

You can also specify the port you are forwarding to if it is something other than 8080:

```shell
make run DEV_MODE=true DEV_MODE_MODEL_REGISTRY_PORT=8081
```
