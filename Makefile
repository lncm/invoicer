SRC := $(shell find . -type f -name '*.go')

bin/invoicer: $(SRC)
	go build -o $@

bin/linux-arm/invoicer:  $(SRC)
	env GOOS=linux GOARCH=arm GOARM=5 go build -o $@

bin/linux-amd64/invoicer:  $(SRC)
	env GOOS=linux GOARCH=amd64 go build -o $@

run: $(SRC)
	go run main.go

REMOTE_USER=root
REMOTE_HOST=pi-other
REMOTE_DIR=/home/lnd/bin/
deploy: bin/linux-arm/invoicer
	rsync $< "${REMOTE_USER}@${REMOTE_HOST}:${REMOTE_DIR}"

clean:
	rm -rf bin/*

.PHONY: run deploy clean
