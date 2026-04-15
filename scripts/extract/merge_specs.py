#!/usr/bin/env python3
"""
Merge V1 and V3 node specifications into a single node_specs.json file.

Usage:
    python3 scripts/extract/merge_specs.py third_party/ComfyUI

This runs both extractors and merges their output into scripts/extract/node_specs.json.
"""

import json
import os
import re
import subprocess
import sys
from datetime import datetime, timezone

VALIDATION_KIND_STATIC_ENUM = "static_enum"
VALIDATION_KIND_DYNAMIC_INVENTORY = "dynamic_inventory"
VALIDATION_KIND_DYNAMIC_EXPRESSION = "dynamic_expression"
VALIDATION_KIND_FREEFORM = "freeform"

INVENTORY_SOURCE_RE = re.compile(r"folder_paths\.get_filename_list\('([^']+)'\)")


def to_snake_case(name: str) -> str:
    """Convert CamelCase/PascalCase to snake_case."""
    s1 = re.sub(r'([A-Z]+)([A-Z][a-z])', r'\1_\2', name)
    s2 = re.sub(r'([a-z\d])([A-Z])', r'\1_\2', s1)
    return s2.lower().replace(' ', '_').replace('-', '_').replace('__', '_')


def terraform_resource_name(node_id: str) -> str:
    """Generate Terraform resource name from node_id."""
    return f"comfyui_{to_snake_case(node_id)}"


def run_extractor(script_path: str, comfyui_root: str) -> list:
    """Run an extractor script and return parsed JSON output."""
    result = subprocess.run(
        [sys.executable, script_path, comfyui_root],
        capture_output=True,
        text=True,
    )
    if result.returncode != 0:
        print(f"ERROR: {script_path} failed:", file=sys.stderr)
        print(result.stderr, file=sys.stderr)
        sys.exit(1)

    # Print extractor stderr to our stderr
    if result.stderr:
        print(result.stderr, file=sys.stderr, end='')

    try:
        return json.loads(result.stdout)
    except json.JSONDecodeError as e:
        print(f"ERROR: Failed to parse JSON from {script_path}: {e}", file=sys.stderr)
        sys.exit(1)


def validate_node(node: dict, index: int) -> list:
    """Validate a single node spec. Returns list of warning strings."""
    warnings = []
    required_fields = ['node_id', 'class_name', 'category', 'inputs', 'outputs', 'source']

    def validate_input_spec(input_spec: dict, context: str) -> None:
        if 'name' not in input_spec or 'type' not in input_spec:
            warnings.append(f"{context} missing required normalized fields")
        if 'required' not in input_spec:
            warnings.append(f"{context} missing required flag")
        if 'is_link_type' not in input_spec:
            warnings.append(f"{context} missing is_link_type flag")
        if 'validation_kind' not in input_spec:
            warnings.append(f"{context} missing validation_kind")
        if 'inventory_kind' not in input_spec:
            warnings.append(f"{context} missing inventory_kind")
        if 'supports_strict_plan_validation' not in input_spec:
            warnings.append(f"{context} missing supports_strict_plan_validation")

        for option_index, option in enumerate(input_spec.get('dynamic_combo_options') or []):
            option_context = f"{context} dynamic_combo_options[{option_index}]"
            if 'key' not in option:
                warnings.append(f"{option_context} missing key")
            if 'inputs' not in option:
                warnings.append(f"{option_context} missing inputs")
                continue
            for child_index, child_input in enumerate(option['inputs']):
                validate_input_spec(
                    child_input,
                    f"{option_context} input '{child_input.get('name', child_index)}'",
                )

    for field in required_fields:
        if field not in node:
            warnings.append(f"Node #{index} missing required field '{field}'")

    if 'node_id' in node and not node['node_id']:
        warnings.append(f"Node #{index} has empty node_id")

    if 'source' in node:
        if 'file' not in node['source']:
            warnings.append(f"Node '{node.get('node_id', '?')}' missing source.file")
        if 'pattern' not in node['source']:
            warnings.append(f"Node '{node.get('node_id', '?')}' missing source.pattern")

    for input_index, input_spec in enumerate(node.get('inputs', [])):
        validate_input_spec(
            input_spec,
            f"Node '{node.get('node_id', '?')}' input '{input_spec.get('name', input_index)}'",
        )

    for output_index, output_spec in enumerate(node.get('outputs', [])):
        if 'name' not in output_spec or 'type' not in output_spec or 'slot_index' not in output_spec:
            warnings.append(
                f"Node '{node.get('node_id', '?')}' output #{output_index} missing required normalized fields"
            )
        if 'is_list' not in output_spec:
            warnings.append(
                f"Node '{node.get('node_id', '?')}' output '{output_spec.get('name', output_index)}' missing is_list flag"
            )

    # Ensure terraform_resource_name is set
    if 'terraform_resource_name' not in node or not node.get('terraform_resource_name'):
        node['terraform_resource_name'] = terraform_resource_name(node.get('node_id', ''))

    return warnings


def classify_validation_kind(input_spec: dict) -> tuple[str, str, bool]:
    """Classify input validation behavior from extracted metadata."""
    options = input_spec.get('options') or []
    dynamic_options = bool(input_spec.get('dynamic_options'))
    dynamic_source = input_spec.get('dynamic_options_source') or ""
    input_type = input_spec.get('type', '')
    is_link_type = bool(input_spec.get('is_link_type'))

    if options and not dynamic_options:
        return (VALIDATION_KIND_STATIC_ENUM, "", True)

    inventory_match = INVENTORY_SOURCE_RE.fullmatch(dynamic_source)
    if dynamic_options and inventory_match:
        return (VALIDATION_KIND_DYNAMIC_INVENTORY, inventory_match.group(1), True)

    if dynamic_options:
        return (VALIDATION_KIND_DYNAMIC_EXPRESSION, "", False)

    if input_type == 'COMBO' and options:
        return (VALIDATION_KIND_STATIC_ENUM, "", True)

    if is_link_type:
        return (VALIDATION_KIND_FREEFORM, "", True)

    return (VALIDATION_KIND_FREEFORM, "", True)


