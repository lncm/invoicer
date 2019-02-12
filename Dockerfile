FROM golang:1.11-alpine3.9 as builder

RUN apk add --no-cache --update alpine-sdk \
    make \
    upx

RUN mkdir -p /src/
COPY ./ /src/
WORKDIR /src/

ARG goos
ENV GOOS ${goos}

ARG goarch
ENV GOARCH ${goarch}

ARG goarm=6
ENV GOARM ${goarm}

RUN echo "GOOS:${GOOS} GOARCH:${GOARCH} GOARM:${GOARM}"

RUN make bin/invoicer

# compress output binary
RUN upx /src/bin/invoicer


# Start a new, final image.
FROM alpine:3.9 as final

# Required for endpoints and healthcheck
RUN apk add --no-cache --update bash \
    curl

# Copy the binaries from the builder image.
COPY --from=builder /src/bin/invoicer /bin/

# Copy healthcheck script
COPY check-invoicer.sh /bin/

RUN chmod 755 /bin/check-invoicer.sh

# Expose Invoicer port
EXPOSE 8080

# Health Check line
HEALTHCHECK --interval=30s --timeout=15s --retries=15 \
    CMD /bin/check-invoicer.sh || exit 1

# Specify the start command and entrypoint as the invoicer daemon.
ENTRYPOINT ["invoicer"]
CMD ["invoicer"]

