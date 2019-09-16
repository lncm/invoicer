#!/bin/bash
set -e

SLUG=$1
VERSION=$2

SHORT_VERSION=$3
if [[ -z "${SHORT_VERSION}" ]]; then
  SHORT_VERSION="${VERSION}"
fi

BASE="${SLUG}:${VERSION}"

IMAGE_AMD64="${BASE}-linux-amd64"
IMAGE_ARM64="${BASE}-linux-arm64"
IMAGE_ARM6="${BASE}-linux-arm32v6"
IMAGE_ARM7="${BASE}-linux-arm32v7"


MANIFEST="${SLUG}:${SHORT_VERSION}"

docker -D manifest create "${MANIFEST}"  "${IMAGE_AMD64}"  "${IMAGE_ARM64}"  "${IMAGE_ARM6}"  "${IMAGE_ARM7}"
docker manifest annotate  "${MANIFEST}"  "${IMAGE_ARM64}"  --os linux  --arch arm64
docker manifest annotate  "${MANIFEST}"  "${IMAGE_ARM7}"   --os linux  --arch arm   --variant v7
docker manifest annotate  "${MANIFEST}"  "${IMAGE_ARM6}"   --os linux  --arch arm   --variant v6
docker manifest push      "${MANIFEST}"
