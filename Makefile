goos_linux := linux
goos_mac   := darwin
goarch     := amd64
now        := $(shell date '+%Y-%m-%d_%H:%M:%S')

.PHONY: init
init:
	go mod download
	touch .env .envrc

.PHONY: build
build:
	go build

.PHONY: run
run:
	set -a; . .env; go run main.go

.PHONY: releases
releases:
	GOOS=$(goos_linux) GOARCH=$(goarch) go build -o releases/github-actions-exporter.$(now).$(goos_linux).$(goarch)
	GOOS=$(goos_mac) GOARCH=$(goarch) go build -o releases/github-actions-exporter.$(now).$(goos_mac).$(goarch)
