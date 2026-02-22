.PHONY: build install clean test

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")

build:
	go build -ldflags "-X main.Version=$(VERSION)" -o task_go .

install: build
	mkdir -p ~/bin
	cp task_go ~/bin/task_go
	codesign --force --sign - ~/bin/task_go

clean:
	rm -f task_go

test:
	go test ./... -v
