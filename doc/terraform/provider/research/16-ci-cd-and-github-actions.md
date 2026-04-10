# 16 — CI/CD and GitHub Actions for Terraform Providers

## Overview

Continuous integration and delivery are essential for Terraform provider development.
A well-configured CI pipeline ensures that every pull request is validated against
multiple Terraform versions, that generated code is up to date, that documentation
compiles correctly, and that linting rules are satisfied before a merge is allowed.

This document covers the full CI/CD setup for a Plugin Framework provider using
GitHub Actions, including unit tests, acceptance tests, linting, documentation
validation, caching strategies, and HashiCorp's reusable release workflows.

---

## 1. Core CI Workflow — `.github/workflows/test.yml`

This workflow runs on every pull request targeting `main` and on every push to
`main`. It contains two jobs: a fast build-and-unit-test job and a slower
acceptance-test job that exercises real Terraform operations across multiple
Terraform versions.

```yaml
# .github/workflows/test.yml
name: Tests

on:
  pull_request:
    branches: [main]
    paths-ignore: ['README.md']
  push:
    branches: [main]

permissions:
  contents: read

jobs:
  # ------------------------------------------------------------------
  # Job 1: Build and Unit Tests
  # ------------------------------------------------------------------
  build:
    name: Build & Unit Tests
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version-file: 'go.mod'
          cache: true                   # caches ~/go/pkg/mod and build cache

      - run: go mod download
      - run: go build -v .
      - run: go test -v ./...

  # ------------------------------------------------------------------
  # Job 2: Acceptance Tests (matrix across Terraform versions)
  # ------------------------------------------------------------------
  testacc:
    name: Acceptance Tests (Terraform ${{ matrix.terraform }})
    runs-on: ubuntu-latest
    timeout-minutes: 15
    strategy:
      fail-fast: false
      matrix:
        terraform: ['1.8.*', '1.9.*']
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version-file: 'go.mod'
          cache: true

      - uses: hashicorp/setup-terraform@v3
        with:
          terraform_version: ${{ matrix.terraform }}
          terraform_wrapper: false      # IMPORTANT — see explanation below

      - run: go mod download

      - env:
          TF_ACC: "1"
        run: go test -v -cover ./internal/provider/ -timeout 120m
```

### Key Details

| Setting | Purpose |
|---|---|
| `go-version-file: 'go.mod'` | Reads the Go version from the `go` directive in `go.mod` so the CI always matches the development version. |
| `terraform_wrapper: false` | Disables the thin wrapper script that `hashicorp/setup-terraform` installs by default. The wrapper captures stdout and stderr for use in later steps, but it **breaks** the acceptance test harness because `terraform-plugin-testing` invokes Terraform as a subprocess and parses its output. Always set this to `false` for acceptance tests. |
| `fail-fast: false` | If one matrix leg fails, the others continue. This is essential for understanding whether a failure is version-specific or universal. |
| `timeout-minutes: 15` | Guards against runaway acceptance tests that spin up real infrastructure. Adjust upward if your provider manages slow-to-create resources. |
| `TF_ACC: "1"` | Enables acceptance tests. Without this variable, `resource.Test()` calls are silently skipped. |
| `paths-ignore: ['README.md']` | Avoids wasting CI minutes on doc-only commits. Extend the list as needed (e.g., `'docs/**'`, `'*.md'`). |

### Matrix Strategy — Testing Across Terraform Versions

The `matrix.terraform` array should include every **minor** version of Terraform
that you intend to support. Use the glob wildcard to pick up the latest patch
release:

```yaml
matrix:
  terraform:
    - '1.6.*'
    - '1.7.*'
    - '1.8.*'
    - '1.9.*'
```

If you need to test against the **latest pre-release** (alpha/beta/rc), add it
as an explicit version string, e.g., `'1.10.0-beta1'`.

> **Tip:** The `hashicorp/setup-terraform` action resolves the highest matching
> version from the HashiCorp Releases API. You do not need to update CI when a
> new patch version ships.

---

## 2. Linting with golangci-lint

