#
#  xilt / Makefile
#

APP_VERSION ?= 0.0.1
APP_MAIN    := ./cmd/xilt/main.go
OUTPUT_PATH := .

#
#  Dev targets
#

fmt:
	@gofmt -w -s .

build: fmt
	@go build -o ${OUTPUT_PATH} ./cmd/...

push: build
	@git tag -fa "v${APP_VERSION}" -m "v${APP_VERSION}"
	@git push --follow-tags origin master

run: 
	@rm -f logs.db 
	@go build -o xilt ./cmd/xilt/main.go
	@./xilt -i -v logs logs.db

test:
	@go test -v -coverprofile cover.out ./...
	@go tool cover -html cover.out -o cover.html
	@open cover.html