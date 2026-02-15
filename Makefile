export GOOS ?= linux
export CGO_ENABLED ?= 0
export GOARCH ?= amd64

# Default build (no tracing)
.PHONY: build
build:
	@echo "Building for GOOS=${GOOS}, GOARCH=${GOARCH}, CGO_ENABLED=${CGO_ENABLED}"
	go build -v -trimpath -tags tailscale -ldflags="-s -w" .

# Build with tracing support
.PHONY: build-with-trace
build-with-trace:
	@echo "Building with trace support for GOOS=${GOOS}, GOARCH=${GOARCH}"
	go build -v -trimpath -tags "tailscale,trace" -ldflags="-s -w" .

# reMarkable 2 builds
.PHONY: build-remarkable-2
build-remarkable-2: GOARCH=arm
build-remarkable-2: GOARM=7
build-remarkable-2: build

.PHONY: build-remarkable-2-trace
build-remarkable-2-trace: GOARCH=arm
build-remarkable-2-trace: GOARM=7
build-remarkable-2-trace: build-with-trace

# reMarkable Paper Pro builds
.PHONY: build-remarkable-paper-pro
build-remarkable-paper-pro: GOARCH=arm64
build-remarkable-paper-pro: build

.PHONY: build-remarkable-paper-pro-trace
build-remarkable-paper-pro-trace: GOARCH=arm64
build-remarkable-paper-pro-trace: build-with-trace
