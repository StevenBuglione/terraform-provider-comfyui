# Changelog

All notable changes to the Terraform Provider for ComfyUI will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## Versioning Policy

Provider versions follow the **ComfyUI compatibility line** model:

- Provider `0.18.x` is the compatibility line for ComfyUI `v0.18.5`
- The first release in this line is `v0.18.5`
- Later provider-only fixes are `v0.18.6`, `v0.18.7`, etc.
- The exact upstream pin remains authoritative in generated metadata and `comfyui_provider_info`
- Users should constrain the provider with `~> 0.18` for this compatibility line
- If the pinned upstream ComfyUI version changes materially, a new provider line is started

## [Unreleased]

## [0.18.7] - 2026-04-13

Provider-only patch release for the ComfyUI `v0.18.5` compatibility line.

### BUG FIXES

* **Partner Auth Ergonomics:** Add first-class provider support for `comfy_org` auth token / API key defaults and merge them into workflow `extra_data` automatically so partner-node execution no longer depends on hand-authored JSON payloads
* **Workflow Assembly Stability:** Re-register generated node state during resource `Read` so `comfyui_workflow.node_ids` survives refresh/apply cycles instead of failing with missing in-memory node registry state
* **Dynamic Validation Escape Hatch:** Add configurable `unsupported_dynamic_validation_mode = "error" | "warning" | "ignore"` so runtime-only dynamic node options such as `SaveVideo` can be used intentionally without forking the provider
* **Connection Normalization:** Normalize bare hosts, `host:port`, and full URLs consistently, including correct default port handling for `http` and `https`

### NOTES

* Maintains ComfyUI `v0.18.5` compatibility
* No provider schema breakage for existing `~> 0.18` users; new provider settings are additive

## [0.18.6] - 2026-04-12

Provider-only patch release for the ComfyUI `v0.18.5` compatibility line.

### BUG FIXES

* **Terraform Registry Packaging:** Include `terraform-provider-comfyui_<version>_manifest.json` in the generated `SHA256SUMS` bundle so Terraform Registry can ingest published releases successfully
* **Release Signing Reliability:** Preserve the GitHub Actions GPG import flow used by GoReleaser after correcting the private-key secret formatting

### NOTES

* Maintains ComfyUI `v0.18.5` compatibility
* No provider schema changes - safe upgrade within `version = "~> 0.18"`

## [0.18.5] - 2026-04-12

Initial release of the `0.18.x` provider compatibility line for ComfyUI `v0.18.5`.

### FEATURES

* **New Provider:** Terraform Provider for ComfyUI - Manage ComfyUI workflows with Terraform
* **Generated Node Resources:** `645` generated node resources from ComfyUI `v0.18.5`, providing typed Terraform resources for all built-in and partner ComfyUI nodes
* **New Resource:** `comfyui_workflow` - Queue and execute ComfyUI workflows with full state management
* **New Resource:** `comfyui_workflow_collection` - Group and manage multiple workflows together
* **New Resource:** `comfyui_workspace` - Compose multiple workflows into UI-oriented canvas exports
* **New Resource:** `comfyui_prompt_artifact` - Manage local prompt JSON artifacts with validation
* **New Resource:** `comfyui_workspace_artifact` - Manage local workspace JSON artifacts with validation
* **New Resource:** `comfyui_subgraph` - Define reusable workflow subgraphs as Terraform-managed artifacts
* **New Resource:** `comfyui_uploaded_image` - Upload and manage image inputs to ComfyUI
* **New Resource:** `comfyui_uploaded_mask` - Upload and manage mask inputs to ComfyUI
* **New Resource:** `comfyui_output_artifact` - Download and manage workflow outputs as local files
* **New Data Source:** `comfyui_provider_info` - Query provider version, ComfyUI version, and node count
* **New Data Source:** `comfyui_system_stats` - Retrieve ComfyUI server system statistics
* **New Data Source:** `comfyui_queue` - Query the current workflow execution queue
* **New Data Source:** `comfyui_workflow_history` - Retrieve workflow execution history
* **New Data Source:** `comfyui_inventory` - Query available models, checkpoints, LoRAs, and other runtime assets
* **New Data Source:** `comfyui_node_info` - Retrieve node information from ComfyUI object_info endpoint
* **New Data Source:** `comfyui_node_schema` - Access generated structured node schema contracts
* **New Data Source:** `comfyui_job` - Query individual workflow execution job status
* **New Data Source:** `comfyui_jobs` - Query multiple workflow execution jobs
* **New Data Source:** `comfyui_output` - Retrieve workflow output metadata and file paths
* **New Data Source:** `comfyui_subgraph_catalog` - List available subgraph definitions
* **New Data Source:** `comfyui_subgraph_definition` - Retrieve specific subgraph definition details
* **New Data Source:** `comfyui_prompt_json` - Translate Terraform workflow resources to executable prompt JSON
* **New Data Source:** `comfyui_workspace_json` - Translate Terraform workspace resources to ComfyUI workspace JSON
* **New Data Source:** `comfyui_prompt_to_workspace` - Convert prompt JSON to workspace JSON format
* **New Data Source:** `comfyui_workspace_to_prompt` - Convert workspace JSON to prompt JSON format
* **New Data Source:** `comfyui_prompt_to_terraform` - Synthesize Terraform HCL from prompt JSON artifacts
* **New Data Source:** `comfyui_workspace_to_terraform` - Synthesize Terraform HCL from workspace JSON artifacts
* **New Data Source:** `comfyui_prompt_validation` - Validate prompt JSON against ComfyUI runtime requirements
* **New Data Source:** `comfyui_workspace_validation` - Validate workspace JSON against ComfyUI runtime requirements

### ENHANCEMENTS

* **Code Generation Pipeline:** Automated extraction and generation of node resources from ComfyUI source using Python AST parsers and Go code generators
* **Plan-Time Validation:** Runtime-backed inventory validation during `terraform plan` for checkpoints, LoRAs, and other dynamic assets - fail early if required models are missing
* **Terraform Synthesis:** AI-harness-oriented data sources for translating native ComfyUI artifacts (prompt JSON, workspace JSON) into canonical Terraform HCL
* **Multi-Modal Support:** Support for image generation/editing, video generation, audio synthesis, text/LLM chat, and 3D model generation through partner API nodes
* **Validation Harness Reliability:** Hardened local E2E harnesses for ComfyUI startup, executable workflow validation, and workspace browser verification during release validation
* **Documentation:** Comprehensive provider documentation with 27 research documents covering all aspects of Terraform provider development
* **Testing:** Unit test coverage for code generator, HTTP client, and data sources; Python test suite for extraction pipeline
* **CI/CD:** GitHub Actions workflows for automated testing and GoReleaser-based releases with GPG signing

### NOTES

* Provider requires a reachable ComfyUI server (default: `localhost:8188`)
* Environment variables supported: `COMFYUI_HOST`, `COMFYUI_PORT`, `COMFYUI_API_KEY`
* Version constraint recommendation: `version = "~> 0.18"` for this compatibility line
* Built with Terraform Plugin Framework (not SDKv2)
* Generated node resources are virtual/plan-only - execution happens through `comfyui_workflow`

[Unreleased]: https://github.com/StevenBuglione/terraform-provider-comfyui/compare/v0.18.7...HEAD
[0.18.7]: https://github.com/StevenBuglione/terraform-provider-comfyui/releases/tag/v0.18.7
[0.18.6]: https://github.com/StevenBuglione/terraform-provider-comfyui/releases/tag/v0.18.6
[0.18.5]: https://github.com/StevenBuglione/terraform-provider-comfyui/releases/tag/v0.18.5
