# Yanked from Kolide Makefile
# MIT License
# Copyright (c) 2017 Kolide

.PHONY: build

PATH := $(GOPATH)/bin:$(shell npm bin):$(PATH)
VERSION = $(shell git describe --tags --always --dirty)
BRANCH = $(shell git rev-parse --abbrev-ref HEAD)
REVISION = $(shell git rev-parse HEAD)
REVSHORT = $(shell git rev-parse --short HEAD)
USER = $(shell whoami)
DOCKER_IMAGE_NAME = deeso/go-certs

ifneq ($(OS), Windows_NT)
    # If on macOS, set the shell to bash explicitly
    ifeq ($(shell uname), Darwin)
        SHELL := /bin/bash
    endif

    # The output binary name is different on Windows, so we're explicit here
    OUTPUT = go-certs

    # To populate version metadata, we use unix tools to get certain data
    GOVERSION = $(shell go version | awk '{print $$3}')
    NOW = $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
else
    # The output binary name is different on Windows, so we're explicit here
    OUTPUT = go-certs.exe

    # To populate version metadata, we use windows tools to get the certain data
    GOVERSION_CMD = "(go version).Split()[2]"
    GOVERSION = $(shell powershell $(GOVERSION_CMD))
    NOW = $(shell powershell Get-Date -format s)
endif

ifndef CIRCLE_PR_NUMBER
    DOCKER_IMAGE_TAG = ${REVSHORT}
else
    DOCKER_IMAGE_TAG = dev-${CIRCLE_PR_NUMBER}-${REVSHORT}
endif

ifdef CIRCLE_TAG
    DOCKER_IMAGE_TAG = ${CIRCLE_TAG}
endif

all: build

define HELP_TEXT

  Makefile commands

    make deps         - Install dependent programs and libraries
    make generate     - Generate and bundle required code
    make distclean    - Delete all build artifacts

    make build        - Build the code
    make package      - Build rpm and deb packages for linux

    make test         - Run the full test suite
    make test-go      - Run the Go tests

    make lint         - Run all linters
    make lint-go      - Run the Go linters

endef

help:
    $(info $(HELP_TEXT))

.prefix:
ifeq ($(OS), Windows_NT)
    if not exist build mkdir build
else
    mkdir -p build/linux
    mkdir -p build/darwin
endif

.pre-build:
    $(eval GOGC = off)
    $(eval CGO_ENABLED = 0)

.pre-go-certs:
    $(eval APP_NAME = go-certs)

build: go-certs

go-certs: .prefix .pre-build .pre-go-certs
    go build -i -o build/${OUTPUT} ./cmd/go-certs

lint-go:
    go vet ./...

lint: lint-go lint-js lint-scss lint-ts

test-go:
    go test ./...

analyze-go:
    go test -race -cover ./...


test: lint test-go

deps:
    go get -u \
        github.com/golang/dep/cmd/dep \
        github.com/groob/mockimpl
    dep ensure -vendor-only

distclean:
ifeq ($(OS), Windows_NT)
    if exist build rmdir /s/q build
    if exist vendor rmdir /s/q vendor
else
    rm -rf build vendor
endif

docker-build-release: .pre-go-certs
    GOOS=linux go build -i -o build/linux/${OUTPUT} ./cmd/go-certs
    docker build -t "${DOCKER_IMAGE_NAME}:${DOCKER_IMAGE_TAG}" .
    docker tag "${DOCKER_IMAGE_NAME}:${DOCKER_IMAGE_TAG}" deeso/go-certs:latest

docker-push-release: docker-build-release
    docker push "${DOCKER_IMAGE_NAME}:${DOCKER_IMAGE_TAG}"
    docker push deeso/go-certs:latest

docker-build-circle:
    @echo ">> building docker image"
    GOOS=linux go build -i -o build/linux/${OUTPUT} -ldflags ${KIT_VERSION} ./cmd/go-certs
    docker build -t "${DOCKER_IMAGE_NAME}:${DOCKER_IMAGE_TAG}" .
    docker push "${DOCKER_IMAGE_NAME}:${DOCKER_IMAGE_TAG}"

binary-bundle: generate
    rm -rf build/binary-bundle
    mkdir -p build/binary-bundle/linux
    mkdir -p build/binary-bundle/darwin
    GOOS=linux go build -i -o build/binary-bundle/linux/${OUTPUT}_linux_amd64 ./cmd/go-certs
    GOOS=darwin go build -i -o build/binary-bundle/darwin/${OUTPUT}_darwin_amd64 ./cmd/go-certs
    cd build/binary-bundle && zip -r "go-certs_${VERSION}.zip" darwin/ linux/
    cp build/binary-bundle/go-certs_${VERSION}.zip build/binary-bundle/go-certs_lastest.zip
