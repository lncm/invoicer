# This Dockerfile builds invoicer twice for any given platform (<os>-<architecture> combo, ex: `linux-amd64`) - once on
#   Alpine and once on Debian.  After the build completes both binaries are compared.  If identical, the result
#   binary is stripped, and moved to a final stage that's ready to be uploaded to Docker Hub or Github Registry.

# This stage builds invoicer in an Alpine environment
FROM golang:1.13-alpine3.11 AS alpine-builder

RUN apk add --no-cache \
    musl-dev \
    make \
    file \
    git \
    gcc

RUN mkdir -p /src/
COPY ./ /src/
WORKDIR /src/

# Print versions of software used for this build
# NOTE: sha256sum is part of busybox on Alpine
RUN busybox | head -n 1
RUN make --version
RUN file --version
RUN git --version
RUN gcc --version
RUN go version
RUN uname -a

# Passed to `docker build` using ex: `--build-arg goarch=arm64`
ARG goarch=amd64

# See Makefile.  This builds invoicer for the requested `${goarch}`
RUN make bin/linux-${goarch}/invoicer

# Move built binary to a common directory, so that `${goarch}` no longer needs to be referenced.
RUN mkdir -p /bin \
    && mv  bin/linux-${goarch}/invoicer  /bin/

# Print rudimentary info about the built binary
RUN sha256sum   /bin/invoicer
RUN file -b     /bin/invoicer
RUN du          /bin/invoicer


# This stage builds invoicer in a Debian environment
# NOTE: Comments that would be identical to Alpine stage skipped for brevity
FROM golang:1.13-buster AS debian-builder

RUN apt-get update \
    && apt-get -y install \
        make \
        file \
        git

RUN mkdir -p /src/
COPY ./ /src/
WORKDIR /src/

RUN sha256sum --version
RUN make --version
RUN file --version
RUN git --version
RUN go version
RUN uname -a

ARG goarch=amd64

RUN make bin/linux-${goarch}/invoicer

RUN mkdir -p /bin \
    && mv  bin/linux-${goarch}/invoicer  /bin/

RUN sha256sum   /bin/invoicer
RUN file -b     /bin/invoicer
RUN du          /bin/invoicer


# This stage compares previously built binaries, and only proceeds if they are
FROM alpine:3.11 AS cross-check

# Install utilities used later
RUN apk add --no-cache \
    file \
    upx

# Prepare destination directories for previously built binaries
RUN mkdir -p  /bin  /alpine  /debian

# Copy binaries from prior builds
COPY  --from=alpine-builder  /bin/invoicer  /alpine/
COPY  --from=debian-builder  /bin/invoicer  /debian/

# Compare built binaries
RUN diff -q  /alpine/invoicer  /debian/invoicer

# If identical, proceed to move one binary into /bin/
RUN mv /alpine/invoicer /bin/

# Print binary info PRIOR compression
RUN sha256sum   /bin/invoicer
RUN file -b     /bin/invoicer
RUN du          /bin/invoicer

# Compress, and be verbose about it
RUN upx -v /bin/invoicer

# Print binary info PAST compression
RUN sha256sum /bin/invoicer
RUN file -b   /bin/invoicer
RUN du        /bin/invoicer


# This stage is used to generate /etc/{group,passwd,shadow} files & avoid RUN-ing commands in the `final` layer,
#   which would break cross-compiled images.
FROM alpine:3.11 AS perms

ARG USER=invoicer
ARG DIR=/data/

# NOTE: Default GID == UID == 1000
RUN adduser --disabled-password \
            --home ${DIR} \
            --gecos "" \
            ${USER}


# This is a final stage, destined to be distributed, if successful
FROM alpine:3.11 AS final

ARG USER=invoicer
ARG DIR=/data/

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
