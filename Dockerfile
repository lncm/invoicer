FROM golang:alpine as builder

# Force Go to use the cgo based DNS resolver. This is required to ensure DNS
# queries required to connect to linked containers succeed.
ENV GODEBUG netdns=cgo

# Install dependencies and build the binaries.
RUN apk add --no-cache --update wget

RUN mkdir -p /go/bin
WORKDIR /go/bin
RUN wget "https://github.com/lncm/invoicer/releases/download/v0.0.11/invoicer-linux-arm" \
    && chmod 755 invoicer-linux-arm 

# Start a new, final image.
FROM alpine as final

# Add bash and ca-certs, for quality of life and SSL-related reasons.
RUN apk --no-cache add \
    bash \
    ca-certificates

# Create directory for data assets
RUN mkdir -p /invoicer-data
WORKDIR /invoicer-data

# Copies index.html file into
COPY index.html /invoicer-data

# Copy the binaries from the builder image.
COPY --from=builder /go/bin/invoicer-linux-arm /bin/

# Expose lnd ports (p2p, rpc).
EXPOSE 1666

# Invoicer Entrypoint
CMD ["invoicer-linux-arm", "-lnd-host=localhost", "-lnd-invoice=/lnd/data/chain/bitcoin/mainnet/invoice.macaroon", "-lnd-readonly=/lnd/data/chain/bitcoin/mainnet/readonly.macaroon", "-mainnet", "-lnd-tls=/lnd/tls.cert"]
