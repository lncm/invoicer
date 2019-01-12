FROM golang:alpine as builder

RUN apk add --no-cache --update alpine-sdk \
    make

RUN mkdir -p /src/
COPY ./ /src/
WORKDIR /src/

RUN make bin/invoicer


# Start a new, final image.
FROM alpine as final

RUN apk add --no-cache --update bash

# Create directory for data assets
RUN mkdir -p /static/

# TODO: switch this to invoicer-ui building stage
COPY --from=builder /src/static/ /static/

# Copy the binaries from the builder image.
COPY --from=builder /src/bin/invoicer /bin/

COPY entrypoint-invoicer.sh /bin/

RUN chmod 755 /bin/entrypoint-invoicer.sh

# Expose Invoicer port
EXPOSE 8080

# Invoicer Entrypoint
ENTRYPOINT entrypoint-invoicer.sh

