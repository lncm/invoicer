# This Dockerfile builds invoicer twice for any given platform (<os>-<architecture> combo, ex: `linux-amd64`) - once on
#   Alpine and once on Debian.  After the build completes both binaries are compared.  If identical, the result
#   binary is stripped, and moved to a final stage that's ready to be uploaded to Docker Hub or Github Registry.

# This stage builds invoicer in an Alpine environment
FROM golang:1.13-alpine3.10 AS alpine-builder

RUN apk add --no-cache \
        git \
        make \
        gcc \
        musl-dev \
        file

RUN mkdir -p /src/
COPY ./ /src/
WORKDIR /src/

# Print versions of software used for this build
RUN uname -a
RUN go version
RUN git --version
RUN make --version
RUN gcc --version
RUN file --version
# NOTE: sha256sum is part of busybox on Alpine
RUN busybox | head -n 1

# Passed to `docker build` using ex: `--build-arg goarch=arm64`
ARG goarch=amd64

# See Makefile.  This builds invoicer for the requested `${goarch}`
RUN make bin/linux-${goarch}/invoicer

# Move built binary to a common directory, so that `${goarch}` no longer needs to be referenced.
RUN mkdir -p /bin \
    && mv  bin/linux-${goarch}/invoicer  /bin/

# Print rudimentary info about the built binary
RUN du /bin/invoicer \
    && file -b /bin/invoicer \
    && sha256sum /bin/invoicer


# This stage builds invoicer in a Debian environment
# NOTE: Comments that would be identical to Alpine stage skipped for brevity
FROM golang:1.13-buster AS debian-builder

RUN apt-get update && \
    apt-get -y install \
        git \
        make \
        file

RUN mkdir -p /src/
COPY ./ /src/
WORKDIR /src/

RUN uname -a
RUN go version
RUN git --version
RUN make --version
RUN file --version
RUN sha256sum --version

ARG goarch=amd64

RUN make bin/linux-${goarch}/invoicer

RUN mkdir -p /bin \
    && mv  bin/linux-${goarch}/invoicer  /bin/

RUN du /bin/invoicer \
    && file -b /bin/invoicer \
    && sha256sum /bin/invoicer


# This stage compares all previously built binaries, and if they are identical, strips the binary
FROM alpine:3.10 AS cross-check

# Install utilities used later
RUN apk add --no-cache \
    upx \
    file

# Prepare destination directories for previously built binaries
RUN mkdir -p  /bin  /alpine  /debian

# Copy binaries from all builds
COPY  --from=alpine-builder  /bin/invoicer  /alpine/
COPY  --from=debian-builder  /bin/invoicer  /debian/

# Compare both built binaries
RUN diff -q  /alpine/invoicer  /debian/invoicer

# If all are identical, proceed to move the binary into
RUN mv /alpine/invoicer /bin/

# Print binary info PRIOR compression
RUN du /bin/invoicer \
    && file -b /bin/invoicer \
    && sha256sum /bin/invoicer

# Compress binary, and be verbose about it
RUN upx -v /bin/invoicer

# Print binary info PAST compression
RUN du /bin/invoicer \
    && file -b /bin/invoicer \
    && sha256sum /bin/invoicer


# This is a final stage, destined to be distributed, if successful
FROM alpine:3.10 AS final

# Hai 👋
LABEL maintainer="Damian Mee (@meeDamian)"

# Copy the binaries from the builder image.
COPY  --from=cross-check  /bin/invoicer  /bin/

# Expose the volume to communicate config, log, etc through
VOLUME /root/.lncm

# Expose Invoicer port
EXPOSE 8080

# Specify the start command and entrypoint as the invoicer daemon.
ENTRYPOINT ["invoicer"]
CMD ["invoicer"]

