FROM golang:alpine as builder

# Force Go to use the cgo based DNS resolver. This is required to ensure DNS
# queries required to connect to linked containers succeed.
ENV GODEBUG netdns=cgo
ENV LND_DIR /lnd
ENV COIN bitcoin
ENV NETWORK mainnet

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

# Copy the binaries from the builder image.
COPY --from=builder /go/bin/invoicer-linux-arm /bin/

# Expose lnd ports (p2p, rpc).
EXPOSE 1666

# Invoicer Entrypoint
CMD ["invoicer-linux-arm", "-lnd-host localhost", "-lnd-invoice $LND_DIR/data/chain/$COIN/$NETWORK/invoice.macaroon", "-lnd-readonly $LND_DIR/data/chain/$COIN/$NETWORK/readonly.macaroon", "-mainnet", "-lnd-tls $LND_DIR/tls.cert"]
