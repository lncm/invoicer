VERSION = 0.2.14

VERSION_STAMP="main.version=v$(VERSION)"
VERSION_HASH="main.gitHash=$$(git rev-parse HEAD)"
BUILD_FLAGS="-X ${VERSION_STAMP} -X ${VERSION_HASH}"

SRC := $(shell find . -type f -name '*.go')

bin/invoicer: $(SRC)
	go build -o $@ -ldflags ${BUILD_FLAGS}

bin/invoicer-race: $(SRC)
	go build -race -o $@ -ldflags ${BUILD_FLAGS}


bin/darwin/invoicer: $(SRC)
	env GOOS=darwin GOARCH=amd64 go build -o $@  -ldflags ${BUILD_FLAGS}

bin/linux-arm/invoicer: $(SRC)
	env GOOS=linux GOARCH=arm GOARM=5 go build -o $@  -ldflags ${BUILD_FLAGS}

bin/linux-amd64/invoicer: $(SRC)
	env GOOS=linux GOARCH=amd64 go build -o $@  -ldflags ${BUILD_FLAGS}

bin/freebsd-amd64/invoicer: $(SRC)
	env GOOS=freebsd GOARCH=amd64 go build -o $@  -ldflags ${BUILD_FLAGS}

bin/openbsd-amd64/invoicer: $(SRC)
	env GOOS=openbsd GOARCH=amd64 go build -o $@  -ldflags ${BUILD_FLAGS}


bin/invoicer-$(VERSION)-darwin.tgz: 		bin/darwin/invoicer
	tar -cvzf $@ $<
bin/invoicer-$(VERSION)-linux-arm.tgz: 		bin/linux-arm/invoicer
	tar -cvzf $@ $<
bin/invoicer-$(VERSION)-linux-amd64.tgz: 	bin/linux-amd64/invoicer
	tar -cvzf $@ $<
bin/invoicer-$(VERSION)-freebsd-amd64.tgz: 	bin/freebsd-amd64/invoicer
	tar -cvzf $@ $<
bin/invoicer-$(VERSION)-openbsd-amd64.tgz: 	bin/openbsd-amd64/invoicer
	tar -cvzf $@ $<


run: $(SRC)
	go run main.go

tag:
	git tag -sa $(VERSION) -m "v$(VERSION)"

ci: bin/invoicer-$(VERSION)-darwin.tgz \
	bin/invoicer-$(VERSION)-linux-arm.tgz \
	bin/invoicer-$(VERSION)-linux-amd64.tgz \
	bin/invoicer-$(VERSION)-freebsd-amd64.tgz \
	bin/invoicer-$(VERSION)-openbsd-amd64.tgz

all: tag ci

REMOTE_USER ?= root
REMOTE_HOST ?= pi-hdd
REMOTE_DIR ?= /home/ln/bin/
REMOTE_STATIC ?= /home/ln/static/
deploy: bin/linux-arm/invoicer
	rsync $< "${REMOTE_USER}@${REMOTE_HOST}:${REMOTE_DIR}"
	rsync static/index.html "${REMOTE_USER}@${REMOTE_HOST}:${REMOTE_STATIC}"

clean:
	rm -rf bin/*

.PHONY: run tag all deploy clean ci

