.PHONY: build watch dist tidy precommit install-hooks-windows

build:
	go build ./...

dist: build
	go build .

watch:
	modd

tidy: build
	go mod tidy
	go fmt ./...

precommit: tidy

install-hooks-windows:
	copy hooks\\pre-commit.windows .git\\hooks\\pre-commit
	copy hooks\\pre-commit.ps1 .git\\hooks\\pre-commit.ps1
