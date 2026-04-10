# 18 — Terraform Registry Publishing

## Overview

The Terraform Registry is the primary distribution channel for Terraform
providers. When users write `terraform init`, Terraform downloads the provider
binary from the Registry. Publishing to the Registry requires a specific
repository structure, GPG-signed releases, a manifest file, and semantic
versioning.

This document covers the full publishing process for both the public Terraform
Registry and private registries (Terraform Cloud / Enterprise).

---

## 1. Terraform Registry — Public vs. Private

### Public Registry (`registry.terraform.io`)

- Free, open to any GitHub user or organization.
- Providers are referenced as `<namespace>/<name>`, e.g.,
  `sbuglione/comfyui`.
- Requires a **public** GitHub repository.
- Automatically detects new releases via GitHub webhooks.
- Anyone can browse and install the provider.

### Private Registry (Terraform Cloud / Enterprise)

- Requires a Terraform Cloud or Terraform Enterprise account.
- Providers are scoped to an organization.
- Supports private GitHub, GitLab, and Bitbucket repositories.
- Access is controlled by organization membership and team permissions.
- Useful for internal providers that should not be publicly available.

---

## 2. Requirements for Publishing to the Public Registry

All five requirements must be satisfied for the Registry to accept and serve
your provider:

### Requirement 1: Public GitHub Repository Named `terraform-provider-<name>`

The repository **must** follow this naming convention exactly. The `<name>`
portion becomes the provider's type name in Terraform configurations:

```
Repository:  github.com/sbuglione/terraform-provider-comfyui
Provider:    sbuglione/comfyui
Usage:       required_providers { comfyui = { source = "sbuglione/comfyui" } }
```

Rules:
- The repository must be **public**.
- The `<name>` must be lowercase and may contain hyphens.
- The repository must be owned by the same GitHub account (user or org) that
  signs in to the Registry.

### Requirement 2: Semantic Version Tags

