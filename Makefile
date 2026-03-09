.PHONY: build test run clean

build:
	go build -o server/marvin-relay ./server

test:
	go test ./server/...

run: build
	./server/marvin-relay --config server/config

clean:
	rm -f server/marvin-relay
