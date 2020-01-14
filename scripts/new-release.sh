#!/usr/bin/env bash

set -eo pipefail

#
## Given version (and optional variant), this script creates & pushes a relevant git-tag.
#

# required version
VERSION=$1

# Verify version to-be-released is provided
if [[ -z "${VERSION}" ]]; then
  >&2 printf "\nERR: version missing:  version needs to be passed as the first argument.  Try:\n"
  >&2 printf "\t./%s  %s\n\n"   "$(basename "$0")"  "v0.7.2"
  exit 1
fi

# Verify there's no uncommitted changes in the working dir
if [[ -n "$(git status --untracked-files=no --porcelain)" ]]; then
  >&2 printf "\nERR: working directory not clean.  Commit, or stash changes to continue.\n\n"
  exit 1
fi

if ! grep -q "${VERSION}" "./Dockerfile" ; then
  >&2 printf "\nERR: Requested version not present in Dockerfile. Make sure that's what you want to do.\n\n"
  exit 1
fi

git fetch --tags

# Get last build number
LAST=$(git tag | grep '+build' | sed 's|^.*build||' | sort -h | tail -n 1)

# Increment it
((LAST++))

TAG="${VERSION}+build${LAST}"

printf "Creating tag: %s…\t" "${TAG}"

git tag -sa "${TAG}" -m "${TAG}"

echo "done"


printf "Pushing tag: %s…\t" "${TAG}"

git push origin "${TAG}"

echo "All done"
