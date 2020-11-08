BIN := varnam
HASH := $(shell git rev-parse HEAD | cut -c 1-8)
BUILD_DATE := $(shell date '+%Y-%m-%d %H:%M:%S')
ARCH := $(shell uname -m)
COMMIT_DATE := $(shell git show -s --format=%ci ${HASH})
VERSION := ${HASH} (${COMMIT_DATE})
PRETTY_VERSION := $(shell git describe --abbrev=0 --tags)
RELEASE_NAME := varnam-${PRETTY_VERSION}
STATIC := ui:/
LDFLAGS := -X 'main.buildVersion=${VERSION}' -X 'main.buildDate=${BUILD_DATE}' -s -w

ifeq ($(OS),Windows_NT)
LDFLAGS := $(LDFLAGS) -H windowsgui
BIN := $(BIN).exe
RELEASE_NAME_32 := $(RELEASE_NAME)-windows-32
RELEASE_NAME := $(RELEASE_NAME)-windows-${ARCH}
else
RELEASE_NAME := $(RELEASE_NAME)-linux-${ARCH}
endif

libvarnam-windows:
	cd libvarnam/libvarnam && cmake -Bbuild . && cd build && cmake --build . --config Release && cp Release/varnam.dll ../

libvarnam-windows-32:
	cd libvarnam/libvarnam && cmake -A Win32 -Bbuild . && cd build && cmake --build . --config Release && cp Release/varnam.dll ../

.ONESHELL:
libvarnam-nix:
	cd libvarnam/libvarnam
	mkdir -p build
	cd build
	cmake ..
	cmake --build . --config Release
	cp -P libvarnam.so* ../

ifeq ($(OS),Windows_NT)
.PHONY: libvarnam
libvarnam: libvarnam-windows

.PHONY: editor
editor:
	build-editor.bat
else
.PHONY: libvarnam
libvarnam: libvarnam-nix

.PHONY: editor
editor:
	./build-editor.sh
endif

deps:
	go get -u github.com/knadh/stuffbin/...

.PHONY: build
build: ## Build the binary (default)
	go build -ldflags="${LDFLAGS}" -o ${BIN}
	stuffbin -a stuff -in ${BIN} -out ${BIN} ${STATIC}

# 32-bit releases are only for Windows
build-32:
	set GOARCH=386
	$(MAKE) build

release-linux:
	mkdir -p ${RELEASE_NAME}
	cp varnam.sh varnam libvarnam/libvarnam/libvarnam.so.3 config.toml ${RELEASE_NAME}
	tar -cvzf ${RELEASE_NAME}.tar.gz ${RELEASE_NAME}

release-windows:
	mkdir -p ${RELEASE_NAME}
	cp -a varnam.exe windows-setup.bat libvarnam/libvarnam/varnam.dll config.toml ${RELEASE_NAME}
	powershell "Compress-Archive -Force ${RELEASE_NAME} ${RELEASE_NAME}.zip"

release-windows-32:
	mkdir -p ${RELEASE_NAME_32}
	cp -a varnam.exe windows-setup.bat libvarnam/libvarnam/varnam.dll config.toml ${RELEASE_NAME_32}
	tar -acvf ${RELEASE_NAME_32}.zip ${RELEASE_NAME_32}

release-32:
	$(MAKE) libvarnam-windows-32
	$(MAKE) build-32
	$(MAKE) release-windows-32

ifeq ($(OS),Windows_NT)
release-os:
	$(MAKE) release-windows
else
release-os:
	$(MAKE) release-linux
endif

release:
	$(MAKE) libvarnam
	$(MAKE) editor
	$(MAKE) build
	$(MAKE) release-os

.PHONY: run
run: build
	./${BIN}

.PHONY: clean
clean: ## Remove temporary files and the binary
	go clean
	git clean -fdx
	cd libvarnam/libvarnam && git clean -fdx

# Absolutely awesome: http://marmelab.com/blog/2016/02/29/auto-documented-makefile.html
.PHONY: help
help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

.DEFAULT_GOAL := build