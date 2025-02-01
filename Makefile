# Go parameters
GOCMD?=go
GO_VERSION=$(shell go list -m -f "{{.GoVersion}}")
LIBPAKTOOLS_VERSION=$(shell ./scripts/version.sh)
PACKAGE_BASE=github.com/paketo-buildpacks/libpak-tools
OUTDIR=$(HOME)/go/bin
VERSION=$(shell git describe --always --long --dirty)
LDFLAGS="-s -w -X ${PACKAGE_BASE}/commands.version=${VERSION}"

all: test libpak-tools

out:
	mkdir -p $(OUTDIR)

libpak-tools: out
	@echo "> Building libpak-tools..."
	go build -ldflags=$(LDFLAGS) -o $(OUTDIR)/libpak-tools main.go

install-goimports:
	@echo "> Installing goimports..."
	cd tools && $(GOCMD) install golang.org/x/tools/cmd/goimports

format: install-goimports
	@echo "> Formating code..."
	@goimports -l -w -local ${PACKAGE_BASE} .

install-golangci-lint:
	@echo "> Installing golangci-lint..."
	cd tools && $(GOCMD) install github.com/golangci/golangci-lint/cmd/golangci-lint

lint: install-golangci-lint
	@echo "> Linting code..."
	@golangci-lint run -c golangci.yaml

test: format lint
	$(GOCMD) test -parallel=1 -count=1 -v ./...

clean:
	rm -rf ./bin/
	rm -rf ./binaries/
