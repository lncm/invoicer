#!/usr/bin/env bash

set -eo pipefail

SLUG=$1
VERSION=$2
SHORT_VERSION=${3:-$VERSION}

BASE="$SLUG:$VERSION"

IMAGE_AMD64="$BASE-amd64"
IMAGE_ARM64="$BASE-arm64"
IMAGE_ARM6="$BASE-arm32v6"
IMAGE_ARM7="$BASE-arm32v7"


MANIFEST="$SLUG:$SHORT_VERSION"

docker -D manifest create "$MANIFEST"  "$IMAGE_AMD64"  "$IMAGE_ARM64"  "$IMAGE_ARM6"  "$IMAGE_ARM7"
docker manifest annotate  "$MANIFEST"  "$IMAGE_ARM64"  --os linux  --arch arm64 --variant v8
docker manifest annotate  "$MANIFEST"  "$IMAGE_ARM7"   --os linux  --arch arm   --variant v7
docker manifest annotate  "$MANIFEST"  "$IMAGE_ARM6"   --os linux  --arch arm   --variant v6
docker manifest push      "$MANIFEST"
