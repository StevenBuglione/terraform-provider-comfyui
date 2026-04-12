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
COMFYUI_RUNTIME_DIR=$(CURDIR)/validation/comfyui_dev/.runtime

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
	./scripts/extract/run_ui_hints.sh
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

docs-validate:
	go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs validate --provider-name comfyui

docs-check: docs docs-validate
	@if ! git diff --compact-summary --exit-code -- docs; then \
		echo "::error::Documentation is out of date. Run 'make docs' and commit the changes."; \
		exit 1; \
	fi
	@if [ -n "$$(git ls-files --others --exclude-standard -- docs)" ]; then \
		echo "::error::Documentation is out of date. Commit the newly generated files under docs/."; \
		git ls-files --others --exclude-standard -- docs; \
		exit 1; \
	fi

tools: $(LEFTHOOK) $(GOLANGCI_LINT)

hooks-install: $(LEFTHOOK)
	$(LEFTHOOK) install --force

hooks-run-pre-commit: $(LEFTHOOK)
	$(LEFTHOOK) run pre-commit

hooks-run-pre-push: $(LEFTHOOK)
	$(LEFTHOOK) run pre-push

comfyui-start:
	@WORKSPACE_E2E_RUNTIME_DIR="$(COMFYUI_RUNTIME_DIR)" COMFYUI_HOST="$(COMFYUI_HOST)" COMFYUI_PORT="$(COMFYUI_PORT)" ./scripts/workspace-e2e/start-comfyui.sh

comfyui-stop:
	@WORKSPACE_E2E_RUNTIME_DIR="$(COMFYUI_RUNTIME_DIR)" COMFYUI_HOST="$(COMFYUI_HOST)" COMFYUI_PORT="$(COMFYUI_PORT)" ./scripts/workspace-e2e/stop-comfyui.sh

comfyui-status:
	@RUNTIME_DIR="$(COMFYUI_RUNTIME_DIR)"; \
	PID_FILE="$$RUNTIME_DIR/comfyui.pid"; \
	ENV_FILE="$$RUNTIME_DIR/runtime.env"; \
	if [ -f "$$PID_FILE" ] && kill -0 "$$(cat "$$PID_FILE")" 2>/dev/null; then \
		echo "ComfyUI running (pid=$$(cat "$$PID_FILE"))"; \
		echo "runtime_dir=$$RUNTIME_DIR"; \
		if [ -f "$$ENV_FILE" ]; then \
			. "$$ENV_FILE"; \
			echo "base_url=$$WORKSPACE_E2E_BASE_URL"; \
			echo "log_file=$$WORKSPACE_E2E_LOG_FILE"; \
		fi; \
	else \
		echo "ComfyUI not running"; \
		echo "runtime_dir=$$RUNTIME_DIR"; \
	fi

workspace-e2e-browser-install:
	cd validation/workspace_e2e/browser && npm install && npx playwright install chromium

workspace-e2e:
	./scripts/workspace-e2e/run.sh && cd validation/workspace_e2e/browser && npx playwright test tests/workspace_layout.spec.ts --project=chromium

release-e2e-browser-install:
	cd validation/release_e2e/browser && npm install && npx playwright install chromium

release-e2e:
	./scripts/release-e2e/run.sh && cd validation/release_e2e/browser && npx playwright test tests/release_workflows.spec.ts --project=chromium

synthesis-e2e:
	./scripts/synthesis-e2e/run.sh

inventory-plan-e2e:
	./scripts/inventory-plan-e2e/run.sh

execution-e2e:
	./scripts/execution-e2e/run.sh

validate-release-version:
	@if [ "$(VERSION)" = "0.1.0" ] || [ -z "$(VERSION)" ]; then \
		echo "ERROR: VERSION not provided or using default dev version"; \
		echo "Usage: make validate-release-version VERSION=v0.18.5"; \
		exit 1; \
	fi
	./scripts/validate-release-version.sh $(VERSION)

verify: fmt-check generate docs-check vet lint test

release-preflight:
	@if [ "$(VERSION)" = "0.1.0" ] || [ -z "$(VERSION)" ]; then \
		echo "ERROR: VERSION not provided or using default dev version"; \
		echo "Usage: make release-preflight VERSION=v0.18.5"; \
		exit 1; \
	fi
	@if [ -n "$$(git status --porcelain)" ]; then \
		echo "ERROR: release-preflight requires a clean worktree."; \
		echo "Commit or stash local changes before running release preflight."; \
		exit 1; \
	fi
	$(MAKE) verify
	$(MAKE) validate-release-version VERSION=$(VERSION)

clean:
	rm -f $(BINARY)

.PHONY: build install test testacc generate lint fmt fmt-check tidy vet docs docs-validate docs-check tools hooks-install hooks-run-pre-commit hooks-run-pre-push comfyui-start comfyui-stop comfyui-status workspace-e2e-browser-install workspace-e2e release-e2e-browser-install release-e2e synthesis-e2e inventory-plan-e2e execution-e2e validate-release-version verify release-preflight clean
