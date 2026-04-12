#!/usr/bin/env python3
"""Tests for node spec extractors and merge pipeline."""

import json
import os
import subprocess
import sys
import unittest

SCRIPT_DIR = os.path.dirname(os.path.abspath(__file__))
PROJECT_ROOT = os.path.join(SCRIPT_DIR, '..', '..')
COMFYUI_ROOT = os.path.join(PROJECT_ROOT, 'third_party', 'ComfyUI')
SPEC_PATH = os.path.join(SCRIPT_DIR, 'node_specs.json')
UI_HINTS_PATH = os.path.join(SCRIPT_DIR, 'node_ui_hints.json')


def comfyui_available():
    """Check if ComfyUI submodule is initialized."""
    return os.path.isdir(COMFYUI_ROOT) and os.path.isfile(
        os.path.join(COMFYUI_ROOT, 'nodes.py')
    )


def run_extractor(script, comfyui_root):
    """Run an extractor script and return parsed JSON."""
    result = subprocess.run(
        [sys.executable, script, comfyui_root],
        capture_output=True, text=True,
    )
    assert result.returncode == 0, f"Extractor failed (exit {result.returncode}): {result.stderr}"
    return json.loads(result.stdout)


class TestV1Extractor(unittest.TestCase):
    """Test V1 node extractor."""

    @unittest.skipUnless(comfyui_available(), "ComfyUI submodule not initialized")
    def test_produces_nodes(self):
        script = os.path.join(SCRIPT_DIR, 'extract_v1_nodes.py')
        nodes = run_extractor(script, COMFYUI_ROOT)
        self.assertGreaterEqual(len(nodes), 100, f"Expected 100+ V1 nodes, got {len(nodes)}")

    @unittest.skipUnless(comfyui_available(), "ComfyUI submodule not initialized")
    def test_ksampler_exists(self):
        script = os.path.join(SCRIPT_DIR, 'extract_v1_nodes.py')
        nodes = run_extractor(script, COMFYUI_ROOT)
        ksampler = next((n for n in nodes if n['node_id'] == 'KSampler'), None)
        self.assertIsNotNone(ksampler, "KSampler not found in V1 nodes")
        self.assertEqual(ksampler['category'], 'sampling')
        self.assertEqual(ksampler['terraform_resource_name'], 'comfyui_ksampler')
        # Check model input exists
        model_inputs = [i for i in ksampler['inputs'] if i['name'] == 'model' and i['type'] == 'MODEL']
        self.assertTrue(len(model_inputs) > 0, "KSampler missing model input")
        # Check LATENT output exists
        latent_outputs = [o for o in ksampler['outputs'] if o['type'] == 'LATENT']
        self.assertTrue(len(latent_outputs) > 0, "KSampler missing LATENT output")

    @unittest.skipUnless(comfyui_available(), "ComfyUI submodule not initialized")
    def test_required_fields(self):
        script = os.path.join(SCRIPT_DIR, 'extract_v1_nodes.py')
        nodes = run_extractor(script, COMFYUI_ROOT)
        for node in nodes:
            self.assertIn('node_id', node, f"Missing node_id")
            self.assertIn('inputs', node, f"Missing inputs in {node.get('node_id', '?')}")
            self.assertIn('outputs', node, f"Missing outputs in {node.get('node_id', '?')}")
            self.assertIn('source', node, f"Missing source in {node.get('node_id', '?')}")
            self.assertIn('terraform_resource_name', node)
            self.assertTrue(
                node['terraform_resource_name'].startswith('comfyui_'),
                f"Bad terraform name: {node['terraform_resource_name']}"
            )

    @unittest.skipUnless(comfyui_available(), "ComfyUI submodule not initialized")
    def test_source_patterns(self):
        script = os.path.join(SCRIPT_DIR, 'extract_v1_nodes.py')
        nodes = run_extractor(script, COMFYUI_ROOT)
        patterns = {n['source']['pattern'] for n in nodes}
        self.assertTrue(
            patterns.issubset({'v1_core', 'v1_extras'}),
            f"Unexpected V1 patterns: {patterns}"
        )
        self.assertIn('v1_core', patterns, "Missing v1_core pattern")


