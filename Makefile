.PHONY: build install clean test

build:
	go build -o task_go .

install: build
	mkdir -p ~/bin
	cp task_go ~/bin/task_go
	codesign --force --sign - ~/bin/task_go

clean:
	rm -f task_go

test:
	go test ./... -v
