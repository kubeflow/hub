#!/usr/bin/env bash

set -euo pipefail

image="$1"
archive="$2"
cluster_name="${3:-}"

if [[ ! -f "$archive" ]]; then
    docker pull "$image"
    docker save -o "$archive" "$image"
fi

if [[ -n "$cluster_name" ]]; then
    kind load image-archive "$archive" -n "$cluster_name"
fi
