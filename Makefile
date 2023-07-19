# Go parameters
GOCMD?=go
GO_VERSION=$(shell go list -m -f "{{.GoVersion}}")
LIBPAKTOOLS_VERSION=$(shell ./scripts/version.sh)
PACKAGE_BASE=github.com/paketo-buildpacks/libpak-tools
OUTDIR=./binaries
LDFLAGS="-s -w"

all: test build-all

build-all: create-package update-build-image-dependency update-package-dependency update-buildmodule-dependency update-lifecycle-dependency

out:
	mkdir -p $(OUTDIR)

create-package: out
	@echo "> Building create-package..."
	go build -ldflags=$(LDFLAGS) -o $(OUTDIR)/create-package create-package/main.go

update-build-image-dependency: out
	@echo "> Building update-build-image-dependency..."
	go build -ldflags=$(LDFLAGS) -o $(OUTDIR)/update-build-image-dependency update-build-image-dependency/main.go

update-package-dependency: out
	@echo "> Building update-package-dependency..."
	go build -ldflags=$(LDFLAGS) -o $(OUTDIR)/update-package-dependency update-package-dependency/main.go

update-buildmodule-dependency: out
	@echo "> Building update-buildmodule-dependency..."
	go build -ldflags=$(LDFLAGS) -o $(OUTDIR)/update-buildmodule-dependency update-buildmodule-dependency/main.go

update-lifecycle-dependency: out
	@echo "> Building update-lifecycle-dependency..."
	go build -ldflags=$(LDFLAGS) -o $(OUTDIR)/update-lifecycle-dependency update-lifecycle-dependency/main.go

package: OUTDIR=./bin
package: clean build-all
	@echo "> Packaging up binaries..."
	mkdir -p dist/
	tar czf dist/libpak-tools-$(LIBPAKTOOLS_VERSION).tgz $(OUTDIR)/*
	rm -rf ./bin

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
