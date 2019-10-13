VERSION = v0.6.2

# NOTE: `-buildid=` is a fix as per: https://github.com/golang/go/issues/33772
BUILD_FLAGS := -ldflags "-s -w -buildid= -X main.version=$(VERSION) -X main.gitHash=$$(git rev-parse HEAD)"

GO111MODULE = on

PREFIX :=  CGO_ENABLED=0
GOBUILD := $(PREFIX) go build -v -trimpath -mod=readonly $(BUILD_FLAGS)

SRC := $(shell find . -type f -name '*.go') go.mod go.sum

bin/invoicer: $(SRC)
	$(PREFIX) go build -v -o $@

bin/invoicer-race: $(SRC)
	go build -v -o $@  -race


bin/darwin/invoicer: $(SRC)
	env GOOS=darwin GOARCH=amd64  		$(GOBUILD) -o $@

bin/linux-amd64/invoicer: $(SRC)
	env GOOS=linux GOARCH=amd64 		$(GOBUILD) -o $@

bin/linux-arm32v6/invoicer: $(SRC)
	env GOOS=linux GOARCH=arm GOARM=6 	$(GOBUILD) -o $@

bin/linux-arm32v7/invoicer: $(SRC)
	env GOOS=linux GOARCH=arm GOARM=7 	$(GOBUILD) -o $@

bin/linux-arm64/invoicer: $(SRC)
	env GOOS=linux GOARCH=arm64 		$(GOBUILD) -o $@


bin/invoicer-$(VERSION)-darwin.tgz: 		bin/darwin/invoicer
	tar -cvzf $@ $<
bin/invoicer-$(VERSION)-linux-amd64.tgz: 	bin/linux-amd64/invoicer
	tar -cvzf $@ $<
bin/invoicer-$(VERSION)-linux-arm32v6.tgz: 	bin/linux-arm32v6/invoicer
	tar -cvzf $@ $<
bin/invoicer-$(VERSION)-linux-arm32v7.tgz: 	bin/linux-arm32v7/invoicer
	tar -cvzf $@ $<
bin/invoicer-$(VERSION)-linux-arm64.tgz: 	bin/linux-arm64/invoicer
	tar -cvzf $@ $<

run: $(SRC)
	go run main.go -config ./invoicer.conf

tag:
	git tag -sa $(VERSION) -m "$(VERSION)"

ci: bin/invoicer-$(VERSION)-darwin.tgz \
	bin/invoicer-$(VERSION)-linux-amd64.tgz \
	bin/invoicer-$(VERSION)-linux-arm32v6.tgz \
	bin/invoicer-$(VERSION)-linux-arm32v7.tgz \
	bin/invoicer-$(VERSION)-linux-arm64.tgz

all: tag ci

clean:
	rm -rf bin/*


#
# My personal shortcuts, probably not useful to anyone else, but maybe 🤷🏻‍♂️
#
REMOTE_USER ?= ln
REMOTE_HOST ?= pi-hdd

static/index.html:
	mkdir -p static
	curl -s https://api.github.com/repos/lncm/donations/releases/latest \
		| grep "browser_download_url.*html" \
		| cut -d '"' -f 4 \
		| wget -O $@ -qi -

REMOTE_DIR_BINARY ?= /home/ln/bin/
deploy-invoicer: bin/linux-arm32v7/invoicer
	rsync $< "${REMOTE_USER}@${REMOTE_HOST}:${REMOTE_DIR_BINARY}"

REMOTE_DIR_STATIC ?= /home/ln/static/
deploy-static: static/index.html
	rsync $< "${REMOTE_USER}@${REMOTE_HOST}:${REMOTE_DIR_STATIC}"

deploy: deploy-invoicer deploy-static

.PHONY: run tag all deploy deploy-invoicer deploy-static clean ci static/index.html
