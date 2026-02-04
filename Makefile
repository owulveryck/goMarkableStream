export GOOS ?= linux
export CGO_ENABLED ?= 0
export GOARCH ?= amd64

.PHONY: build-remarkable-2
build-remarkable-2: GOARCH=arm
build-remarkable-2: build

.PHONY: build-remarkable-paper-pro
build-remarkable-paper-pro: GOARCH=arm64
build-remarkable-paper-pro: build

.PHONY: build
build:
	@echo "Building for GOOS=${GOOS}, GOARCH=${GOARCH}, CGO_ENABLED=${CGO_ENABLED}"
	go build -v -trimpath -tags tailscale -ldflags="-s -w" .
