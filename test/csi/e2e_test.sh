#!/bin/bash

set -e
set -o xtrace

# This test assumes there is a Kubernetes environment up and running.
# It could be either a remote one or a local one (e.g., using KinD or minikube).

DIR="$(dirname "$0")"

source ./${DIR}/test_utils.sh

KUBECTL=${KUBECTL:-"kubectl"}

# You can provide a local version of the model registry storage initializer
# In that case, assure that is visible to the local k8s env, e.g., using
# `kind load docker-image $MRCSI_IMG`
MRCSI_IMG=${MRCSI_IMG:-"ghcr.io/kubeflow/hub/storage-initializer:main"}

KSERVE_VERSION=${KSERVE_VERSION:-"0.16"}
MODELREGISTRY_VERSION=${MODELREGISTRY_VERSION:-"v0.3.2"}
MODELREGISTRY_CSI=${MODELREGISTRY_CSI:-"v0.3.2"}

# FIXME: Use ghcr.io/kubeflow/hub/* images after v0.3.9

# You can provide a local model registry container image
MR_IMG=${MR_IMG:-"ghcr.io/kubeflow/model-registry/server:$MODELREGISTRY_VERSION"}
# You can provide a local model registry storage initializer container image
MR_CSI_IMG=${MR_CSI_IMG:-"ghcr.io/kubeflow/model-registry/storage-initializer:$MODELREGISTRY_CSI"}

# Export variables so they're available to setup_test_env.sh
export KSERVE_VERSION
export MODELREGISTRY_VERSION
export MODELREGISTRY_CSI
export MR_IMG
export MR_CSI_IMG
export MRCSI_IMG

# Check if KUBECTL is a valid command
if [ ! command -v "$KUBECTL" > /dev/null 2>&1 ]; then
    echo "KUBECTL command not found at: $KUBECTL"
    exit 1
fi

if [ ! "$KUBECTL" cluster-info > /dev/null 2>&1 ]; then
    echo "Cluster not available!"
    exit 1
fi

# Setup the environment
./${DIR}/setup_test_env.sh

# Wait for Knative webhook to be ready (critical for KServe 0.15)
echo "Waiting for Knative webhook to be ready..."
kubectl wait --for=condition=Ready pod -l app=webhook -n knative-serving --timeout=5m || {
    echo "Warning: Knative webhook not ready, checking status..."
    kubectl get pods -n knative-serving
    kubectl describe pod -l app=webhook -n knative-serving
}

# Wait for KServe webhook to be ready
echo "Waiting for KServe webhook to be ready..."
kubectl wait --for=condition=Ready pod -l control-plane=kserve-controller-manager -n kserve --timeout=5m || {
    echo "Warning: KServe webhook not ready, checking status..."
    kubectl get pods -n kserve
    kubectl describe pod -l control-plane=kserve-controller-manager -n kserve
}

echo "All webhooks ready, proceeding with tests..."

# Apply the port forward to access the model registry
NAMESPACE=${NAMESPACE:-"kubeflow"}
TESTNAMESPACE=${TESTNAMESPACE:-"test"}
MODEL_REGISTRY_SERVICE=model-registry-service
MODEL_REGISTRY_REST_PORT=$(kubectl get svc/$MODEL_REGISTRY_SERVICE -n $NAMESPACE --output jsonpath='{.spec.ports[0].targetPort}')
INGRESS_HOST="localhost:8081"

echo "======== Preparing test environment ========"

echo "Applying Model Registry custom storage initializer ..."

kubectl apply -f - <<EOF
apiVersion: "serving.kserve.io/v1alpha1"
kind: ClusterStorageContainer
metadata:
  name: mr-initializer
spec:
  container:
    name: storage-initializer
    image: $MR_CSI_IMG
    imagePullPolicy: IfNotPresent
    env:
    - name: MODEL_REGISTRY_BASE_URL
      value: "$MODEL_REGISTRY_SERVICE.$NAMESPACE.svc.cluster.local:$MODEL_REGISTRY_REST_PORT"
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

echo "Serving the istio ingress gateway on $INGRESS_HOST ..."

