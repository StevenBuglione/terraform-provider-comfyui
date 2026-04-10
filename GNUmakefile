default: build

BINARY=terraform-provider-comfyui
HOSTNAME=registry.terraform.io
NAMESPACE=StevenBuglione
NAME=comfyui
VERSION=0.1.0
OS_ARCH=$(shell go env GOOS)_$(shell go env GOARCH)

build:
	go build -o $(BINARY)

install: build
	mkdir -p ~/.terraform.d/plugins/$(HOSTNAME)/$(NAMESPACE)/$(NAME)/$(VERSION)/$(OS_ARCH)
	cp $(BINARY) ~/.terraform.d/plugins/$(HOSTNAME)/$(NAMESPACE)/$(NAME)/$(VERSION)/$(OS_ARCH)/

test:
	go test ./... -v $(TESTARGS) -timeout 120s

testacc:
	TF_ACC=1 go test ./... -v $(TESTARGS) -timeout 120m

generate:
	go generate ./...

lint:
	golangci-lint run ./...

fmt:
	gofmt -s -w .

vet:
	go vet ./...

docs:
	go generate ./...

clean:
	rm -f $(BINARY)

.PHONY: build install test testacc generate lint fmt vet docs clean
