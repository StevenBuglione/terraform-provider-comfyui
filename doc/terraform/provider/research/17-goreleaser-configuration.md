# 17 — GoReleaser Configuration for Terraform Providers

## Overview

GoReleaser is the standard build-and-release tool for Terraform providers. It
compiles the provider binary for every supported OS/architecture combination,
packages the binaries into ZIP archives, generates SHA256 checksums, signs the
checksum file with GPG, and creates a GitHub Release with all assets attached.

The Terraform Registry **requires** a specific release asset structure. GoReleaser
is configured via a `.goreleaser.yml` file at the project root to produce
exactly the right output.

---

## 1. What GoReleaser Does

When you run GoReleaser (either locally or in CI), it performs these steps in
order:

1. **Before hooks** — Runs commands like `go mod tidy` to ensure a clean state.
2. **Build** — Cross-compiles the Go binary for every target in the `goos` ×
   `goarch` matrix, applying the specified flags and ldflags.
3. **Archive** — Packages each binary into a ZIP file with a predictable name.
4. **Checksum** — Computes SHA256 hashes of every archive and writes them to a
   single `_SHA256SUMS` file.
5. **Sign** — GPG-signs the checksum file, producing a `_SHA256SUMS.sig` file.
6. **Release** — Creates a GitHub Release, attaches all archives, the checksum
   file, its signature, and any extra files (like the registry manifest).
7. **Changelog** — Auto-generates a changelog from Git commit messages.

---

## 2. Complete `.goreleaser.yml`

```yaml
# .goreleaser.yml
version: 2

before:
  hooks:
    - go mod tidy

builds:
  - env:
      - CGO_ENABLED=0
    mod_timestamp: '{{ .CommitTimestamp }}'
    flags:
      - -trimpath
    ldflags:
      - '-s -w -X main.version={{.Version}} -X main.commit={{.Commit}}'
    goos:
      - freebsd
      - windows
      - linux
      - darwin
    goarch:
      - amd64
      - '386'
      - arm
      - arm64
    ignore:
      - goos: darwin
        goarch: '386'
    binary: '{{ .ProjectName }}_v{{ .Version }}'

archives:
  - format: zip
    name_template: '{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}'

checksum:
  name_template: '{{ .ProjectName }}_{{ .Version }}_SHA256SUMS'
  algorithm: sha256

signs:
  - artifacts: checksum
    args:
      - "--batch"
      - "--local-user"
      - "{{ .Env.GPG_FINGERPRINT }}"
      - "--output"
      - "${signature}"
      - "--detach-sign"
      - "${artifact}"

release:
  extra_files:
    - glob: 'terraform-registry-manifest.json'
      name_template: '{{ .ProjectName }}_{{ .Version }}_manifest.json'

changelog:
  sort: asc
  filters:
    exclude:
      - "^docs:"
      - "^test:"
```

---

## 3. Section-by-Section Explanation

### 3.1 `version`

```yaml
version: 2
```

Declares the GoReleaser configuration schema version. Version 2 is the current
format and is required by GoReleaser v2.x. If you omit this or set it to `1`,
GoReleaser v2 will reject the configuration.

### 3.2 `before.hooks`

```yaml
before:
  hooks:
    - go mod tidy
```

Commands that run **before** any build step. `go mod tidy` ensures `go.mod` and
`go.sum` are clean and consistent. If they are not, the build would still
succeed, but the resulting binary might carry unnecessary module references.
This is a safety net.

You can add additional hooks here:

```yaml
before:
  hooks:
    - go mod tidy
    - go generate ./...
    - go vet ./...
```

### 3.3 `builds`

```yaml
builds:
  - env:
      - CGO_ENABLED=0
    mod_timestamp: '{{ .CommitTimestamp }}'
    flags:
      - -trimpath
    ldflags:
      - '-s -w -X main.version={{.Version}} -X main.commit={{.Commit}}'
    goos:
      - freebsd
      - windows
      - linux
      - darwin
    goarch:
      - amd64
      - '386'
      - arm
      - arm64
    ignore:
      - goos: darwin
        goarch: '386'
    binary: '{{ .ProjectName }}_v{{ .Version }}'
```

