#!/usr/bin/env bash

set -euo pipefail

image="$1"
archive="$2"
cluster_name="${3:-}"

if [[ ! -f "$archive" ]]; then
    platform="$(docker version --format '{{.Server.Os}}/{{.Server.Arch}}')"
    docker pull --platform "$platform" "$image"
    # Kind imports all platforms referenced by the archive. Export only the
    # local platform so the archive does not reference unavailable manifests.
    docker save --platform "$platform" -o "$archive" "$image"
fi

if [[ -n "$cluster_name" ]]; then
    kind load image-archive "$archive" -n "$cluster_name"
fi
