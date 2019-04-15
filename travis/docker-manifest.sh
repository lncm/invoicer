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
IMAGE_AMD64="${IMAGE_VERSIONED}-linux-amd64"
IMAGE_ARM="${IMAGE_VERSIONED}-linux-arm"


docker pull "${IMAGE_AMD64}"
docker pull "${IMAGE_ARM}"


echo     "Pushing manifest ${IMAGE_VERSIONED}"
docker -D manifest create "${IMAGE_VERSIONED}"  "${IMAGE_AMD64}"  "${IMAGE_ARM}"
docker manifest annotate  "${IMAGE_VERSIONED}"  "${IMAGE_ARM}"  --os linux --arch arm --variant v6
docker manifest push      "${IMAGE_VERSIONED}"


IMAGE_LATEST="${TRAVIS_REPO_SLUG}:latest"

echo     "Pushing manifest ${IMAGE_LATEST}"
docker -D manifest create "${IMAGE_LATEST}"  "${IMAGE_AMD64}"  "${IMAGE_ARM}"
docker manifest annotate  "${IMAGE_LATEST}"  "${IMAGE_ARM}"  --os linux --arch arm --variant v6
docker manifest push      "${IMAGE_LATEST}"