Every release must be tagged with a [semantic version](https://semver.org/)
prefixed with `v`:

```bash
git tag v1.0.0
git push origin v1.0.0
```

Valid examples: `v0.1.0`, `v1.0.0`, `v2.3.1`, `v1.0.0-rc1`.

Invalid examples: `1.0.0` (no `v` prefix), `v1` (incomplete), `release-1.0`.

The Registry uses these tags to determine which versions are available. It
parses the semver components to display version ordering, detect pre-releases,
and determine the "latest" version.

### Requirement 3: GPG-Signed Releases

The release must include:
- A `_SHA256SUMS` file containing hashes of all archive files.
- A `_SHA256SUMS.sig` file containing a detached GPG signature of the checksums.

The corresponding GPG **public** key must be registered with the Terraform
Registry (see §4). Terraform CLI verifies the signature on every
`terraform init` to ensure release integrity.

### Requirement 4: `terraform-registry-manifest.json` in Release Assets

The manifest file tells the Registry which plugin protocol version(s) the
provider supports:

```json
{
  "version": 1,
  "metadata": {
    "protocol_versions": ["6.0"]
  }
}
```

This file is checked into the repository root and attached to the GitHub
Release by GoReleaser (see document 17).

#### Protocol Version Reference

| Protocol | Framework | Notes |
|---|---|---|
| `5.0` | SDKv2 (`terraform-plugin-sdk/v2`) | Legacy framework |
| `6.0` | Plugin Framework (`terraform-plugin-framework`) | Current recommended framework |
| `5.0` and `6.0` | Mux (both frameworks) | For providers migrating from SDKv2 |

If your provider uses **only** the Plugin Framework, declare `["6.0"]`.

If your provider uses the mux approach (mixing SDKv2 and Plugin Framework
resources during a migration), declare `["5.0", "6.0"]`.

### Requirement 5: Properly Structured Release Assets

A valid release contains these files (example for version 1.0.0):

```
terraform-provider-comfyui_1.0.0_darwin_amd64.zip
terraform-provider-comfyui_1.0.0_darwin_arm64.zip
terraform-provider-comfyui_1.0.0_freebsd_386.zip
terraform-provider-comfyui_1.0.0_freebsd_amd64.zip
terraform-provider-comfyui_1.0.0_freebsd_arm.zip
terraform-provider-comfyui_1.0.0_freebsd_arm64.zip
terraform-provider-comfyui_1.0.0_linux_386.zip
terraform-provider-comfyui_1.0.0_linux_amd64.zip
terraform-provider-comfyui_1.0.0_linux_arm.zip
terraform-provider-comfyui_1.0.0_linux_arm64.zip
terraform-provider-comfyui_1.0.0_windows_386.zip
terraform-provider-comfyui_1.0.0_windows_amd64.zip
terraform-provider-comfyui_1.0.0_windows_arm.zip
terraform-provider-comfyui_1.0.0_windows_arm64.zip
terraform-provider-comfyui_1.0.0_SHA256SUMS
terraform-provider-comfyui_1.0.0_SHA256SUMS.sig
terraform-provider-comfyui_1.0.0_manifest.json
```

The naming convention is strict:
```
<project>_<version>_<os>_<arch>.zip
```

GoReleaser produces all of these automatically when configured as described in
document 17.

---

## 3. The `terraform-registry-manifest.json` File

### File Contents

```json
{
  "version": 1,
  "metadata": {
    "protocol_versions": ["6.0"]
  }
}
```

### Where It Lives

- **In the repository:** at the project root, checked into version control.
- **In the release:** attached as `<project>_<version>_manifest.json` by
  GoReleaser's `release.extra_files` configuration.

### What Each Field Means

| Field | Type | Description |
|---|---|---|
| `version` | integer | Manifest schema version. Always `1`. |
| `metadata.protocol_versions` | array of strings | Which Terraform plugin protocols this provider supports. |

### Common Configurations

**Plugin Framework only (protocol 6):**

```json
{
  "version": 1,
  "metadata": {
    "protocol_versions": ["6.0"]
  }
}
```

**SDKv2 only (protocol 5):**

```json
{
  "version": 1,
  "metadata": {
    "protocol_versions": ["5.0"]
  }
}
```

**Mux — both frameworks during migration:**

```json
{
  "version": 1,
  "metadata": {
    "protocol_versions": ["5.0", "6.0"]
  }
}
```

---

## 4. GPG Key Setup

### Step 1: Generate a Key Pair

```bash
gpg --full-generate-key
```

When prompted:
- **Key type:** RSA and RSA (default)
- **Key size:** 4096
- **Expiration:** 0 (no expiration) or a long duration (e.g., 5y)
- **Real name:** Your Name or Organization Name
- **Email:** your-email@example.com
- **Passphrase:** Use a strong passphrase (or none, for CI convenience)

### Step 2: Find the Key ID

```bash
gpg --list-secret-keys --keyid-format=long
```

Output example:

```
/home/user/.gnupg/secring.gpg
-----------------------------
sec   rsa4096/ABCDEF1234567890 2024-01-01 [SC]
      Key fingerprint = 1234 5678 90AB CDEF 1234  5678 90AB CDEF 1234 5678
uid                 [ultimate] Your Name <your-email@example.com>
ssb   rsa4096/0987654321FEDCBA 2024-01-01 [E]
```

The key ID is `ABCDEF1234567890`. The full fingerprint is
`1234567890ABCDEF1234567890ABCDEF12345678`.

### Step 3: Export the Public Key

```bash
gpg --armor --export ABCDEF1234567890
```

This outputs the ASCII-armored public key block:

```
-----BEGIN PGP PUBLIC KEY BLOCK-----
mQINBGWY...
...
-----END PGP PUBLIC KEY BLOCK-----
```

Copy this entire block — you will paste it into the Terraform Registry.

### Step 4: Export the Private Key (for CI)

```bash
gpg --armor --export-secret-keys ABCDEF1234567890
```

Store the output as a GitHub Actions secret named `GPG_PRIVATE_KEY`:

1. Go to your GitHub repository → **Settings** → **Secrets and variables** →
   **Actions**.
2. Click **New repository secret**.
3. Name: `GPG_PRIVATE_KEY`
4. Value: Paste the entire ASCII-armored private key block.

If your key has a passphrase, also store it as `GPG_PASSPHRASE`.

### Step 5: Upload the Public Key to the Terraform Registry

1. Go to [registry.terraform.io](https://registry.terraform.io).
2. Sign in with your GitHub account.
3. Navigate to **User Settings** → **Signing Keys** (or visit
   `https://registry.terraform.io/settings/gpg-keys`).
4. Click **Add a GPG Key**.
5. Paste the ASCII-armored public key.
6. Click **Add GPG Key**.

The Registry uses this key to verify the `_SHA256SUMS.sig` file in every
release.

---

## 5. Publishing Workflow — Step by Step

### Step 1: Sign In to the Terraform Registry

Visit [registry.terraform.io](https://registry.terraform.io) and click **Sign
in** → **Sign in with GitHub**. Authorize the Terraform Registry GitHub App.

### Step 2: Add Your GPG Public Key

Follow the process in §4, Step 5. This only needs to be done once per key.

### Step 3: Register Your Provider Repository

1. Click **Publish** → **Provider** in the Registry UI.
2. Select your GitHub account or organization.
3. Select the `terraform-provider-comfyui` repository.
4. Agree to the terms of service.
5. Click **Publish Provider**.

The Registry creates a webhook on your repository. From this point forward, any
GitHub Release with a valid semver tag triggers the Registry to index the new
version.

### Step 4: Create a Release

Tag and push:

```bash
git tag v0.1.0
git push origin v0.1.0
```

This triggers the release GitHub Actions workflow (see below), which runs
GoReleaser to build, sign, and publish the release.

### Step 5: Registry Detects the Release

The GitHub webhook notifies the Registry of the new release. The Registry:

1. Downloads the `_manifest.json` to determine protocol versions.
2. Downloads the `_SHA256SUMS` and `_SHA256SUMS.sig` files.
3. Verifies the signature against your registered GPG public key.
4. Indexes all platform-specific ZIP files.
5. Makes the new version available for `terraform init`.

This process typically completes within a few minutes.

---

## 6. HashiCorp's Reusable Release Action

HashiCorp publishes a reusable GitHub Actions workflow specifically for
Terraform provider releases:

```
hashicorp/ghaction-terraform-provider-release
```

This action encapsulates the entire release process: GoReleaser execution, GPG
import, signing, and manifest attachment.

### Complete Release Workflow Using the Reusable Action

```yaml
# .github/workflows/release.yml
name: Release

on:
  push:
    tags:
      - 'v*'

# Require write permissions to create the GitHub Release
permissions:
  contents: write

jobs:
  # ------------------------------------------------------------------
  # Optional: Run tests before releasing
  # ------------------------------------------------------------------
  test:
    name: Pre-release Tests
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version-file: 'go.mod'
      - run: go mod download
      - run: go build -v .
      - run: go test -v ./...

  # ------------------------------------------------------------------
  # Release: uses HashiCorp's reusable workflow
  # ------------------------------------------------------------------
  release:
    name: Release
    needs: [test]
    uses: hashicorp/ghaction-terraform-provider-release/.github/workflows/community.yml@v5
    secrets:
      gpg-private-key: ${{ secrets.GPG_PRIVATE_KEY }}
    with:
      setup-go-version-file: 'go.mod'
```

### What the Reusable Workflow Does Internally

1. Checks out the repository with full Git history (`fetch-depth: 0`).
2. Sets up Go using the version from `go.mod`.
3. Imports the GPG private key from the secret.
4. Runs GoReleaser with the project's `.goreleaser.yml`.
5. Creates a GitHub Release with all assets attached.

### Secrets Required

| Secret | Description |
|---|---|
| `gpg-private-key` | ASCII-armored GPG private key |

### Optional Inputs

| Input | Default | Description |
|---|---|---|
| `setup-go-version-file` | `'go.mod'` | File to read Go version from |
| `goreleaser-release-args` | `''` | Extra arguments passed to GoReleaser |

---

## 7. Custom Release Workflow (Without Reusable Action)

If you need more control over the release process, here is a standalone
workflow:

```yaml
# .github/workflows/release.yml
name: Release

on:
  push:
    tags:
      - 'v*'

permissions:
  contents: write

jobs:
  goreleaser:
    name: GoReleaser Release
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
        uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version-file: 'go.mod'

      - name: Import GPG key
        id: import_gpg
        uses: crazy-max/ghaction-import-gpg@v6
        with:
          gpg_private_key: ${{ secrets.GPG_PRIVATE_KEY }}

      - name: Run GoReleaser
        uses: goreleaser/goreleaser-action@v6
        with:
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          GPG_FINGERPRINT: ${{ steps.import_gpg.outputs.fingerprint }}
```

### Key Differences from the Reusable Workflow

- You manage each step explicitly.
- You can insert additional steps (e.g., running tests, building documentation,
  sending notifications).
- You are responsible for keeping action versions up to date.

---

## 8. Private Registry Publishing (Terraform Cloud / Enterprise)

### Terraform Cloud Private Registry

Terraform Cloud organizations include a private registry that can host
providers visible only to organization members.

#### Setup

1. In Terraform Cloud, go to **Registry** → **Providers** → **Publish** →
   **Provider**.
2. Connect your VCS provider (GitHub, GitLab, Bitbucket, Azure DevOps).
3. Select the `terraform-provider-<name>` repository.
4. Add your GPG public key to the organization's registry settings.

#### Publishing

The process is identical to the public registry:

1. Tag a release.
2. CI builds and signs the artifacts.
3. Create a GitHub Release.
4. Terraform Cloud detects the release via VCS webhook.

#### Usage in Terraform Configuration

```hcl
terraform {
  required_providers {
    comfyui = {
      source  = "app.terraform.io/<org>/comfyui"
      version = "~> 1.0"
    }
  }
}
```

The `app.terraform.io/<org>` prefix routes the download through the private
registry instead of the public one.

### Terraform Enterprise

Terraform Enterprise (self-hosted) supports the same private registry
functionality. The hostname changes to your Enterprise installation:

```hcl
terraform {
  required_providers {
    comfyui = {
      source  = "tfe.example.com/<org>/comfyui"
      version = "~> 1.0"
    }
  }
}
```

### API-Based Publishing

Both Terraform Cloud and Enterprise support publishing providers via API,
without a VCS connection:

```bash
# Create provider version
curl -s \
  --header "Authorization: Bearer $TFC_TOKEN" \
  --header "Content-Type: application/vnd.api+json" \
  --request POST \
  "https://app.terraform.io/api/v2/organizations/$ORG/registry-providers/private/$ORG/comfyui/versions" \
  --data '{
    "data": {
      "type": "registry-provider-versions",
      "attributes": {
        "version": "1.0.0",
        "key-id": "<gpg-key-id>",
        "protocols": ["6.0"]
      }
    }
  }'
```

After creating the version, upload each platform binary, the checksums, and
the signature via the upload URLs returned in the API response.

---

## 9. Versioning Best Practices

### Semantic Versioning Rules

```
MAJOR.MINOR.PATCH

MAJOR — Breaking changes (removed resources, changed attribute types)
MINOR — New features (new resources, data sources, attributes)
PATCH — Bug fixes, documentation updates
```

### Pre-release Versions

Use pre-release tags for testing before a stable release:

```bash
git tag v1.0.0-rc1
git tag v1.0.0-beta1
git tag v1.0.0-alpha1
```

Terraform treats pre-release versions specially:
- They are not installed by default.
- Users must specify the exact pre-release version in `required_providers`.
- They do not satisfy version constraints like `~> 1.0`.

### Version Constraints in Terraform

```hcl
required_providers {
  comfyui = {
    source  = "sbuglione/comfyui"
    version = "~> 1.0"       # >= 1.0.0, < 2.0.0
  }
}
```

Other constraint operators:

| Constraint | Meaning |
|---|---|
| `= 1.0.0` | Exactly version 1.0.0 |
| `!= 1.0.0` | Any version except 1.0.0 |
| `> 1.0.0` | Greater than 1.0.0 |
| `>= 1.0.0` | Greater than or equal to 1.0.0 |
| `< 2.0.0` | Less than 2.0.0 |
| `~> 1.0` | Pessimistic (>= 1.0.0, < 2.0.0) |
| `~> 1.0.0` | Pessimistic (>= 1.0.0, < 1.1.0) |
| `>= 1.0, < 1.5` | Combined constraints |

---

## 10. End-to-End Checklist

Use this checklist before your first publish:

```
[ ] Repository is public and named terraform-provider-<name>
[ ] go.mod declares the correct module path
[ ] main.go compiles and passes go vet
[ ] .goreleaser.yml is present and valid (goreleaser check)
[ ] terraform-registry-manifest.json is at the project root
[ ] GPG key pair generated (RSA 4096)
[ ] GPG public key uploaded to Terraform Registry
[ ] GPG private key stored as GitHub Actions secret (GPG_PRIVATE_KEY)
[ ] .github/workflows/release.yml is configured
[ ] CI tests pass on main branch
[ ] First tag pushed: git tag v0.1.0 && git push origin v0.1.0
[ ] GitHub Release created with all expected assets
[ ] Terraform Registry shows the provider and version
[ ] terraform init successfully downloads the provider
```

---

## 11. Troubleshooting

| Symptom | Cause | Fix |
|---|---|---|
| Registry says "No versions published" | No GitHub Release with valid assets | Create a release with a `v*` tag and all required assets |
| `signature verification failed` | Wrong GPG key or unsigned release | Verify the public key in Registry matches the signing key |
| `provider not found` | Repository not registered with Registry | Go to registry.terraform.io and publish the provider |
| `incompatible protocol version` | Manifest declares wrong protocol | Update `terraform-registry-manifest.json` |
| Pre-release not installable | User did not pin exact version | Use `version = "1.0.0-rc1"` (exact match) |
| `terraform init` hangs | Network or rate limiting | Check GitHub API rate limits; retry |
| Release assets missing | GoReleaser misconfigured | Run `goreleaser release --snapshot --clean` locally to verify |

---

## References

- [Terraform Registry — Publishing Providers](https://developer.hashicorp.com/terraform/registry/providers/publishing)
- [Terraform Registry — Provider Documentation](https://developer.hashicorp.com/terraform/registry/providers/docs)
- [HashiCorp — Release & Publish Tutorial](https://developer.hashicorp.com/terraform/tutorials/providers-plugin-framework/providers-plugin-framework-release-publish)
- [hashicorp/ghaction-terraform-provider-release](https://github.com/hashicorp/ghaction-terraform-provider-release)
- [Terraform Cloud — Private Registry](https://developer.hashicorp.com/terraform/cloud-docs/registry)
- [Terraform CLI — Provider Installation](https://developer.hashicorp.com/terraform/cli/config/config-file#provider-installation)
- [Semantic Versioning 2.0.0](https://semver.org/)
- [GPG — GNU Privacy Guard](https://gnupg.org/documentation/)
