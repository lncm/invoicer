#!/bin/bash
set -e

#
## This script adjusts `--platform=` of the final Dockerfile stage according to provided CPU architecture.
#

ARCH=$1

if [[ -z "${ARCH}" ]]; then
  >&2 printf "\nERR: This scripts requires architecture passed.  Try:\n"
  >&2 printf "\t./%s  %s\n\n"   "$(basename "$0")"  "arm64"
  exit 1
fi


# Convert matrix-supplied architecture to a format understood by Docker's `--platform=`  
case "${ARCH}" in
arm32v6)  CPU="arm/v6" ;;
arm32v7)  CPU="arm/v7" ;;
arm64)    CPU="arm64"  ;;
esac


# If `${CPU}` is empty here, we're done, as final image base needn't be changed
if [[ -z "${CPU}" ]]; then
  exit 0
fi

# If `gsed` is available in the system, then use it instead as the available `sed` might not be too able (MacOS)…
SED="sed"
if command -v gsed >/dev/null; then
  SED=gsed
fi

# Decyphering `sed` expressions is always "fun"…  So to make it easier on the reader here's an explanation:
# tl;dr: Replace last FROM with one that specifies target CPU architecture.
#
# This command replaces the last `FROM` statement in `Dockerfile`, with one that specifies `--platform=`,
#   ex for arm32v7:
#
#   FROM                         alpine:3.10 AS final
#     ⬇                               ⬇           ⬇
#   FROM --platform=linux/arm/v7 alpine:3.10 AS final
#
# Note:
#   `-i` - apply changes in-place (actually change the file)
#   `s/` - substitute; followed by two `/`-separated sections:
#     1st section looks for a match.  Escaped \(\) define a _capture group_
#     2nd section defines replacement.  `\1` is the value of the _capture group_ from the 1st section
${SED} -i "s|^FROM \(.*final\)$|FROM --platform=linux/$CPU \1|" Dockerfile

echo "Dockerfile modified: CPU architecture of the final stage set to: ${CPU}"