| Field | Purpose |
|---|---|
| `CGO_ENABLED=0` | Produces a fully static binary with no C library dependency. Essential for cross-compilation and for running in minimal containers. |
| `mod_timestamp` | Sets the modification timestamp of every file in the binary to the Git commit timestamp. This improves reproducibility — two builds of the same commit produce bit-identical binaries. |
| `-trimpath` | Strips the local filesystem path from the binary. Without this, the binary contains your build machine's absolute paths in stack traces, which is a minor information leak. |
| `-s -w` | Strips the symbol table (`-s`) and DWARF debug information (`-w`). Reduces binary size by ~30%. Debug information is not needed for production Terraform providers. |
| `-X main.version={{.Version}}` | Injects the release version into the `main.version` variable at link time. This allows `terraform-provider-comfyui --version` to print the correct version. |
| `-X main.commit={{.Commit}}` | Injects the Git commit SHA for traceability. |
| `goos` / `goarch` | The cross-compilation matrix. See §4 for details. |
| `ignore` | Excludes impossible or unsupported combinations. macOS dropped 32-bit support years ago, so `darwin/386` is excluded. |
| `binary` | The output binary name. Terraform providers **must** follow the naming convention `terraform-provider-<name>_v<version>`. The `{{ .ProjectName }}` template resolves to the repository name (e.g., `terraform-provider-comfyui`). |

### 3.4 `archives`

```yaml
archives:
  - format: zip
    name_template: '{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}'
```

Each binary is packaged into a ZIP archive. The Terraform Registry expects ZIP
format specifically — not `.tar.gz`.

The `name_template` produces filenames like:

```
terraform-provider-comfyui_1.0.0_linux_amd64.zip
terraform-provider-comfyui_1.0.0_darwin_arm64.zip
terraform-provider-comfyui_1.0.0_windows_amd64.zip
```

These names must follow this exact pattern for the Registry to correctly parse
the OS and architecture from the filename.

### 3.5 `checksum`

```yaml
checksum:
  name_template: '{{ .ProjectName }}_{{ .Version }}_SHA256SUMS'
  algorithm: sha256
```

Produces a single file containing SHA256 hashes of every archive:

```
a1b2c3d4...  terraform-provider-comfyui_1.0.0_linux_amd64.zip
e5f6a7b8...  terraform-provider-comfyui_1.0.0_darwin_arm64.zip
...
```

The Terraform CLI downloads this file and verifies the integrity of the
provider binary after download. SHA256 is the only algorithm the Registry
supports.

### 3.6 `signs`

```yaml
signs:
  - artifacts: checksum
    args:
      - "--batch"
      - "--local-user"
      - "{{ .Env.GPG_FINGERPRINT }}"
      - "--output"
      - "${signature}"
      - "--detach-sign"
      - "${artifact}"
```

Signs the SHA256SUMS file with GPG. The result is a detached signature file:

```
terraform-provider-comfyui_1.0.0_SHA256SUMS.sig
```

| Argument | Purpose |
|---|---|
| `--batch` | Non-interactive mode. Essential for CI where there is no TTY. |
| `--local-user` | Specifies which GPG key to sign with. The `GPG_FINGERPRINT` environment variable is set in CI from a GitHub Actions secret. |
| `--output ${signature}` | GoReleaser fills in `${signature}` with the expected output path. |
| `--detach-sign ${artifact}` | Creates a detached signature (`.sig`) rather than wrapping the artifact. |

The Terraform Registry uses the signature to verify that the release was
produced by the provider author. The corresponding **public** key must be
uploaded to the Registry.

### 3.7 `release`

```yaml
release:
  extra_files:
    - glob: 'terraform-registry-manifest.json'
      name_template: '{{ .ProjectName }}_{{ .Version }}_manifest.json'
```

Attaches additional files to the GitHub Release. The
`terraform-registry-manifest.json` file tells the Registry which Terraform
plugin protocol version(s) this provider supports. Without this file, the
Registry cannot serve the provider.

The `name_template` renames the file to include the project name and version,
matching the convention the Registry expects:

```
terraform-provider-comfyui_1.0.0_manifest.json
```