class TestV3Extractor(unittest.TestCase):
    """Test V3 node extractor."""

    @unittest.skipUnless(comfyui_available(), "ComfyUI submodule not initialized")
    def test_produces_nodes(self):
        script = os.path.join(SCRIPT_DIR, 'extract_v3_nodes.py')
        nodes = run_extractor(script, COMFYUI_ROOT)
        self.assertGreaterEqual(len(nodes), 400, f"Expected 400+ V3 nodes, got {len(nodes)}")

    @unittest.skipUnless(comfyui_available(), "ComfyUI submodule not initialized")
    def test_basic_scheduler_exists(self):
        script = os.path.join(SCRIPT_DIR, 'extract_v3_nodes.py')
        nodes = run_extractor(script, COMFYUI_ROOT)
        bs = next((n for n in nodes if n['node_id'] == 'BasicScheduler'), None)
        self.assertIsNotNone(bs, "BasicScheduler not found in V3 nodes")
        self.assertIn('sampling', bs['category'])

    @unittest.skipUnless(comfyui_available(), "ComfyUI submodule not initialized")
    def test_required_fields(self):
        script = os.path.join(SCRIPT_DIR, 'extract_v3_nodes.py')
        nodes = run_extractor(script, COMFYUI_ROOT)
        for node in nodes:
            self.assertIn('node_id', node)
            self.assertIn('inputs', node)
            self.assertIn('outputs', node)
            self.assertIn('source', node)
            self.assertIn(
                node['source']['pattern'], ('v3_extras', 'v3_api'),
                f"Unexpected V3 pattern for {node['node_id']}: {node['source']['pattern']}"
            )


class TestMerge(unittest.TestCase):
    """Test merge pipeline."""

    @unittest.skipUnless(comfyui_available(), "ComfyUI submodule not initialized")
    def test_produces_valid_spec(self):
        script = os.path.join(SCRIPT_DIR, 'merge_specs.py')
        result = subprocess.run(
            [sys.executable, script, COMFYUI_ROOT],
            capture_output=True, text=True,
        )
        self.assertEqual(result.returncode, 0, f"Merge failed: {result.stderr}")
        spec = json.loads(result.stdout)
        self.assertEqual(spec['version'], '1.0.0')
        self.assertGreaterEqual(spec['total_nodes'], 600)
        self.assertEqual(spec['total_nodes'], len(spec['nodes']))

    @unittest.skipUnless(comfyui_available(), "ComfyUI submodule not initialized")
    def test_no_duplicate_node_ids(self):
        script = os.path.join(SCRIPT_DIR, 'merge_specs.py')
        result = subprocess.run(
            [sys.executable, script, COMFYUI_ROOT],
            capture_output=True, text=True,
        )
        self.assertEqual(result.returncode, 0)
        spec = json.loads(result.stdout)
        ids = [n['node_id'] for n in spec['nodes']]
        self.assertEqual(len(ids), len(set(ids)), "Duplicate node_ids found")

    @unittest.skipUnless(comfyui_available(), "ComfyUI submodule not initialized")
    def test_no_duplicate_terraform_names(self):
        script = os.path.join(SCRIPT_DIR, 'merge_specs.py')
        result = subprocess.run(
            [sys.executable, script, COMFYUI_ROOT],
            capture_output=True, text=True,
        )
        self.assertEqual(result.returncode, 0)
        spec = json.loads(result.stdout)
        tf_names = [n['terraform_resource_name'] for n in spec['nodes']]
        self.assertEqual(len(tf_names), len(set(tf_names)), "Duplicate terraform names found")


