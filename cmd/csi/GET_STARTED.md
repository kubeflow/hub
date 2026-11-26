# Get Started

Embark on your journey with this custom storage initializer by exploring a simple hello-world example. Learn how to seamlessly integrate and leverage the power of this tool in just a few steps.

## Prerequisites

* Install [Kind](https://kind.sigs.k8s.io/docs/user/quick-start) (Kubernetes in Docker) to run local Kubernetes cluster with Docker container nodes.
* Install the [Kubernetes CLI (kubectl)](https://kubernetes.io/docs/tasks/tools/), which allows you to run commands against Kubernetes clusters.
* Install the [Kustomize](https://kustomize.io/), which allows you to customize app configuration.

## Environment Preparation

We assume all [prerequisites](#prerequisites) are satisfied at this point.

All the following instructions should be performed from the model-registry root directory.

### Create the environment

1. After having kind installed, create a kind cluster with:
    ```bash
    kind create cluster
    ```

2. Configure `kubectl` to use kind context
    ```bash
    kubectl config use-context kind-kind
    ```

3. Setup local deployment of *Kserve* using the provided *Kserve quick installation* script
    ```bash
    curl -s "https://raw.githubusercontent.com/kserve/kserve/release-0.14/hack/quick_install.sh" | bash
    ```

4. Install *model registry* in the local cluster

    [Optional] Use local model registry container image:

    ```bash
    TAG=$(git rev-parse HEAD) && \
    MR_IMG=kubeflow/hub:$TAG && \
    make IMG_VERSION=$TAG image/build && \
    kind load docker-image $MR_IMG
    ```

    then:

    ```bash
    bash ./test/scripts/install_modelregistry.sh -i $MR_IMG
    ```

> [!NOTE]
> The `./test/scripts/install_modelregistry.sh` will make some change to [base/kustomization.yaml](../../manifests/kustomize/base/kustomization.yaml) that you DON'T need to commit!!

5. [Optional] Use local CSI container image

    Either, using the local model-registry library as dependency:
    ```bash
    TAG=$(git rev-parse HEAD)
    IMG=ghcr.io/kubeflow/hub/storage-initializer:$TAG && \
    make IMG_VERSION=$TAG IMG_REPO=model-registry-storage-initializer image/build && \
    kind load docker-image $IMG
    ```

## First InferenceService with ModelRegistry URI

In this tutorial, you will deploy an InferenceService with a predictor that will load a model indexed into the model registry, the indexed model refers to a scikit-learn model trained with the [iris](https://archive.ics.uci.edu/ml/datasets/iris) dataset. This dataset has three output class: Iris Setosa, Iris Versicolour, and Iris Virginica.

You will then send an inference request to your deployed model in order to get a prediction for the class of iris plant your request corresponds to.

Since your model is being deployed as an InferenceService, not a raw Kubernetes Service, you just need to provide the storage location of the model using the `model-registry://` URI format and it gets some super powers out of the box.


### Register a Model into ModelRegistry

Apply `Port Forward` to the model registry service in order to being able to interact with it from the outside of the cluster.
```bash
kubectl port-forward --namespace kubeflow svc/model-registry-service 8080:8080
```

And then (in another terminal):
```bash
export MR_HOSTNAME=localhost:8080
```

Then, in the same terminal where you exported `MR_HOSTNAME`, perform the following actions:
1. Register an empty `RegisteredModel`
    ```bash
    curl --silent -X 'POST' \
      "$MR_HOSTNAME/api/model_registry/v1alpha3/registered_models" \
      -H 'accept: application/json' \
      -H 'Content-Type: application/json' \
      -d '{
      "description": "Iris scikit-learn model",
      "name": "iris"
    }'
    ```

    Expected output:
    ```bash
    {"createTimeSinceEpoch":"1709287882361","customProperties":{},"description":"Iris scikit-learn model","id":"1","lastUpdateTimeSinceEpoch":"1709287882361","name":"iris"}
    ```

2. Register the first `ModelVersion`
    ```bash
    curl --silent -X 'POST' \
      "$MR_HOSTNAME/api/model_registry/v1alpha3/model_versions" \
      -H 'accept: application/json' \
      -H 'Content-Type: application/json' \
      -d '{
      "description": "Iris model version v1",
      "name": "v1",
      "registeredModelID": "1"
    }'
    ```

    Expected output:
    ```bash
    {"createTimeSinceEpoch":"1709287890365","customProperties":{},"description":"Iris model version v1","id":"2","lastUpdateTimeSinceEpoch":"1709287890365","name":"v1"}
    ```

3. Register the raw `ModelArtifact`
    This artifact defines where the actual trained model is stored, i.e., `gs://kfserving-examples/models/sklearn/1.0/model`

    ```bash
    curl --silent -X 'POST' \
      "$MR_HOSTNAME/api/model_registry/v1alpha3/model_versions/2/artifacts" \
      -H 'accept: application/json' \
      -H 'Content-Type: application/json' \
      -d '{
      "description": "Model artifact for Iris v1",
      "uri": "gs://kfserving-examples/models/sklearn/1.0/model",
      "state": "UNKNOWN",
      "name": "iris-model-v1",
      "modelFormatName": "sklearn",
      "modelFormatVersion": "1",
      "artifactType": "model-artifact"
    }'
    ```

    Expected output:
    ```bash
    {"artifactType":"model-artifact","createTimeSinceEpoch":"1709287972637","customProperties":{},"description":"Model artifact for Iris v1","id":"1","lastUpdateTimeSinceEpoch":"1709287972637","modelFormatName":"sklearn","modelFormatVersion":"1","name":"iris-model-v1","state":"UNKNOWN","uri":"gs://kfserving-examples/models/sklearn/1.0/model"}
    ```

> [!NOTE]
> Double check the provided IDs are the expected ones.

### Apply the `ClusterStorageContainer` resource

Retrieve the model registry service and MLMD port:
```bash
MODEL_REGISTRY_SERVICE=model-registry-service
MODEL_REGISTRY_REST_PORT=$(kubectl get svc/$MODEL_REGISTRY_SERVICE -n kubeflow --output jsonpath='{.spec.ports[0].targetPort}' )
```

Apply the cluster-scoped `ClusterStorageContainer` CR to setup configure the `model registry storage initilizer` for `model-registry://` URI formats.

```bash
kubectl apply -f - <<EOF
apiVersion: "serving.kserve.io/v1alpha1"
kind: ClusterStorageContainer
metadata:
  name: mr-initializer
spec:
  container:
    name: storage-initializer
    image: $IMG
    env:
    - name: MODEL_REGISTRY_BASE_URL
      value: "$MODEL_REGISTRY_SERVICE.kubeflow.svc.cluster.local:$MODEL_REGISTRY_REST_PORT"
    - name: MODEL_REGISTRY_SCHEME
      value: "http"
    resources:
      requests:
        memory: 100Mi
        cpu: 100m
      limits:
        memory: 1Gi
  supportedUriFormats:
    - prefix: model-registry://

EOF
```

> [!NOTE]
> As `$IMG` you could use either the one created during [env preparation](#environment-preparation) or any other remote img in the container registry.

### Create an `InferenceService`

1. Create a namespace
    ```bash
    kubectl create namespace kserve-test
    ```

2. Create the `InferenceService`
    ```bash
    kubectl apply -n kserve-test -f - <<EOF
    apiVersion: "serving.kserve.io/v1beta1"
    kind: "InferenceService"
    metadata:
      name: "iris-model"
    spec:
      predictor:
        model:
          modelFormat:
            name: sklearn
          storageUri: "model-registry://iris/v1"
    EOF
    ```

3. Check `InferenceService` status
    ```bash
    kubectl get inferenceservices iris-model -n kserve-test
    ```

4. Determine the ingress IP and ports
    ```bash
    kubectl get svc istio-ingressgateway -n istio-system
    ```

    And then:
    ```bash
    INGRESS_GATEWAY_SERVICE=$(kubectl get svc --namespace istio-system --selector="app=istio-ingressgateway" --output jsonpath='{.items[0].metadata.name}')
    kubectl port-forward --namespace istio-system svc/${INGRESS_GATEWAY_SERVICE} 8081:80
    ```

    After that (in another terminal):
    ```bash
    export INGRESS_HOST=localhost
    export INGRESS_PORT=8081
    ```

5. Perform the inference request

    Prepare the input data:
    ```bash
    cat <<EOF > "/tmp/iris-input.json"
    {
      "instances": [
        [6.8,  2.8,  4.8,  1.4],
        [6.0,  3.4,  4.5,  1.6]
      ]
    }
    EOF
    ```

    If you do not have DNS, you can still curl with the ingress gateway external IP using the HOST Header.
    ```bash
    SERVICE_HOSTNAME=$(kubectl get inferenceservice iris-model -n kserve-test -o jsonpath='{.status.url}' | cut -d "/" -f 3)
    curl -v -H "Host: ${SERVICE_HOSTNAME}" -H "Content-Type: application/json" "http://${INGRESS_HOST}:${INGRESS_PORT}/v1/models/iris-model:predict" -d @/tmp/iris-input.json
    ```
