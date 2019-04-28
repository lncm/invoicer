#!/bin/bash
set -e

# Update Docker Image if base architecture of the final stage needs changing
if [[ "${ARCH}" = "linux-armv6" ]] || [[ "${ARCH}" = "linux-armv7" ]]; then
    BASE_ARCH="arm32v"${ARCH: -1}""

    sed -ie "s/FROM alpine/FROM ${BASE_ARCH}\/alpine/g" Dockerfile
    echo "Dockerfile modified: Final stage image, base CPU architecture changed to: ${BASE_ARCH}"
fi