### 3.8 `changelog`

```yaml
changelog:
  sort: asc
  filters:
    exclude:
      - "^docs:"
      - "^test:"
```

Auto-generates a changelog from Git commit messages between the previous tag
and the current tag. Commits matching the exclude patterns are omitted —
documentation and test changes are typically noise in a release changelog.

The changelog appears in the GitHub Release body and is visible to users
browsing the release page and the Terraform Registry.

---

## 4. Cross-Compilation Targets

### Supported Combinations

The `goos` × `goarch` matrix in the configuration above produces these targets:

| OS | amd64 | 386 | arm | arm64 |
|---|---|---|---|---|
| **linux** | ✅ | ✅ | ✅ | ✅ |
| **darwin** | ✅ | ❌ (excluded) | — | ✅ |
| **windows** | ✅ | ✅ | ✅ | ✅ |
| **freebsd** | ✅ | ✅ | ✅ | ✅ |

### Why These Targets?

- **linux/amd64** — The most common CI and server platform.
- **linux/arm64** — AWS Graviton, Apple Silicon VMs, Raspberry Pi 4+.
- **darwin/amd64** — Intel Macs (still widely used).
- **darwin/arm64** — Apple Silicon (M1/M2/M3/M4).
- **windows/amd64** — Windows workstations.
- **freebsd** — FreeBSD is a supported Terraform platform.
- **386/arm** — Embedded systems, older hardware, 32-bit environments.
- **darwin/386** — Excluded because macOS has not supported 32-bit since
  Catalina (10.15).

### Adding or Removing Targets

To add a new target (e.g., `linux/riscv64`):

```yaml
goarch:
  - amd64
  - '386'
  - arm
  - arm64
  - riscv64           # add the new architecture
```

To exclude a specific combination:

```yaml
ignore:
  - goos: darwin
    goarch: '386'
  - goos: windows     # example: drop Windows ARM
    goarch: arm
```

---

## 5. GPG Signing Configuration

### Local Setup

Generate a GPG key pair dedicated to provider signing:

```bash
gpg --full-generate-key
# Choose: RSA and RSA, 4096 bits, no expiration (or set a long expiration)
# Name: Your Name (Terraform Provider Signing)
# Email: your-email@example.com
```

List keys to find the fingerprint:

```bash
gpg --list-secret-keys --keyid-format=long
```

Output:

```
sec   rsa4096/ABCDEF1234567890 2024-01-01 [SC]
      1234567890ABCDEF1234567890ABCDEF12345678
uid                 [ultimate] Your Name <your-email@example.com>
```

The 40-character hex string is the **fingerprint**. The shorter `ABCDEF1234567890`
is the **key ID**.

### Exporting for CI

Export the ASCII-armored private key:

```bash
gpg --armor --export-secret-keys ABCDEF1234567890 > private-key.asc
```

Store the contents of `private-key.asc` as the GitHub Actions secret
`GPG_PRIVATE_KEY`. Then delete the file:

```bash
rm private-key.asc
```

Export the ASCII-armored public key for the Terraform Registry:

```bash
gpg --armor --export ABCDEF1234567890 > public-key.asc
```

Upload the contents of `public-key.asc` to the Terraform Registry (see
document 18 for details).

### CI Key Import

The HashiCorp reusable release action handles key import automatically. If you
are writing a custom release workflow, import the key like this:

```yaml
      - name: Import GPG key
        id: import_gpg
        uses: crazy-max/ghaction-import-gpg@v6
        with:
          gpg_private_key: ${{ secrets.GPG_PRIVATE_KEY }}
          passphrase: ${{ secrets.GPG_PASSPHRASE }}   # if the key has one

      - name: Run GoReleaser
        uses: goreleaser/goreleaser-action@v6
        with:
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          GPG_FINGERPRINT: ${{ steps.import_gpg.outputs.fingerprint }}
```

---

## 6. The Manifest File

Create `terraform-registry-manifest.json` at the project root:

```json
{
  "version": 1,
  "metadata": {
    "protocol_versions": ["6.0"]
  }
}
```

This file is **checked into the repository** and attached to every release by
GoReleaser via the `release.extra_files` configuration.

