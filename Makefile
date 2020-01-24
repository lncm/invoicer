# Define lnd version being used on the backend
LND_VERSION = v0.9.0-beta

VERSION = v0.8.1

GO111MODULE = on

PREFIX :=  CGO_ENABLED=0
BUILD_FLAGS := -ldflags "-s -w -buildid= -X main.gitHash=$$(git rev-parse HEAD)"
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

bin/linux-arm64v8/invoicer: $(SRC)
	env GOOS=linux GOARCH=arm64 		$(GOBUILD) -o $@

run: $(SRC)
	go run main.go  -config ./invoicer.conf

clean:
	rm -rf bin/*


#
# PROTO stuff
#
ln/lnd:
	mkdir -p $@

ln/lnd/google/api:
	mkdir -p $@

# Fetch rpc.proto files from https://github.com/lightningnetwork/lnd
ln/lnd/rpc.proto: ln/lnd/%: ln/lnd
	wget -qO - https://raw.githubusercontent.com/lightningnetwork/lnd/$(LND_VERSION)/lnrpc/$* | \
	 	sed 's|github.com/lightningnetwork/lnd/lnrpc|lnd|' > $@

# Fetch .proto files that ln/lnd/rpc.proto depends on
#  Files fetched from https://github.com/googleapis/googleapis are:
#	* google/api/annotations.proto, and
#	* google/api/http.proto
$(patsubst %, ln/lnd/google/api/%.proto, annotations http): ln/lnd/%: ln/lnd/google/api
	wget -qO $@  https://raw.githubusercontent.com/googleapis/googleapis/master/$*

clean-proto:
	rm -rf ln/lnd/

proto:   clean-proto  ln/lnd/rpc.proto  ln/lnd/google/api/annotations.proto  ln/lnd/google/api/http.proto
	go generate ./ln/...


# Linter install instructions are here:
#	https://github.com/golangci/golangci-lint#install
lint:
	golangci-lint run ./...

lint-all:
	golangci-lint run --enable-all ./...


static/index.html:
	mkdir -p static
	curl -s https://api.github.com/repos/lncm/donations/releases/latest \
		| jq -r '.assets[0].browser_download_url' \
		| wget -O $@ -qi -


.PHONY: run clean  proto clean-proto  lint lint-all  static/index.html