INGRESS_GATEWAY_SERVICE=$(kubectl get svc -n istio-system --selector="app=istio-ingressgateway" --output jsonpath='{.items[0].metadata.name}')
kubectl port-forward -n istio-system svc/${INGRESS_GATEWAY_SERVICE} 8081:80 &
ingress_pf_pid=$!

wait_for_port 8081

echo "Creating dummy input data for testing ..."

cat <<EOF > "/tmp/iris-input.json"
{
  "instances": [
    [6.8,  2.8,  4.8,  1.4],
    [6.0,  3.4,  4.5,  1.6]
  ]
}
EOF

echo "======== Finished preparing test environment ========"

# Array to track background process PIDs
declare -a scenario_pids=()
declare -a scenario_results=()

# Generic function to run a test scenario
# Parameters:
#   $1: scenario_num (1-4)
#   $2: mr_port (port for model registry port-forward)
#   $3: mr_namespace (namespace where model registry is running: kubeflow or test)
#   $4: model_name (registered model name)
#   $5: version_name (model version name, or empty for latest)
#   $6: inference_service_name (name of the InferenceService)
#   $7: artifact_name (name of the model artifact)
#   $8: description (test scenario description)
run_scenario() {
    local scenario_num=$1
    local mr_port=$2
    local mr_namespace=$3
    local model_name=$4
    local version_name=$5
    local inference_service_name=$6
    local artifact_name=$7
    local description=$8

    local kserve_namespace="kserve-test-${scenario_num}"
    local mr_hostname="localhost:${mr_port}"

    # Determine if using custom MR service
    local use_custom_mr=false
    if [ "$mr_namespace" = "test" ]; then
        use_custom_mr=true
    fi

    echo "======== Scenario ${scenario_num} - ${description} ========"

    # Create namespace
    if ! kubectl get namespace $kserve_namespace &> /dev/null; then
       kubectl create namespace $kserve_namespace
    fi

    # Port forward for model registry
    kubectl port-forward -n $mr_namespace svc/$MODEL_REGISTRY_SERVICE "${mr_port}:$MODEL_REGISTRY_REST_PORT" &
    local pf_pid=$!

    wait_for_port $mr_port

    echo "Initializing data into Model Registry${use_custom_mr:+ in $mr_namespace namespace} for scenario ${scenario_num}..."

    # Create registered model and capture the ID
    local rm_response=$(curl --silent -X 'POST' \
      "$mr_hostname/api/model_registry/v1alpha3/registered_models" \
      -H 'accept: application/json' \
      -H 'Content-Type: application/json' \
      -d "$(jq -n \
        --arg name "$model_name" \
        '{
          description: "Iris scikit-learn model",
          name: $name
        }')")
    local rm_id=$(echo "$rm_response" | jq -r '.id')
    echo "Created registered model with ID: $rm_id"

    # Create model version and capture the ID
    local mv_name="${version_name:-v1}"
    local mv_response=$(curl --silent -X 'POST' \
      "$mr_hostname/api/model_registry/v1alpha3/model_versions" \
      -H 'accept: application/json' \
      -H 'Content-Type: application/json' \
      -d "$(jq -n \
        --arg name "$mv_name" \
        --arg description "Iris model version $mv_name" \
        --arg rmId "$rm_id" \
        '{
          description: $description,
          name: $name,
          registeredModelID: $rmId
        }')")
    local mv_id=$(echo "$mv_response" | jq -r '.id')
    echo "Created model version with ID: $mv_id"

    # Create model artifact
    curl --silent -X 'POST' \
      "$mr_hostname/api/model_registry/v1alpha3/model_versions/$mv_id/artifacts" \
      -H 'accept: application/json' \
      -H 'Content-Type: application/json' \
      -d "$(jq -n \
        --arg name "$artifact_name" \
        --arg description "Model artifact for Iris $mv_name" \
        '{
          description: $description,
          uri: "gs://kfserving-examples/models/sklearn/1.0/model",
          state: "UNKNOWN",
          name: $name,
          modelFormatName: "sklearn",
          modelFormatVersion: "1",
          artifactType: "model-artifact"
        }')"

    echo "Starting test for scenario ${scenario_num}..."

    # Build storage URI
    local storage_uri
    if [ "$use_custom_mr" = true ]; then
        if [ -n "$version_name" ]; then
            storage_uri="model-registry://$model_name/$version_name?namespace=$mr_namespace"
        else
            storage_uri="model-registry://$model_name?namespace=$mr_namespace"
        fi
    else
        if [ -n "$version_name" ]; then
            storage_uri="model-registry://$model_name/$version_name"
        else
            storage_uri="model-registry://$model_name"
        fi
    fi

    kubectl apply -n $kserve_namespace -f - <<EOF
