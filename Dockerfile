FROM golang:alpine as builder

# Install dependencies (LND) and build/download the binaries.
RUN apk add --no-cache --update alpine-sdk \
    make

RUN mkdir -p /src/
COPY ./ /src/
WORKDIR /src/

RUN make bin/invoicer

RUN apk add --no-cache --update wget

# Start a new, final image.
FROM alpine as final


# Create directory for data assets
RUN mkdir -p /static/

# Copy the binaries from the builder image.
COPY --from=builder /src/static/ /static/
COPY --from=builder /src/invoicer /bin/

COPY entrypoint-invoicer.sh /bin/

RUN chmod 755 /bin/entrypoint-invoicer.sh

# Expose lnd ports (p2p, rpc).
EXPOSE 1666

# Invoicer Entrypoint
ENTRYPOINT entrypoint-invoicer.sh

