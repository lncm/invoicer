# This Dockerfile builds invoicer twice: once on Alpine, once on Debian.  If the binaries are the same,
#   one is compressed and moved to the `final` stage.

# invoicer version to be build
ARG VERSION=v0.7.3

# Target CPU archtecture of built IPFS binary
ARG ARCH=amd64

# Define default versions so that they don't have to be repreated throughout the file
ARG VER_GO=1.13
ARG VER_ALPINE=3.11

ARG USER=invoicer
ARG DIR=/data/


#
## The pairs of Docker stages below define Go Environment necessary for cross-compilation on
#   two different base images: Alpine, and Debian.  Later build stages can be started as:
#
#   `FROM ${ARCH}-debian`  or
#   `FROM ${ARCH}-alpine`
#
## Stage defining Alpine environment
FROM golang:${VER_GO}-alpine${VER_ALPINE} AS alpine-base
ENV GOOS=linux  GCO_ENABLED=0

## Stage defining Debian environment
FROM golang:${VER_GO}-buster AS debian-base
ENV GOOS=linux  GCO_ENABLED=0


FROM alpine-base AS amd64-alpine
ENV GOARCH=amd64

FROM debian-base AS amd64-debian
ENV GOARCH=amd64


FROM alpine-base AS arm64v8-alpine
ENV GOARCH=arm64

FROM debian-base AS arm64v8-debian
ENV GOARCH=arm64


FROM alpine-base AS arm32v7-alpine
ENV GOARCH=arm  GOARM=7

FROM debian-base AS arm32v7-debian
ENV GOARCH=arm  GOARM=7


FROM alpine-base AS arm32v6-alpine
ENV GOARCH=arm  GOARM=6

FROM debian-base AS arm32v6-debian
ENV GOARCH=arm  GOARM=6



# This stage builds invoicer in an Alpine environment
FROM ${ARCH}-alpine AS alpine-builder

ARG ARCH
ARG VERSION

RUN apk add --no-cache  musl-dev  file  git  gcc

RUN mkdir -p /go/src/

COPY ./ /go/src/
WORKDIR /go/src/

# Print versions of software used for this build
# NOTE: sha256sum is part of busybox on Alpine
RUN busybox | head -n 1
RUN file --version
RUN git --version
RUN gcc --version
RUN uname -a
RUN env && go version && go env

# All `-tags`, ` -buildid=`, and `-trimpath` added to make the output binary reproducible
##   ctx: https://github.com/golang/go/issues/26492
RUN export LD="-s -w -buildid= -X main.version=${VERSION} -X main.gitHash=$(git rev-parse HEAD)"; \
    echo "Building with ldflags: '${LD}'" && \
    go build -v  -trimpath  -mod=readonly \
        -ldflags="${LD}"  -tags="osusergo netgo static_build" \
        -o /go/bin/

# Print rudimentary info about the built binary
RUN sha256sum   /go/bin/invoicer
RUN file -b     /go/bin/invoicer
RUN du          /go/bin/invoicer



# This stage builds invoicer in a Debian environment
# NOTE: Comments that would be identical to Alpine stage skipped for brevity
FROM ${ARCH}-debian AS debian-builder

ARG ARCH
ARG VERSION

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

RUN export LD="-s -w -buildid= -X main.version=${VERSION} -X main.gitHash=$(git rev-parse HEAD)"; \
    echo "Building with ldflags: '${LD}'" && \
    go build -v  -trimpath  -mod=readonly \
        -ldflags="${LD}"  -tags="osusergo netgo static_build"\
        -o /go/bin/

RUN sha256sum   /go/bin/invoicer
RUN file -b     /go/bin/invoicer
RUN du          /go/bin/invoicer



# This stage compares previously built binaries, and only proceeds if they are identical
FROM alpine:${VER_ALPINE} AS cross-check

# Install utilities used later
RUN apk add --no-cache  file  upx

# Prepare destination directories for previously built binaries
RUN mkdir -p  /bin  /alpine  /debian

# Copy binaries from prior builds
COPY  --from=alpine-builder  /go/bin/invoicer  /alpine/
COPY  --from=debian-builder  /go/bin/invoicer  /debian/

# Print binary info PRIOR comparison & compression
RUN sha256sum   /debian/invoicer  /alpine/invoicer
RUN file        /debian/invoicer  /alpine/invoicer
RUN du          /debian/invoicer  /alpine/invoicer

# Compare built binaries
RUN diff -q  /alpine/invoicer  /debian/invoicer

# If identical, proceed to move one binary into /bin/
RUN mv /alpine/invoicer /bin/

# Compress, and be verbose about it
RUN upx -v /bin/invoicer

# Print binary info PAST compression
RUN sha256sum /bin/invoicer
RUN file -b   /bin/invoicer
RUN du        /bin/invoicer


# This stage is used to generate /etc/{group,passwd,shadow} files & avoid RUN-ing commands in the `final` layer,
#   which would break cross-compiled images.
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
FROM ${ARCH}/alpine:${VER_ALPINE} AS final

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