apiVersion: "serving.kserve.io/v1beta1"
kind: "InferenceService"
metadata:
  name: "$inference_service_name"
spec:
  predictor:
    model:
      modelFormat:
        name: sklearn
      storageUri: "$storage_uri"
EOF

    # Give InferenceService a moment to be processed
    sleep 2

    # Debug: Check InferenceService status
    echo "Checking InferenceService status for scenario ${scenario_num}..."
    kubectl get inferenceservice -n $kserve_namespace || true
    kubectl describe inferenceservice $inference_service_name -n $kserve_namespace || true
    echo "Checking for events in namespace..."
    kubectl get events -n $kserve_namespace --sort-by='.lastTimestamp' || true
    echo "Checking all resources in namespace..."
    kubectl get all -n $kserve_namespace || true

    # wait for pod predictor to be initialized
    repeat_cmd_until "kubectl get pod -n $kserve_namespace --selector='component=predictor' --output jsonpath='{.items[*].metadata.name}' | grep ${inference_service_name}-predictor | wc -l" 60 "" "-gt" "0"

    # If we failed to find the pod, show more debug info
    if [ $? -ne 0 ]; then
        echo "Failed to find predictor pod for scenario ${scenario_num}. Showing debug info..."
        kubectl get pods -n $kserve_namespace -o wide || true
        kubectl get inferenceservice $inference_service_name -n $kserve_namespace -o yaml || true
    fi

    local predictor=$(kubectl get pod -n $kserve_namespace --selector="component=predictor" --output jsonpath='{.items[0].metadata.name}')

    kubectl wait --for=condition=Ready pod/$predictor -n $kserve_namespace --timeout=5m

    kubectl wait --for=jsonpath='{.status.url}' inferenceservice/$inference_service_name -n $kserve_namespace --timeout=5m
    sleep 5

    local service_hostname=$(kubectl get inferenceservice $inference_service_name -n $kserve_namespace -o jsonpath='{.status.url}' | cut -d "/" -f 3)
    local result=$(curl -s -H "Host: ${service_hostname}" -H "Content-Type: application/json" "http://${INGRESS_HOST}/v1/models/${inference_service_name}:predict" -d @/tmp/iris-input.json)
    echo "Scenario ${scenario_num} received: $result"

    if [ ! "$result" = "{\"predictions\":[1,1]}" ]; then
        echo "Scenario ${scenario_num} - Prediction does not match expectation!"
        echo "Printing some logs for debugging.."
        kubectl logs pod/$predictor -n $kserve_namespace -c storage-initializer || true
        kubectl logs pod/$predictor -n $kserve_namespace -c kserve-container || true
        kill $pf_pid 2>/dev/null || true
        return 1
    else
        echo "Scenario ${scenario_num} - Test succeeded!"
        kubectl logs pod/$predictor -n $kserve_namespace -c storage-initializer || true
        kubectl logs pod/$predictor -n $kserve_namespace -c kserve-container || true
    fi

    echo "Cleaning up scenario ${scenario_num}..."
    # Delete without waiting - namespace isolation means we don't need to wait
    kubectl delete inferenceservice $inference_service_name -n $kserve_namespace --wait=false

    # Delete the namespace to clean up all resources
    kubectl delete namespace $kserve_namespace --wait=false

    kill $pf_pid 2>/dev/null || true

    echo "======== Finished Scenario ${scenario_num} ========"
    return 0
}

# Wrapper functions for each scenario
run_scenario_1() {
    run_scenario 1 8080 "$NAMESPACE" "iris-scenario-1" "v1" "sklearn-iris-scenario-one" "sklearn-iris-v1-scenario-1" "Testing with default model registry service"
}

run_scenario_2() {
    run_scenario 2 8090 "$NAMESPACE" "iris-scenario-2" "" "sklearn-iris-scenario-two" "sklearn-iris-v1-scenario-2" "Testing with default model registry service without model version"
}

