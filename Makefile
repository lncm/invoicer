BUILD_FLAGS := -ldflags "-s -w -buildid= -X main.gitHash=$$(git rev-parse HEAD)"

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

bin/linux-arm64v8/invoicer: $(SRC)
	env GOOS=linux GOARCH=arm64 		$(GOBUILD) -o $@

run: $(SRC)
	go run main.go -config ./invoicer.conf

clean:
	rm -rf bin/*

static/index.html:
	mkdir -p static
	curl -s https://api.github.com/repos/lncm/donations/releases/latest \
		| jq -r '.assets[0].browser_download_url' \
		| wget -O $@ -qi -

.PHONY: run clean static/index.html
