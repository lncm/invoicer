VERSION = v0.1.1

VERSION_STAMP="main.version=$(VERSION)"
VERSION_HASH="main.gitHash=$$(git rev-parse HEAD)"
BUILD_FLAGS="-X ${VERSION_STAMP} -X ${VERSION_HASH}"

SRC := $(shell find . -type f -name '*.go')

bin/invoicer: $(SRC)
	go build -o $@ -ldflags ${BUILD_FLAGS}

bin/invoicer-linux-arm: $(SRC)
	env GOOS=linux GOARCH=arm GOARM=5 go build -o $@  -ldflags ${BUILD_FLAGS}

bin/invoicer-linux-amd64: $(SRC)
	env GOOS=linux GOARCH=amd64 go build -o $@  -ldflags ${BUILD_FLAGS}

bin/invoicer-darwin: $(SRC)
	env GOOS=darwin GOARCH=amd64 go build -o $@  -ldflags ${BUILD_FLAGS}

bin/invoicer-freebsd-amd64: $(SRC)
	env GOOS=freebsd GOARCH=amd64 go build -o $@  -ldflags ${BUILD_FLAGS}

bin/invoicer-openbsd-amd64: $(SRC)
	env GOOS=openbsd GOARCH=amd64 go build -o $@  -ldflags ${BUILD_FLAGS}

run: $(SRC)
	go run main.go

tag:
	git tag -sa $(VERSION) -m "$(VERSION)"

all: tag bin/invoicer-linux-arm bin/invoicer-linux-amd64 bin/invoicer-darwin

static/index.html:
	wget -P static -N https://raw.githubusercontent.com/lncm/invoicer-ui/master/dist/index.html

REMOTE_USER ?= root
REMOTE_HOST ?= pi-hdd
REMOTE_DIR ?= /home/ln/bin/
deploy: bin/invoicer-linux-arm static/index.html
	rsync $< "${REMOTE_USER}@${REMOTE_HOST}:${REMOTE_DIR}"
	rsync static/index.html "${REMOTE_USER}@${REMOTE_HOST}:${REMOTE_DIR}"

clean:
	rm -rf bin/*

.PHONY: run tag all deploy clean static/index.html

