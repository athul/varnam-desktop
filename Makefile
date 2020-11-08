BIN := varnam
ifeq ($(OS),Windows_NT)
	#HASH := ${shell powershell -Command git rev-parse HEAD | % {echo $_.subString(0, 8)}}
	HASH := ${shell git rev-parse HEAD}
	BUILD_DATE := $(shell powershell "Get-Date")
	ARCH := $(shell powershell "(Get-WmiObject Win32_Processor).AddressWidth")
else
	HASH := $(shell git rev-parse HEAD | cut -c 1-8)
	BUILD_DATE := $(shell date '+%Y-%m-%d %H:%M:%S')
	ARCH := $(shell uname -m)
endif
COMMIT_DATE := $(shell git show -s --format=%ci ${HASH})
VERSION := ${HASH} (${COMMIT_DATE})
PRETTY_VERSION := $(shell git describe --abbrev=0 --tags)
RELEASE_NAME := varnam-${PRETTY_VERSION}
STATIC := ui:/
LDFLAGS := -X 'main.buildVersion=${VERSION}' -X 'main.buildDate=${BUILD_DATE}' -s -w

ifeq ($(OS),Windows_NT)
	LDFLAGS := $(LDFLAGS) -H windowsgui
	BIN := $(BIN).exe
	RELEASE_NAME := $(RELEASE_NAME)-windows-${ARCH}
else
	RELEASE_NAME := $(RELEASE_NAME)-linux-${ARCH}
endif

libvarnam-windows:
	cd libvarnam/libvarnam & cmake -Bbuild . & cd build & cmake --build . --config Release && copy Release\varnam.dll ..\

.ONESHELL:
libvarnam-nix:
	cd libvarnam/libvarnam
	mkdir -p build
	cd build
	cmake ..
	cmake --build . --config Release
	cp libvarnam.so.3* ../

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

release-linux:
	mkdir ${RELEASE_NAME}
	cp -a varnam.sh varnam libvarnam/libvarnam/libvarnam.so.3* config.toml ${RELEASE_NAME}
	tar -cvzf ${RELEASE_NAME}.tar.gz ${RELEASE_NAME}

release-windows:
	mkdir ${RELEASE_NAME}
	copy varnam.exe ${RELEASE_NAME}
	copy windows-setup.bat ${RELEASE_NAME}
	copy "libvarnam\libvarnam\varnam.dll" ${RELEASE_NAME}
	copy "config.toml" ${RELEASE_NAME}
	copy "webview.dll" ${RELEASE_NAME}
	copy "WebView2Loader.dll" ${RELEASE_NAME}
	tar -acvf ${RELEASE_NAME}.zip ${RELEASE_NAME}

ifeq ($(OS),Windows_NT)
release-os:
	$(MAKE) release-windows
else
release-os:
	$(MAKE) release-linux
endif

release:
	$(MAKE) editor
	$(MAKE) build
	$(MAKE) release-os

.PHONY: run
run: build
	./${BIN}
.PHONY: clean
clean: ## Remove temporary files and the binary
	go clean

# Absolutely awesome: http://marmelab.com/blog/2016/02/29/auto-documented-makefile.html
.PHONY: help
help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

.DEFAULT_GOAL := build