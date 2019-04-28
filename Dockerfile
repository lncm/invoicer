FROM golang:1.12-alpine3.9 as builder

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

LABEL maintainer="Damian Mee (@meeDamian)"

# Copy the binaries from the builder image.
COPY --from=builder /src/bin/invoicer /bin/

VOLUME /root/.invoicer

# Expose Invoicer port
EXPOSE 8080

# Specify the start command and entrypoint as the invoicer daemon.
ENTRYPOINT ["invoicer"]
CMD ["invoicer"]