run_scenario_3() {
    run_scenario 3 8082 "$TESTNAMESPACE" "iris-test-scenario-3" "v1-test" "sklearn-iris-scenario-three" "sklearn-iris-test-v1-scenario-3" "Testing with custom model registry service"
}

run_scenario_4() {
    run_scenario 4 8092 "$TESTNAMESPACE" "iris-test-scenario-4" "" "sklearn-iris-scenario-four" "sklearn-iris-test-v1-scenario-4" "Testing with custom model registry service without model version"
}

# Launch all scenarios in parallel
echo "======== Launching all scenarios in parallel ========"

run_scenario_1 &
scenario_pids[1]=$!

run_scenario_2 &
scenario_pids[2]=$!

run_scenario_3 &
scenario_pids[3]=$!

run_scenario_4 &
scenario_pids[4]=$!

# Wait for all scenarios and collect results
echo "======== Waiting for all scenarios to complete ========"

all_passed=true
for i in 1 2 3 4; do
    if wait ${scenario_pids[$i]}; then
        echo "✅ Scenario $i passed"
        scenario_results[$i]=0
    else
        echo "❌ Scenario $i failed"
        scenario_results[$i]=1
        all_passed=false
    fi
done

# Cleanup ingress port-forward
kill $ingress_pf_pid 2>/dev/null || true

# Print summary
echo "========================================"
echo "Test Summary:"
echo "========================================"
for i in 1 2 3 4; do
    if [ ${scenario_results[$i]} -eq 0 ]; then
        echo "Scenario $i: ✅ PASSED"
    else
        echo "Scenario $i: ❌ FAILED"
    fi
done
echo "========================================"

# Wait for all test namespaces to be fully deleted
echo ""
echo "Waiting for namespace cleanup to complete..."
cleanup_timeout=300  # 5 minutes timeout
cleanup_interval=2
cleanup_start=$(date +%s)
cleanup_failed=false

while true; do
    current_time=$(date +%s)
    elapsed=$((current_time - cleanup_start))

    if [ $elapsed -gt $cleanup_timeout ]; then
        echo "❌ ERROR: Namespace cleanup timed out after ${cleanup_timeout} seconds!"
        cleanup_failed=true
        break
    fi

    # Check if all namespaces are deleted
    all_deleted=true
    for i in 1 2 3 4; do
        ns="kserve-test-$i"
        if kubectl get namespace "$ns" &>/dev/null; then
            all_deleted=false
            break
        fi
    done

    if [ "$all_deleted" = true ]; then
        echo "✅ All test namespaces successfully deleted (took ${elapsed}s)"
        break
    fi

    # Show progress every 10 seconds
    if [ $((elapsed % 10)) -eq 0 ] && [ $elapsed -gt 0 ]; then
        remaining_list=""
        remaining_count=0
        for i in 1 2 3 4; do
            if kubectl get namespace "kserve-test-$i" &>/dev/null; then
                remaining_list="${remaining_list}kserve-test-$i "
                remaining_count=$((remaining_count + 1))
            fi
        done
        if [ $remaining_count -gt 0 ]; then
            echo "⏳ Waiting for cleanup... (${elapsed}s elapsed, ${remaining_count} remaining: ${remaining_list})"
        fi
    fi

    sleep $cleanup_interval
done

# Final verification - show status of each namespace
echo ""
echo "Final namespace status:"
for i in 1 2 3 4; do
    ns="kserve-test-$i"
    if kubectl get namespace "$ns" &>/dev/null; then
        status=$(kubectl get namespace "$ns" -o jsonpath='{.status.phase}' 2>/dev/null || echo "Unknown")
        echo "  Namespace $ns: ❌ Still exists (status: $status)"
        cleanup_failed=true
    else
        echo "  Namespace $ns: ✅ Deleted"
    fi
done

echo "========================================"

# Fail the test if cleanup failed
if [ "$cleanup_failed" = true ]; then
    echo "❌ Namespace cleanup failed!"
    exit 1
fi

if [ "$all_passed" = true ]; then
    echo "🎉 All tests passed!"
    exit 0
else
    echo "❌ Some tests failed!"
    exit 1
fi
