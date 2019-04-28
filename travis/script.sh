#!/bin/bash
set -e

# Build image for specified architecture, if specified
if [[ ! -z "${ARCH}" ]]; then
    docker build --no-cache -t invoicer --build-arg "arch=${ARCH}" .

    # Push image, if tag was specified
    if [[ -n "${TRAVIS_TAG}" ]]; then
        echo "${DOCKER_PASS}" | docker login -u="${DOCKER_USER}" --password-stdin

        docker tag invoicer "${TRAVIS_REPO_SLUG}:${TRAVIS_TAG}-${ARCH}"
        docker push "${TRAVIS_REPO_SLUG}:${TRAVIS_TAG}-${ARCH}"
    fi

    exit 0
fi

# This happens when no ARCH was provided.  Specifically, in the deploy job.
go version

make ci
