set -e
set -o xtrace

############################################################
# Help                                                     #
############################################################
Help() {
  # Display Help
  echo "ModelRegistry install script."
  echo
  echo "Syntax: [-n NAMESPACE] [-i IMAGE]"
  echo "options:"
  echo "  n Namespace."
  echo "  i Model registry image."
  echo
}

MR_ROOT="$(dirname "$0")/../.."

namespace=kubeflow
image=ghcr.io/kubeflow/hub/server:latest
while getopts ":hn:i:" option; do
   case $option in
      h) # display Help
         Help
         exit;;
      n) # override namespace
          namespace=$OPTARG;;
      i) # override model registry image
          image=$OPTARG;;
     \?) # Invalid option
         echo "Error: Invalid option"
         exit;;
   esac
done

# Create namespace if not already existing
if ! kubectl get namespace "$namespace" &> /dev/null; then
   kubectl create namespace $namespace
fi
# Apply model-registry kustomize manifests
echo Using model registry image: $image
cd $MR_ROOT/manifests/kustomize/base && kustomize edit set image ghcr.io/kubeflow/hub/server:latest=${image} && \
kustomize edit set namespace $namespace && cd -
cd $MR_ROOT/manifests/kustomize/overlays/db && kustomize edit set namespace $namespace && cd -
kubectl -n $namespace apply -k "$MR_ROOT/manifests/kustomize/overlays/db"

# Wait for model registry deployment
kubectl wait --for=condition=Available deployment/model-registry-deployment -n "$namespace" --timeout=15m
modelregistry=$(kubectl get pod -n $namespace --selector="component=model-registry-server" --output jsonpath='{.items[0].metadata.name}')
kubectl wait --for=condition=Ready pod/$modelregistry -n $namespace --timeout=15m
