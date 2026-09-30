#!/usr/bin/env bash

set -euo pipefail

archive="$1"
source_image="public.ecr.aws/docker/library/busybox:latest"
local_image="localhost:5001/busybox:latest"

curl --fail --silent --show-error --retry 10 --retry-connrefused --retry-delay 1 \
    http://localhost:5001/v2/ > /dev/null
docker load -i "$archive"
docker tag "$source_image" "$local_image"
docker push "$local_image"
curl --fail --silent --show-error \
    --header 'Accept: application/vnd.docker.distribution.manifest.v2+json, application/vnd.docker.distribution.manifest.list.v2+json, application/vnd.oci.image.manifest.v1+json, application/vnd.oci.image.index.v1+json' \
    http://localhost:5001/v2/busybox/manifests/latest > /dev/null