def annotate_validation_metadata(node: dict) -> None:
    """Add generated validation metadata to extracted inputs."""
    def annotate_input(input_spec: dict) -> None:
        validation_kind, inventory_kind, strict = classify_validation_kind(input_spec)
        input_spec['validation_kind'] = validation_kind
        input_spec['inventory_kind'] = inventory_kind
        input_spec['supports_strict_plan_validation'] = strict
        for option in input_spec.get('dynamic_combo_options') or []:
            for child_input in option.get('inputs', []):
                annotate_input(child_input)

    for input_spec in node.get('inputs', []):
        annotate_input(input_spec)


def main():
    if len(sys.argv) < 2:
        print("Usage: merge_specs.py <comfyui_root>", file=sys.stderr)
        sys.exit(1)

    comfyui_root = os.path.abspath(sys.argv[1])
    scripts_dir = os.path.dirname(os.path.abspath(__file__))

    v1_script = os.path.join(scripts_dir, 'extract_v1_nodes.py')
    v3_script = os.path.join(scripts_dir, 'extract_v3_nodes.py')

    print("=" * 60, file=sys.stderr)
    print("Merging V1 and V3 node specifications", file=sys.stderr)
    print("=" * 60, file=sys.stderr)

    # Run extractors
    print("\n--- Running V1 extractor ---", file=sys.stderr)
    v1_nodes = run_extractor(v1_script, comfyui_root)
    print(f"V1 extracted: {len(v1_nodes)} nodes", file=sys.stderr)

    print("\n--- Running V3 extractor ---", file=sys.stderr)
    v3_nodes = run_extractor(v3_script, comfyui_root)
    print(f"V3 extracted: {len(v3_nodes)} nodes", file=sys.stderr)

    # Merge and deduplicate
    all_nodes = []
    seen_ids = {}
    duplicates = []

    for node in v1_nodes + v3_nodes:
        annotate_validation_metadata(node)
        nid = node.get('node_id', '')
        if nid in seen_ids:
            duplicates.append((nid, seen_ids[nid]['source']['pattern'], node['source']['pattern']))
            # Prefer V3 over V1 if duplicate (V3 is more structured)
            if node['source']['pattern'].startswith('v3'):
                # Replace the V1 entry
                all_nodes = [n for n in all_nodes if n['node_id'] != nid]
                all_nodes.append(node)
                seen_ids[nid] = node
        else:
            all_nodes.append(node)
            seen_ids[nid] = node

    # Validate all nodes
    all_warnings = []
    for i, node in enumerate(all_nodes):
        warnings = validate_node(node, i)
        all_warnings.extend(warnings)

    # Check for terraform_resource_name collisions
    tf_names = {}
    for node in all_nodes:
        tf_name = node.get('terraform_resource_name', '')
        if tf_name in tf_names:
            all_warnings.append(
                f"Terraform name collision: '{tf_name}' used by both "
                f"'{tf_names[tf_name]}' and '{node['node_id']}'"
            )
        else:
            tf_names[tf_name] = node['node_id']

    # Sort by node_id for deterministic output
    all_nodes.sort(key=lambda n: n.get('node_id', ''))

    # Detect ComfyUI version from git tag
    comfyui_version = "unknown"
    try:
        result = subprocess.run(
            ['git', 'describe', '--tags', '--exact-match'],
            capture_output=True, text=True, cwd=comfyui_root
        )
        if result.returncode == 0:
            comfyui_version = result.stdout.strip()
    except Exception:
        pass

    # Build final output
    output = {
        "version": "1.0.0",
        "comfyui_version": comfyui_version,
        "extracted_at": datetime.now(timezone.utc).isoformat(),
        "total_nodes": len(all_nodes),
        "nodes": all_nodes,
    }

    # Summary
    patterns = {}
    for node in all_nodes:
        p = node.get('source', {}).get('pattern', 'unknown')
        patterns[p] = patterns.get(p, 0) + 1

    print("\n" + "=" * 60, file=sys.stderr)
    print("MERGE SUMMARY", file=sys.stderr)
    print("=" * 60, file=sys.stderr)
    print(f"Total unique nodes: {len(all_nodes)}", file=sys.stderr)
    for pattern, count in sorted(patterns.items()):
        print(f"  {pattern}: {count}", file=sys.stderr)
    if duplicates:
        print(f"\nDuplicates resolved: {len(duplicates)}", file=sys.stderr)
        for nid, p1, p2 in duplicates:
            print(f"  {nid}: {p1} → {p2} (kept V3)", file=sys.stderr)
    if all_warnings:
        print(f"\nWarnings: {len(all_warnings)}", file=sys.stderr)
        for w in all_warnings:
            print(f"  ⚠ {w}", file=sys.stderr)
    else:
        print("\nNo warnings.", file=sys.stderr)

    # Write to file
    output_path = os.path.join(scripts_dir, 'node_specs.json')
    with open(output_path, 'w') as f:
        json.dump(output, f, indent=2)
    print(f"\nWritten to: {output_path}", file=sys.stderr)

    # Also write to stdout
    json.dump(output, sys.stdout, indent=2)


if __name__ == '__main__':
    main()
