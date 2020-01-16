# This Dockerfile builds invoicer twice: once on Alpine, once on Debian.
# If the binaries are the same, one is compressed, and copied to the `final` stage.

# invoicer version to be build
ARG VERSION=v0.7.5

# Target CPU archtecture of built Invoicer binary
ARG ARCH

# Define default versions so that they don't have to be repreated throughout the file
ARG VER_GO=1.13
ARG VER_ALPINE=3.11

ARG USER=invoicer
ARG DIR=/data/

# NOTE: You should only override this if you know what you're doing
ARG TAGS="osusergo,netgo,static_build"


#
## This stage builds `invoicer` in an Alpine environment
#
FROM golang:${VER_GO}-alpine${VER_ALPINE} AS alpine-builder

# Provided by Docker by default
ARG TARGETVARIANT

# These two should only be set for cross-compilation
ARG GOARCH
ARG GOARM

# Capture ARGs defined globally
ARG VERSION
ARG TAGS

# Only set GOOS if GOARCH is set
ENV GOOS ${GOARCH:+linux}

# If GOARM is not set, but TARGETVARIANT is set - hardcode GOARM to 6
ENV GOARM ${GOARM:-${TARGETVARIANT:+6}}
ENV GCO_ENABLED 0
ENV LDFLAGS "-s -w -buildid= -X main.version=${VERSION}"
ENV BINARY /go/bin/invoicer

RUN apk add --no-cache  musl-dev  file  git  gcc

RUN mkdir -p /go/src/

COPY ./ /go/src/

WORKDIR /go/src/

## Print versions of software used for this build
#   NOTE: sha256sum is part of busybox on Alpine
RUN busybox | head -n 1
RUN file --version
RUN git --version
RUN gcc --version
RUN uname -a
RUN env && go version && go env

## Build invoicer binary.
#   The first line gets hash of the last git-commit & second one prints it.
#   And here's all other flags explained:
#       `-x` [verbocity++] print all executed commands
#       `-v` [verbocity++] print names of compiled packages
#       `-mod=readonly` [reproducibility] do not change versions of used packages no matter what
#       `-trimpath` [reproducibility] make sure absolute paths are not included anywhere in the binary
#       `-tags` [reproducibility] tell Go to build a static binary, see more: https://github.com/golang/go/issues/26492
#       `-ldflags`
#           `-s` [size--] do not include symbol table and debug info
#           `-w` [size--] do not include DWARF symbol table
#           `-buildid` [reproducibility] while this should always be the same in our setup, clear it just-in-case
#           `-X` [info] is used twice to inject version, and git-hash into the built binary
#
#   NOTE: all of this has to happen in a single `RUN`, because it's impossible to set ENV var in Docker to
#       an output of an expression.
RUN export GIT_HASH="$(git rev-parse HEAD)"; \
    echo "Building git tag: ${GIT_HASH}"; \
    go build  -x  -v  -trimpath  -mod=readonly  -tags="${TAGS}" \
        -ldflags="${LDFLAGS} -X main.gitHash=${GIT_HASH}" \
        -o "${BINARY}"

# Print rudimentary info about the built binary
RUN sha256sum   "${BINARY}"
RUN file -b     "${BINARY}"
RUN du          "${BINARY}"


#
## This stage builds `invoicer` in a Debian environment
#
# NOTE: Comments that would be identical to Alpine stage skipped for brevity
FROM golang:${VER_GO}-buster AS debian-builder

ARG TARGETVARIANT
ARG GOARCH
ARG GOARM
ARG VERSION
ARG TAGS

ENV GOOS ${GOARCH:+linux}
ENV GOARM ${GOARM:-${TARGETVARIANT:+6}}
ENV GCO_ENABLED 0
ENV LDFLAGS "-s -w -buildid= -X main.version=${VERSION}"
ENV BINARY /go/bin/invoicer

RUN apt-get update \
    && apt-get -y install  file  git

RUN mkdir -p /go/src/

COPY ./ /go/src/
WORKDIR /go/src/

RUN sha256sum --version
RUN make --version
RUN file --version
RUN git --version
RUN uname -a
RUN env && go version && go env

RUN export GIT_HASH="$(git rev-parse HEAD)"; \
    echo "Building git hash: ${GIT_HASH}"; \
    go build  -x  -v  -trimpath  -mod=readonly  -tags="${TAGS}" \
        -ldflags="${LDFLAGS} -X main.gitHash=${GIT_HASH}" \
        -o "${BINARY}"

RUN sha256sum   "${BINARY}"
RUN file -b     "${BINARY}"
RUN du          "${BINARY}"



#
## This stage compares previously built binaries, and only proceeds if they are identical
#
FROM alpine:${VER_ALPINE} AS cross-check

# Install utilities used later
RUN apk add --no-cache  file  upx

# Prepare destination directories for previously built binaries
RUN mkdir -p  /bin  /alpine  /debian

# Copy binaries from prior builds
COPY  --from=alpine-builder /go/bin/invoicer  /alpine/
COPY  --from=debian-builder /go/bin/invoicer  /debian/

# Print binary info PRIOR comparison & compression
RUN sha256sum   /debian/invoicer  /alpine/invoicer
RUN file        /debian/invoicer  /alpine/invoicer
RUN du          /debian/invoicer  /alpine/invoicer

# Compare built binaries
RUN diff -q  /alpine/invoicer  /debian/invoicer

# If identical, proceed to move one binary into `/bin/`
RUN mv /alpine/invoicer /bin/

# Compress, and be verbose about it
RUN upx -v /bin/invoicer

# Print binary info PAST compression
RUN sha256sum /bin/invoicer
RUN file -b   /bin/invoicer
RUN du        /bin/invoicer



#
## This stage is used to generate /etc/{group,passwd,shadow} files & avoid RUN-ing commands in the `final` layer,
#   which would break cross-compiled images.
#
FROM alpine:${VER_ALPINE} AS perms

ARG USER
ARG DIR

# NOTE: Default GID == UID == 1000
RUN adduser --disabled-password \
            --home ${DIR} \
            --gecos "" \
            ${USER}



#
## This is the final image that gets shipped to Docker Hub
#
# NOTE: `${ARCH:+${ARCH}/}` - if ARCH is set, append `/` to it, leave it empty otherwise
FROM ${ARCH:+${ARCH}/}alpine:${VER_ALPINE} AS final

ARG USER
ARG DIR

# Hai 👋
LABEL maintainer="Damian Mee (@meeDamian)"

# Copy only the relevant parts from the `perms` image
COPY  --from=perms  /etc/group   /etc/
COPY  --from=perms  /etc/passwd  /etc/
COPY  --from=perms  /etc/shadow  /etc/

# Copy the binary from the cross-check stage
COPY  --from=cross-check  /bin/invoicer  /bin/

# Expose the volume to communicate config, log, etc through (default: /data/)
VOLUME ${DIR}

# Expose port Invoicer listens on
EXPOSE 8080

USER ${USER}
WORKDIR ${DIR}

# Specify the start command and entrypoint as the invoicer daemon.
ENTRYPOINT ["invoicer"]

CMD ["-config", "/data/invoicer.conf"]