Add a dedicated lint job so linting failures surface clearly and do not block the
main test matrix.

```yaml
# .github/workflows/lint.yml
name: Lint

on:
  pull_request:
    branches: [main]

permissions:
  contents: read

jobs:
  golangci-lint:
    name: golangci-lint
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version-file: 'go.mod'
          cache: true

      - uses: golangci/golangci-lint-action@v6
        with:
          version: v1.62            # pin a specific version for reproducibility
          args: --timeout=5m
```

### Recommended `.golangci.yml` Configuration

Place a `.golangci.yml` at the project root:

```yaml
run:
  timeout: 5m

linters:
  enable:
    - errcheck
    - gosimple
    - govet
    - ineffassign
    - staticcheck
    - unused
    - misspell
    - gofmt
    - goimports

linters-settings:
  misspell:
    locale: US
```

The `golangci/golangci-lint-action` automatically caches the lint binary and its
analysis cache. Do **not** also run `golangci-lint` inside the `build` job — keep
concerns separated.

---

## 3. Documentation Validation

Terraform providers use `tfplugindocs` to generate documentation from schema
annotations and example files. CI should verify that the checked-in docs match
what the generator would produce.

```yaml
# Inside .github/workflows/test.yml or a separate docs.yml
  docs:
    name: Documentation Validation
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version-file: 'go.mod'
          cache: true

      - run: go install github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs@latest

      # Validate existing docs for structural correctness
      - run: tfplugindocs validate

      # Regenerate docs and verify no diff
      - run: tfplugindocs generate
      - name: Verify no documentation drift
        run: |
          if [ -n "$(git diff --name-only)" ]; then
            echo "::error::Documentation is out of date. Run 'tfplugindocs generate' and commit the result."
            git diff
            exit 1
          fi
```

### What `tfplugindocs validate` Checks

- Every resource and data source declared by the provider has a corresponding
  Markdown file in `docs/`.
- Front-matter fields (`page_title`, `subcategory`, `description`) are present.
- Example blocks reference files that exist in `examples/`.
- No orphaned documentation files exist for resources/data-sources that no
  longer exist in the schema.

---

## 4. Generate Check — `make generate` Produces No Diff

If your Makefile includes a `generate` target (common for running `go generate`
and `tfplugindocs generate`), add a job that runs it and fails if any tracked
files change:

```yaml
  generate:
    name: Verify Generated Code
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version-file: 'go.mod'
          cache: true

      - run: go install github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs@latest
      - run: make generate

      - name: Verify no generated code drift
        run: |
          if [ -n "$(git status --porcelain)" ]; then
            echo "::error::Generated files are out of date. Run 'make generate' and commit the result."
            git status --porcelain
            git diff
            exit 1
          fi
```

A typical `Makefile` `generate` target:

```makefile
.PHONY: generate
generate:
	cd tools && go generate ./...
	go generate ./...
	tfplugindocs generate
```

---

## 5. Caching Go Modules and Build Cache

The `actions/setup-go@v5` action has **built-in caching** that is enabled by
default when `cache: true` (which is the default). It caches:

| Cache Path | Contents |
|---|---|
| `~/go/pkg/mod` | Downloaded module archives |
| `~/.cache/go-build` (Linux) | Compiled package objects |

If you need finer control (e.g., separate cache keys for acceptance tests), use
`actions/cache` directly:

```yaml
      - uses: actions/cache@v4
        with:
          path: |
            ~/go/pkg/mod
            ~/.cache/go-build
          key: ${{ runner.os }}-go-${{ hashFiles('**/go.sum') }}
          restore-keys: |
            ${{ runner.os }}-go-
```

> **Note:** `actions/setup-go@v5` with `cache: true` is sufficient for most
> providers. You only need a manual `actions/cache` step when the built-in
> heuristic does not suit your project layout.

---

## 6. HashiCorp's Reusable Release Workflow

HashiCorp publishes a reusable GitHub Actions workflow for releasing Terraform
providers:

```
hashicorp/ghaction-terraform-provider-release
```

This workflow:

1. Runs GoReleaser to build cross-platform binaries.
2. Signs the SHA256SUMS file with GPG.
3. Attaches the `terraform-registry-manifest.json` to the release.
4. Creates a GitHub Release with all required assets.

### Usage — `.github/workflows/release.yml`

```yaml
# .github/workflows/release.yml
# Trigger: creating a new tag matching v*
name: Release

on:
  push:
    tags:
      - 'v*'

permissions:
  contents: write

jobs:
  release:
    uses: hashicorp/ghaction-terraform-provider-release/.github/workflows/community.yml@v5
    secrets:
      gpg-private-key: ${{ secrets.GPG_PRIVATE_KEY }}
    with:
      setup-go-version-file: 'go.mod'
      goreleaser-release-args: ''          # extra args if needed
```

### Secrets You Must Configure

| Secret Name | Value |
|---|---|
| `GPG_PRIVATE_KEY` | ASCII-armored GPG private key (`gpg --armor --export-secret-keys KEY_ID`) |
| `GPG_FINGERPRINT` | (Sometimes required) The 40-character fingerprint of the signing key |

These are set in **Settings → Secrets and variables → Actions** in your GitHub
repository.

---

## 7. Complete Combined Workflow Example

For smaller providers it is common to keep everything in a single workflow file:

```yaml
name: CI

on:
  pull_request:
    branches: [main]
  push:
    branches: [main]

permissions:
  contents: read

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version-file: 'go.mod' }
      - run: go mod download
      - run: go build -v .
      - run: go test -v ./...

  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version-file: 'go.mod' }
      - uses: golangci/golangci-lint-action@v6
        with: { version: v1.62 }

  generate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version-file: 'go.mod' }
      - run: go install github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs@latest
      - run: make generate
      - run: |
          if [ -n "$(git status --porcelain)" ]; then
            echo "::error::Generated files are out of date."
            git diff
            exit 1
          fi

  testacc:
    runs-on: ubuntu-latest
    timeout-minutes: 15
    needs: [build]              # only run if build passes
    strategy:
      fail-fast: false
      matrix:
        terraform: ['1.8.*', '1.9.*']
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version-file: 'go.mod' }
      - uses: hashicorp/setup-terraform@v3
        with:
          terraform_version: ${{ matrix.terraform }}
          terraform_wrapper: false
      - run: go mod download
      - env: { TF_ACC: "1" }
        run: go test -v -cover ./internal/provider/ -timeout 120m
```

---

## 8. Best Practices Summary

1. **Pin action versions** — Use `@v4`, `@v5`, etc., not `@main`.
2. **Set `terraform_wrapper: false`** — Always, for acceptance tests.
3. **Use `fail-fast: false`** — Learn from all matrix legs, not just the first failure.
4. **Separate lint from test** — Faster feedback, cleaner logs.
5. **Check generated code drift** — Prevents "works on my machine" issues.
6. **Use `timeout-minutes`** — Prevent runaway acceptance tests from burning free minutes.
7. **Cache aggressively** — `actions/setup-go` built-in caching is usually enough.
8. **Keep secrets minimal** — Only the GPG private key is needed for signing.

---

## References

- [HashiCorp — Release & Publish a Provider](https://developer.hashicorp.com/terraform/tutorials/providers-plugin-framework/providers-plugin-framework-release-publish)
- [hashicorp/setup-terraform Action](https://github.com/hashicorp/setup-terraform)
- [hashicorp/ghaction-terraform-provider-release](https://github.com/hashicorp/ghaction-terraform-provider-release)
- [golangci/golangci-lint-action](https://github.com/golangci/golangci-lint-action)
- [actions/setup-go — Caching](https://github.com/actions/setup-go#caching-dependency-files-and-build-outputs)
- [terraform-plugin-docs](https://github.com/hashicorp/terraform-plugin-docs)
- [GitHub Actions — Using a matrix for your jobs](https://docs.github.com/en/actions/writing-workflows/choosing-what-your-workflow-does/running-variations-of-your-job-in-a-workflow)
