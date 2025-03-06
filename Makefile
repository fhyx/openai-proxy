.SILENT :
.PHONY : main clean dist package

WITH_ENV = env `cat .env 2>/dev/null | xargs`

NAME:=openai-proxy
ROOF:=hyyl.xyz/gopak/openai-proxy
SOURCES=$(shell find . -type f \( -name "*.go" ! -name "*_test.go" \) -print )
DATE := $(shell date '+%Y%m%d')
TAG:=$(shell git describe --tags --always)
LDFLAGS:=-X main.name=$(NAME) -X main.version=$(DATE)-$(TAG)
GO=$(shell which go)
GOMOD=$(shell echo "$${GO111MODULE:-auto}")
GOTAG=

main: vet
	echo "Building $(NAME)"
	GO111MODULE=$(GOMOD) $(GO) build -ldflags "$(LDFLAGS)" .

all: clean dist package

vet:
	echo "Checking ./... , with GOMOD=$(GOMOD)"
	GO111MODULE=$(GOMOD) $(GO) vet .

deps:
	GO111MODULE=on $(GO) install -v github.com/golangci/golangci-lint/cmd/golangci-lint@latest

lint:
	GO111MODULE=$(GOMOD) golangci-lint run  -v .

clean:
	echo "Cleaning dist"
	rm -rf dist
	rm -f ./$(NAME)-*

showver:
	echo "version: $(TAG)"

dist/linux_amd64/$(NAME): $(SOURCES) showver
	echo "Building $(NAME) of linux"
	mkdir -p dist/linux_amd64 && GO111MODULE=$(GOMOD) GOOS=linux GOARCH=amd64 $(GO) build -tags=$(GOTAG) -ldflags "$(LDFLAGS) -s -w" -o dist/linux_amd64/$(NAME) .

dist/darwin_amd64/$(NAME): $(SOURCES) showver
	echo "Building $(NAME) of darwin/amd64"
	mkdir -p dist/darwin_amd64 && GO111MODULE=$(GOMOD) GOOS=darwin GOARCH=amd64 $(GO) build -tags=$(GOTAG) -ldflags "$(LDFLAGS) -w" -o dist/darwin_amd64/$(NAME) .

dist/darwin_arm64/$(NAME): $(SOURCES) showver
	echo "Building $(NAME) of darwin/arm64"
	mkdir -p dist/darwin_arm64 && GO111MODULE=$(GOMOD) GOOS=darwin GOARCH=arm64 $(GO) build -tags=$(GOTAG) -ldflags "$(LDFLAGS) -w" -o dist/darwin_arm64/$(NAME) .

dist: vet dist/linux_amd64/$(NAME) dist/darwin_amd64/$(NAME) dist/darwin_arm64/$(NAME)

package: dist
	echo "Packaging $(NAME)"
	ls dist/linux_amd64 | xargs tar -cvJf $(NAME)-linux-amd64-$(TAG).tar.xz -C dist/linux_amd64
	ls dist/darwin_amd64 | xargs tar -cvJf $(NAME)-darwin-amd64-$(TAG).tar.xz -C dist/darwin_amd64

.PHONY: package-deploy
package-deploy: package
	@echo "copy package.tar.?z to gopkg"
	@scp *-linux-amd64-*.tar.?z gopkg:gopkg/cupola/

dist-clean: clean
	rm -f $(NAME)-*.tar.xz