class TestSpecValidation(unittest.TestCase):
    """Validate existing node_specs.json."""

    @unittest.skipUnless(os.path.exists(SPEC_PATH), "node_specs.json not found")
    def test_spec_structure(self):
        with open(SPEC_PATH) as f:
            spec = json.load(f)
        self.assertIn('version', spec)
        self.assertIn('nodes', spec)
        self.assertIn('total_nodes', spec)
        self.assertEqual(spec['total_nodes'], len(spec['nodes']))

    @unittest.skipUnless(os.path.exists(SPEC_PATH), "node_specs.json not found")
    def test_all_nodes_valid(self):
        with open(SPEC_PATH) as f:
            spec = json.load(f)
        for node in spec['nodes']:
            self.assertIn('node_id', node, "Node missing node_id")
            self.assertIn('inputs', node, f"Missing inputs in {node['node_id']}")
            self.assertIn('outputs', node, f"Missing outputs in {node['node_id']}")
            self.assertIn('source', node, f"Missing source in {node['node_id']}")
            self.assertIn('terraform_resource_name', node, f"Missing terraform name in {node['node_id']}")
            self.assertTrue(
                node['terraform_resource_name'].startswith('comfyui_'),
                f"Bad terraform name in {node['node_id']}: {node['terraform_resource_name']}"
            )

    @unittest.skipUnless(os.path.exists(SPEC_PATH), "node_specs.json not found")
    def test_input_fields(self):
        with open(SPEC_PATH) as f:
            spec = json.load(f)
        for node in spec['nodes']:
            for inp in node.get('inputs', []):
                self.assertIn('name', inp, f"Input missing name in {node['node_id']}")
                self.assertIn('type', inp, f"Input missing type in {node['node_id']}")
                self.assertIn('required', inp, f"Input missing required in {node['node_id']}")

    @unittest.skipUnless(os.path.exists(SPEC_PATH), "node_specs.json not found")
    def test_output_fields(self):
        with open(SPEC_PATH) as f:
            spec = json.load(f)
        for node in spec['nodes']:
            for out in node.get('outputs', []):
                self.assertIn('type', out, f"Output missing type in {node['node_id']}")
                self.assertIn('slot_index', out, f"Output missing slot_index in {node['node_id']}")

    @unittest.skipUnless(os.path.exists(SPEC_PATH), "node_specs.json not found")
    def test_all_source_patterns(self):
        with open(SPEC_PATH) as f:
            spec = json.load(f)
        patterns = {n['source']['pattern'] for n in spec['nodes']}
        expected = {'v1_core', 'v1_extras', 'v3_extras', 'v3_api'}
        self.assertEqual(patterns, expected, f"Unexpected patterns: {patterns}")

    @unittest.skipUnless(os.path.exists(SPEC_PATH), "node_specs.json not found")
    def test_no_duplicates(self):
        with open(SPEC_PATH) as f:
            spec = json.load(f)
        ids = [n['node_id'] for n in spec['nodes']]
        self.assertEqual(len(ids), len(set(ids)), "Duplicate node_ids in spec")
        tf_names = [n['terraform_resource_name'] for n in spec['nodes']]
        self.assertEqual(len(tf_names), len(set(tf_names)), "Duplicate terraform names in spec")


class TestUIHintsValidation(unittest.TestCase):
    """Validate extracted frontend UI sizing hints."""

    @unittest.skipUnless(os.path.exists(UI_HINTS_PATH), "node_ui_hints.json not found")
    def test_ui_hints_structure(self):
        with open(UI_HINTS_PATH) as f:
            hints = json.load(f)
        self.assertIn('version', hints)
        self.assertIn('extracted_at', hints)
        self.assertIn('comfyui_commit_sha', hints)
        self.assertIn('comfyui_version', hints)
        self.assertIn('total_nodes', hints)
        self.assertIn('failed_nodes', hints)
        self.assertIn('nodes', hints)
        self.assertEqual(hints['failed_nodes'], [])

    @unittest.skipUnless(os.path.exists(UI_HINTS_PATH), "node_ui_hints.json not found")
    def test_clip_text_encode_has_multiline_min_size(self):
        with open(UI_HINTS_PATH) as f:
            hints = json.load(f)
        clip = hints['nodes'].get('CLIPTextEncode')
        self.assertIsNotNone(clip, "CLIPTextEncode missing from UI hints")
        self.assertGreaterEqual(clip['min_width'], 400)
        self.assertGreaterEqual(clip['min_height'], 200)
        self.assertIn('text', clip['widgets'])
        text_widget = clip['widgets']['text']
        self.assertEqual(text_widget['widget_type'], 'customtext')
        self.assertGreaterEqual(text_widget['min_node_width'], 400)
        self.assertGreaterEqual(text_widget['min_node_height'], 200)


if __name__ == '__main__':
    unittest.main(verbosity=2)
