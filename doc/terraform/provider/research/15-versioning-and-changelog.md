# 15 — Versioning and Changelog

## Overview

Terraform providers follow Semantic Versioning (SemVer) for version constraints in
`required_providers`. This covers versioning, changelogs, deprecation, and releases.

**Primary references:**
- [HashiCorp Versioning Best Practices](https://developer.hashicorp.com/terraform/plugin/best-practices/versioning)
- [Plugin Framework Deprecations](https://developer.hashicorp.com/terraform/plugin/framework/deprecations)

---

## Semantic Versioning (SemVer)

Providers use **MAJOR.MINOR.PATCH** versioning as defined by [semver.org](https://semver.org):

### PATCH (e.g., 1.2.3 → 1.2.4)

Bug fixes only. No new features, no behavior changes.

- Fix a crash in a resource's Read function.
- Fix incorrect attribute value mapping.
- Fix documentation typos or errors.

### MINOR (e.g., 1.2.4 → 1.3.0)

New features and enhancements that are fully backward-compatible.

- Add a new resource or data source.
- Add new optional attributes to existing resources.
- Add new provider-defined functions.
- Announce deprecations (but do not remove anything).
- Performance improvements.

### MAJOR (e.g., 1.3.0 → 2.0.0)

Breaking changes that require user action.

- Remove a resource, data source, or attribute.
- Rename a resource, data source, or attribute.
- Change the type of an attribute (e.g., string → list).
- Change default values in a way that alters behavior.
- Remove support for an API version.
- Change required/optional status of an attribute.

### Version examples

```
v0.1.0   - Initial development, API may change at any time
v0.2.0   - Pre-1.0 minor release, still unstable
v1.0.0   - First stable release, SemVer contract begins
v1.1.0   - New comfyui_model resource added
v1.1.1   - Bug fix in comfyui_workflow Read
v1.2.0   - Deprecate old_attribute, add new_attribute
v2.0.0   - Remove old_attribute, require new_attribute
```

---

## Stability Expectations

Before v1.0.0, the API is unstable — breaking changes can happen in any minor release.
After v1.0.0, the SemVer contract is in effect: avoid breaking changes except in
major releases, group them (no more than once per year), provide upgrade guides, and
give at least one minor release cycle of deprecation warnings before removal.

---

## Changelog Best Practices

Maintain a `CHANGELOG.md` at the repository root. Record **all** user-facing
changes in every release.

### Changelog categories

Use these standardized categories in order:

1. **BREAKING CHANGES** — changes that require user action.
2. **FEATURES** — entirely new resources, data sources, or functions.
3. **ENHANCEMENTS** — new attributes, improved behavior, or performance.
4. **BUG FIXES** — corrections to existing behavior.
5. **DEPRECATIONS** — items marked for future removal.

### Entry format

Each entry should include the affected resource/data source name and a concise
description:

```markdown
## 1.3.0 (January 15, 2025)

FEATURES:

* **New Resource:** `comfyui_model` - Manage model files in ComfyUI ([#45](https://github.com/sbuglione/terraform-provider-comfyui/issues/45))
* **New Data Source:** `comfyui_nodes` - List all available ComfyUI nodes ([#48](https://github.com/sbuglione/terraform-provider-comfyui/issues/48))

ENHANCEMENTS:

* resource/comfyui_workflow: Add `tags` attribute ([#42](https://github.com/sbuglione/terraform-provider-comfyui/issues/42))

BUG FIXES:

* resource/comfyui_workflow: Fix panic when `api_json` contains null nodes ([#41](https://github.com/sbuglione/terraform-provider-comfyui/issues/41))

DEPRECATIONS:

* resource/comfyui_workflow: The `json` attribute is deprecated in favor of `api_json`. It will be removed in v2.0.0 ([#46](https://github.com/sbuglione/terraform-provider-comfyui/issues/46))
```

### Major release changelog

```markdown
## 2.0.0 (July 1, 2025)

BREAKING CHANGES:

* resource/comfyui_workflow: The `json` attribute removed. Use `api_json`. See [v2 Upgrade Guide](docs/guides/version-2-upgrade.md) ([#60])
* provider: The `endpoint` attribute renamed to `host` ([#62])

FEATURES:

* **New Resource:** `comfyui_workspace` ([#55])
```

---

## Deprecation Workflow

Deprecation is a multi-release process. Never remove something without
deprecating it first.

### Step 1: Deprecate in a MINOR release

Add `DeprecationMessage` to the attribute or resource schema:

```go
// Deprecating a schema attribute
"json": schema.StringAttribute{
    MarkdownDescription: "The workflow JSON. **Deprecated:** Use `api_json` instead.",
    Optional:            true,
    Computed:            true,
    DeprecationMessage:  "Use api_json instead. This attribute will be removed in v2.0.0.",
},
```

For deprecating an entire resource:

```go
resp.Schema = schema.Schema{
    DeprecationMessage: "Use comfyui_workflow instead. This resource will be removed in v2.0.0.",
    Attributes: map[string]schema.Attribute{
        // ... existing attributes ...
    },
}
```

### Step 2: Terraform warns users

When `DeprecationMessage` is set, Terraform automatically emits warnings during `plan` and `apply`:

```
Warning: Deprecated Attribute
  on main.tf line 5:
Use api_json instead. This attribute will be removed in v2.0.0.
```

No extra code is needed — the Plugin Framework handles the warning.

### Step 3: Remove in a MAJOR release

In the next major version, remove the deprecated item:

```go
// v2.0.0 — "json" attribute removed
Attributes: map[string]schema.Attribute{
    "id":       schema.StringAttribute{Computed: true},
    "api_json": schema.StringAttribute{Required: true},
    // "json" is gone
},
```

### Step 4: Document the migration

Provide an upgrade guide at `docs/guides/version-2-upgrade.md` showing users
how to migrate from the deprecated item to the replacement. Include before/after
HCL examples and note whether state migration is needed.

---

## Version Constraints

Users specify version constraints in `required_providers`:

```hcl
terraform {
  required_providers {
    comfyui = {
      source  = "sbuglione/comfyui"
      version = "~> 1.2"    # >= 1.2.0, < 2.0.0
    }
  }
}
```

### Common constraint operators

| Operator    | Meaning                          | Example           |
|-------------|----------------------------------|--------------------|
| `= 1.2.3`  | Exact version                    | Only 1.2.3         |
| `>= 1.2.3` | Greater than or equal            | 1.2.3+             |
| `~> 1.2`   | Pessimistic (minor)              | >= 1.2, < 2.0      |
| `~> 1.2.3` | Pessimistic (patch)              | >= 1.2.3, < 1.3    |

**Recommendation:** Use `~> MAJOR.MINOR` (e.g., `~> 1.2`) to get patches and
new features while avoiding breaking changes.

---

## Dependency Lock File

Terraform generates `.terraform.lock.hcl` to record the exact provider versions
and their checksums. This file should be committed to version control:

```hcl
# .terraform.lock.hcl (auto-generated, do not edit)
provider "registry.terraform.io/sbuglione/comfyui" {
  version     = "1.2.3"
  constraints = "~> 1.2"
  hashes = [
    "h1:abc123...",
    "zh:def456...",
  ]
}
```

Key points: commit `.terraform.lock.hcl` to version control for reproducible builds.
Run `terraform init -upgrade` to update within constraints.

---

## Git Tagging and Releases

The Terraform Registry discovers provider versions via Git tags. Tags must
follow the `vMAJOR.MINOR.PATCH` format.

### Creating a release

```bash
# Ensure CHANGELOG.md is updated
git add CHANGELOG.md
git commit -m "Update changelog for v1.3.0"

# Create an annotated tag
git tag -a v1.3.0 -m "Release v1.3.0"

# Push the tag
git push origin v1.3.0
```

### Embedding version in the provider binary

Pass the version at build time via `-ldflags`:

```go
// main.go
var version string = "dev"

func main() {
    opts := providerserver.ServeOpts{Address: "registry.terraform.io/sbuglione/comfyui"}
    err := providerserver.Serve(context.Background(), provider.New(version), opts)
    if err != nil { log.Fatal(err.Error()) }
}
```

```bash
go build -ldflags="-X main.version=1.3.0" -o terraform-provider-comfyui
```

### GoReleaser integration

Most providers use [GoReleaser](https://goreleaser.com/) for automated cross-platform
builds and Registry publishing. Key `.goreleaser.yml` settings:

```yaml
builds:
  - env: [CGO_ENABLED=0]
    flags: [-trimpath]
    ldflags: ['-s -w -X main.version={{ .Version }}']
    goos: [linux, darwin, windows]
    goarch: [amd64, arm64]
    binary: '{{ .ProjectName }}_v{{ .Version }}'
archives:
  - format: zip
    name_template: '{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}'
signs:
  - artifacts: checksum
    args: ["--batch", "--local-user", "{{ .Env.GPG_FINGERPRINT }}",
           "--output", "${signature}", "--detach-sign", "${artifact}"]
```

### GitHub Actions release workflow

```yaml
name: Release
on:
  push:
    tags: ['v*']
permissions:
  contents: write
jobs:
  release:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with: { fetch-depth: 0 }
      - uses: actions/setup-go@v5
        with: { go-version-file: 'go.mod' }
      - uses: crazy-max/ghaction-import-gpg@v6
        id: import_gpg
        with:
          gpg_private_key: ${{ secrets.GPG_PRIVATE_KEY }}
          passphrase: ${{ secrets.GPG_PASSPHRASE }}
      - uses: goreleaser/goreleaser-action@v6
        with: { args: "release --clean" }
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          GPG_FINGERPRINT: ${{ steps.import_gpg.outputs.fingerprint }}
```

---

## Version Lifecycle Summary

```
v0.x.y → Unstable   →   v1.0.0 → Stable (SemVer begins)
  →   v1.1.0 (features + deprecations)   →   v1.1.1 (bug fix)
  →   v2.0.0 (breaking changes, deprecated items removed, upgrade guide)
```

---

## References

- [HashiCorp Provider Versioning Best Practices](https://developer.hashicorp.com/terraform/plugin/best-practices/versioning)
- [Plugin Framework Deprecations](https://developer.hashicorp.com/terraform/plugin/framework/deprecations)
- [Semantic Versioning 2.0.0](https://semver.org/)
- [Terraform Version Constraints](https://developer.hashicorp.com/terraform/language/expressions/version-constraints)
- [GoReleaser](https://goreleaser.com/)
- [Terraform Registry Publishing](https://developer.hashicorp.com/terraform/registry/providers/publishing)
