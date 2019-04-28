FROM golang:1.12-alpine3.9 as builder

RUN apk add --no-cache --update alpine-sdk \
    make \
    upx

RUN mkdir -p /src/
COPY ./ /src/
WORKDIR /src/

ARG arch

RUN make bin/${arch}/invoicer

RUN mkdir -p /bin \
    && mv bin/${arch}/invoicer /bin/

# compress output binary
RUN upx /bin/invoicer


# Start a new, final image.
FROM alpine:3.9 as final

LABEL maintainer="Damian Mee (@meeDamian)"

# Copy the binaries from the builder image.
COPY --from=builder /bin/invoicer /bin/

VOLUME /root/.invoicer

# Expose Invoicer port
EXPOSE 8080

# Specify the start command and entrypoint as the invoicer daemon.
ENTRYPOINT ["invoicer"]
CMD ["invoicer"]