- `"version": 1` — The manifest schema version (always `1`).
- `"protocol_versions": ["6.0"]` — Declares which Terraform plugin protocol
  versions this provider supports. Protocol 6 corresponds to the Plugin
  Framework. If you also support protocol 5 (SDKv2), list both: `["5.0", "6.0"]`.

---

## 7. Testing GoReleaser Locally

Before pushing a tag and triggering a real release, validate the configuration
locally:

### Snapshot Build (No Publish)

```bash
goreleaser release --snapshot --clean
```

- `--snapshot` — Skips tag validation and does not publish to GitHub. Uses a
  pseudo-version like `0.0.0-SNAPSHOT-abc1234`.
- `--clean` — Removes the `dist/` directory before building.

The output lands in `./dist/`:

```
dist/
├── terraform-provider-comfyui_0.0.0-SNAPSHOT_linux_amd64.zip
├── terraform-provider-comfyui_0.0.0-SNAPSHOT_darwin_arm64.zip
├── terraform-provider-comfyui_0.0.0-SNAPSHOT_SHA256SUMS
├── terraform-provider-comfyui_0.0.0-SNAPSHOT_manifest.json
└── ...
```

### Validate Configuration Only

```bash
goreleaser check
```

Parses `.goreleaser.yml` and reports any syntax or semantic errors without
building anything.

### Skip Signing (For Local Testing)

If you do not have the GPG key on your local machine, skip the sign step:

```bash
goreleaser release --snapshot --clean --skip=sign
```

### Full Local Dry Run

If you do have the GPG key available locally:

```bash
export GPG_FINGERPRINT="1234567890ABCDEF1234567890ABCDEF12345678"
goreleaser release --snapshot --clean
```

Verify the `.sig` file is produced and the checksums are correct:

```bash
cd dist
sha256sum -c terraform-provider-comfyui_0.0.0-SNAPSHOT_SHA256SUMS
```

---

## 8. Integrating GoReleaser with GitHub Actions

If you are **not** using HashiCorp's reusable workflow, here is a standalone
release workflow:

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
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0       # GoReleaser needs full history for changelog

      - uses: actions/setup-go@v5
        with:
          go-version-file: 'go.mod'

      - name: Import GPG key
        id: import_gpg
        uses: crazy-max/ghaction-import-gpg@v6
        with:
          gpg_private_key: ${{ secrets.GPG_PRIVATE_KEY }}

      - uses: goreleaser/goreleaser-action@v6
        with:
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          GPG_FINGERPRINT: ${{ steps.import_gpg.outputs.fingerprint }}
```

> **Important:** `fetch-depth: 0` is required so GoReleaser can read the full
> Git history to generate the changelog.

---

## 9. Troubleshooting

| Problem | Cause | Fix |
|---|---|---|
| `signing failed: no secret key` | GPG key not imported or wrong fingerprint | Verify `GPG_FINGERPRINT` matches the imported key |
| `tag v1.0.0 was not made against commit ...` | Tag points to wrong commit | Delete and recreate the tag at the correct commit |
| `could not open dist/` | Stale `dist/` from a previous run | Use `--clean` flag |
| Archives are `.tar.gz` instead of `.zip` | Missing `format: zip` in `archives` | Add `format: zip` explicitly |
| Registry rejects release | Missing manifest file or wrong naming | Verify `release.extra_files` and `name_template` |

---

## References

- [GoReleaser Documentation — Customization](https://goreleaser.com/customization/)
- [GoReleaser — Builds](https://goreleaser.com/customization/builds/)
- [GoReleaser — Signing](https://goreleaser.com/customization/sign/)
- [GoReleaser — Archives](https://goreleaser.com/customization/archive/)
- [GoReleaser — Changelog](https://goreleaser.com/customization/changelog/)
- [HashiCorp — Publishing Providers](https://developer.hashicorp.com/terraform/registry/providers/publishing)
- [crazy-max/ghaction-import-gpg](https://github.com/crazy-max/ghaction-import-gpg)
- [goreleaser/goreleaser-action](https://github.com/goreleaser/goreleaser-action)
