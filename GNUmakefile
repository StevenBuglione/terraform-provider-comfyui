default: build

BINARY=terraform-provider-comfyui
HOSTNAME=registry.terraform.io
NAMESPACE=StevenBuglione
NAME=comfyui
VERSION=0.1.0
OS_ARCH=$(shell go env GOOS)_$(shell go env GOARCH)
TOOLS_BIN=$(CURDIR)/.bin
LEFTHOOK=$(TOOLS_BIN)/lefthook
LEFTHOOK_VERSION=v2.1.5
GOLANGCI_LINT=$(TOOLS_BIN)/golangci-lint
GOLANGCI_LINT_VERSION=v2.11.4

$(TOOLS_BIN):
	mkdir -p $(TOOLS_BIN)

$(LEFTHOOK): | $(TOOLS_BIN)
	GOBIN=$(TOOLS_BIN) go install github.com/evilmartians/lefthook/v2@$(LEFTHOOK_VERSION)

$(GOLANGCI_LINT): | $(TOOLS_BIN)
	GOBIN=$(TOOLS_BIN) go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)

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
	go run ./cmd/generate

lint: $(GOLANGCI_LINT)
	$(GOLANGCI_LINT) run ./...

fmt:
	gofmt -s -w .

fmt-check:
	@files=$$(git ls-files '*.go'); \
	if [ -z "$$files" ]; then \
		exit 0; \
	fi; \
	unformatted=$$(gofmt -l $$files); \
	if [ -n "$$unformatted" ]; then \
		echo "Unformatted Go files:"; \
		echo "$$unformatted"; \
		exit 1; \
	fi

tidy:
	go mod tidy

vet:
	go vet ./...

docs:
	go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs generate --provider-name comfyui

tools: $(LEFTHOOK) $(GOLANGCI_LINT)

hooks-install: $(LEFTHOOK)
	$(LEFTHOOK) install --force

hooks-run-pre-commit: $(LEFTHOOK)
	$(LEFTHOOK) run pre-commit

hooks-run-pre-push: $(LEFTHOOK)
	$(LEFTHOOK) run pre-push

workspace-e2e-browser-install:
	cd validation/workspace_e2e/browser && npm install && npx playwright install chromium

workspace-e2e:
	./scripts/workspace-e2e/run.sh && cd validation/workspace_e2e/browser && npx playwright test tests/workspace_layout.spec.ts --project=chromium

release-e2e-browser-install:
	cd validation/release_e2e/browser && npm install && npx playwright install chromium

release-e2e:
	./scripts/release-e2e/run.sh && cd validation/release_e2e/browser && npx playwright test tests/release_workflows.spec.ts --project=chromium

execution-e2e:
	./scripts/execution-e2e/run.sh

verify: fmt-check generate vet lint test

clean:
	rm -f $(BINARY)

.PHONY: build install test testacc generate lint fmt fmt-check tidy vet docs tools hooks-install hooks-run-pre-commit hooks-run-pre-push workspace-e2e-browser-install workspace-e2e release-e2e-browser-install release-e2e execution-e2e verify clean
