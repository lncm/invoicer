#!/bin/bash
set -e

# make sure Docker's config folder exists
mkdir -p ~/.docker

# Putting experimental:true to config enables manifest options
echo '{ "experimental": "enabled" }' > ~/.docker/config.json

# put above config into effect
sudo systemctl restart docker

echo "${DOCKER_PASS}" | docker login -u="${DOCKER_USER}" --password-stdin

# print this to verify manifest options are now available
docker version


IMAGE_VERSIONED="${TRAVIS_REPO_SLUG}:${TRAVIS_TAG}"
IMAGE_VER_AMD64="${IMAGE_VERSIONED}-linux-amd64"
IMAGE_VER_ARM="${IMAGE_VERSIONED}-linux-arm"


docker pull "${IMAGE_VER_AMD64}"
docker pull "${IMAGE_VER_ARM}"


echo     "Pushing manifest ${IMAGE_VERSIONED}"
docker -D manifest create "${IMAGE_VERSIONED}"  "${IMAGE_VER_AMD64}"  "${IMAGE_VER_ARM}"
docker manifest annotate  "${IMAGE_VERSIONED}"  "${IMAGE_VER_ARM}"  --os linux --arch arm --variant v6
docker manifest push      "${IMAGE_VERSIONED}"


IMAGE_LATEST="${TRAVIS_REPO_SLUG}:latest"
IMAGE_LATEST_AMD64="${IMAGE_LATEST}-linux-amd64"
IMAGE_LATEST_ARM="${IMAGE_LATEST}-linux-arm"

echo     "Pushing manifest ${IMAGE_LATEST}"
docker -D manifest create "${IMAGE_LATEST}"  "${IMAGE_LATEST_AMD64}"  "${IMAGE_LATEST_ARM}"
docker manifest annotate  "${IMAGE_LATEST}"  "${IMAGE_LATEST_ARM}"  --os linux --arch arm --variant v6
docker manifest push "${IMAGE_LATEST}"
