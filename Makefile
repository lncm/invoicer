SRC := $(shell find . -type f -name '*.go')

bin/invoicer: $(SRC)
	go build -o $@

bin/invoicer-linux-arm: $(SRC)
	env GOOS=linux GOARCH=arm GOARM=5 go build -o $@

bin/invoicer-linux-amd64: $(SRC)
	env GOOS=linux GOARCH=amd64 go build -o $@

bin/invoicer-darwin: $(SRC)
	env GOOS=darwin GOARCH=amd64 go build -o $@

bin/invoicer-freebsd-amd64: $(SRC)
	env GOOS=freebsd GOARCH=amd64 go build -o $@

bin/invoicer-openbsd-amd64: $(SRC)
	env GOOS=openbsd GOARCH=amd64 go build -o $@

run: $(SRC)
	go run main.go

all: bin/invoicer-linux-arm bin/invoicer-linux-amd64 bin/invoicer-darwin bin/invoicer-freebsd-amd64 bin/invoicer-openbsd-amd64

common/index.html:
	wget -NP common/ https://raw.githubusercontent.com/lncm/invoicer-ui/master/dist/index.html

REMOTE_USER ?= root
REMOTE_HOST ?= pi-hdd
REMOTE_DIR ?= /home/ln/bin/
deploy: bin/invoicer-linux-arm common/index.html
	rsync $< "${REMOTE_USER}@${REMOTE_HOST}:${REMOTE_DIR}"
	rsync common/index.html "${REMOTE_USER}@${REMOTE_HOST}:${REMOTE_DIR}"

clean:
	rm -rf bin/*

.PHONY: run all deploy clean common/index.html

