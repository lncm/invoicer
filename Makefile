SRC := $(shell find . -type f -name '*.go')

bin/invoicer: $(SRC)
	go build -o $@

bin/invoicer-linux-arm:  $(SRC)
	env GOOS=linux GOARCH=arm GOARM=5 go build -o $@

bin/invoicer-linux-amd64:  $(SRC)
	env GOOS=linux GOARCH=amd64 go build -o $@

bin/invoicer-darwin:  $(SRC)
	env GOOS=darwin GOARCH=amd64 go build -o $@

run: $(SRC)
	go run main.go

all: bin/invoicer-linux-arm bin/invoicer-linux-amd64 bin/invoicer-darwin

REMOTE_USER=root
REMOTE_HOST=pi-other
REMOTE_DIR=/home/lnd/bin/
deploy: bin/invoicer-linux-arm
	rsync $< "${REMOTE_USER}@${REMOTE_HOST}:${REMOTE_DIR}"

clean:
	rm -rf bin/*

.PHONY: run all deploy clean
