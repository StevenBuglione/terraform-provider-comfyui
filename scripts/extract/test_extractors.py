#!/usr/bin/env python3
"""Tests for node spec extractors and merge pipeline."""

import ast
import json
import io
import os
import subprocess
import sys
import unittest
from contextlib import redirect_stderr

SCRIPT_DIR = os.path.dirname(os.path.abspath(__file__))
PROJECT_ROOT = os.path.join(SCRIPT_DIR, '..', '..')
COMFYUI_ROOT = os.path.join(PROJECT_ROOT, 'third_party', 'ComfyUI')
SPEC_PATH = os.path.join(SCRIPT_DIR, 'node_specs.json')
UI_HINTS_PATH = os.path.join(SCRIPT_DIR, 'node_ui_hints.json')

if SCRIPT_DIR not in sys.path:
    sys.path.insert(0, SCRIPT_DIR)

import extract_v3_nodes


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

    def test_collect_local_assignments_ignores_bare_annotations(self):
        tree = ast.parse(
            """
def define_schema():
    options: list[object]
    resolved = [IO.DynamicCombo.Option("demo", [])]
"""
        )

        func = tree.body[0]
        assignments = extract_v3_nodes.collect_local_assignments(func.body)

        self.assertNotIn("options", assignments)
        self.assertIn("resolved", assignments)

    def test_parse_dynamic_combo_options_warns_on_unparsed_option(self):
        options_node = ast.parse(
            """
[
    IO.DynamicCombo.Option("demo", []),
    not_an_option,
]
"""
        ).body[0].value

        stderr = io.StringIO()
        with redirect_stderr(stderr):
            parsed = extract_v3_nodes.parse_dynamic_combo_options(options_node, {})

        self.assertIsNone(parsed)
        self.assertIn("Could not parse dynamic combo option", stderr.getvalue())

    def test_parse_input_call_resolves_local_dynamic_combo_options(self):
        tree = ast.parse(
            """
def define_schema():
    sampling_options = [
        io.DynamicCombo.Option("on", [io.Float.Input("temperature", default=0.7)]),
        io.DynamicCombo.Option("off", []),
    ]
    return io.DynamicCombo.Input("sampling_mode", options=sampling_options)
"""
        )

        func = tree.body[0]
        local_assignments = extract_v3_nodes.collect_local_assignments(func.body)
        call = func.body[-1].value

        parsed = extract_v3_nodes.parse_input_call(call, call.func.value, local_assignments)

        self.assertEqual(parsed["type"], "COMFY_DYNAMICCOMBO_V3")
        self.assertEqual(parsed["dynamic_options_source"], "sampling_options")
        self.assertEqual([o["key"] for o in parsed["dynamic_combo_options"]], ["on", "off"])
        self.assertEqual(
            [child["name"] for child in parsed["dynamic_combo_options"][0]["inputs"]],
            ["temperature"],
        )

    def test_parse_schema_call_does_not_resolve_top_level_local_lists(self):
        tree = ast.parse(
            """
class Demo(io.ComfyNode):
    @classmethod
    def define_schema(cls):
        inputs = [io.Image.Input("images")]
        outputs = [io.Image.Output(display_name="images")]
        return io.Schema(node_id="Demo", inputs=inputs, outputs=outputs)
"""
        )

        class_node = tree.body[0]
        func = class_node.body[0]
        local_assignments = extract_v3_nodes.collect_local_assignments(func.body)
        schema_call = extract_v3_nodes.find_schema_call(func.body)

        parsed = extract_v3_nodes.parse_schema_call(
            schema_call, "demo.py", class_node.name, local_assignments
        )

        self.assertEqual(parsed["inputs"], [])
        self.assertEqual(parsed["outputs"], [])

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

    @unittest.skipUnless(comfyui_available(), "ComfyUI submodule not initialized")
    def test_extracts_inline_dynamic_combo_option_inputs_for_wan(self):
        script = os.path.join(SCRIPT_DIR, 'extract_v3_nodes.py')
        nodes = run_extractor(script, COMFYUI_ROOT)

        node = next(n for n in nodes if n['node_id'] == 'Wan2ImageToVideoApi')
        model_input = next(i for i in node['inputs'] if i['name'] == 'model')

        self.assertEqual(model_input['type'], 'COMFY_DYNAMICCOMBO_V3')
        self.assertIsInstance(model_input['dynamic_options_source'], str)
        self.assertIn("DynamicCombo.Option", model_input['dynamic_options_source'])
        self.assertIn("wan2.7-i2v", model_input['dynamic_options_source'])
        self.assertEqual([option['key'] for option in model_input['dynamic_combo_options']], ['wan2.7-i2v'])

        option_inputs = model_input['dynamic_combo_options'][0]['inputs']
        self.assertEqual(
            [child['name'] for child in option_inputs],
            ['prompt', 'negative_prompt', 'resolution', 'duration'],
        )

        prompt_input = option_inputs[0]
        self.assertEqual(prompt_input['type'], 'STRING')
        self.assertTrue(prompt_input['required'])
        self.assertTrue(prompt_input['multiline'])

        negative_prompt_input = option_inputs[1]
        self.assertEqual(negative_prompt_input['type'], 'STRING')
        self.assertTrue(negative_prompt_input['multiline'])

        resolution_input = option_inputs[2]
        self.assertEqual(resolution_input['type'], 'COMBO')
        self.assertEqual(resolution_input['options'], ['720P', '1080P'])

        duration_input = option_inputs[3]
        self.assertEqual(duration_input['type'], 'INT')
        self.assertEqual(duration_input['default'], 5)
        self.assertEqual(duration_input['min'], 2)
        self.assertEqual(duration_input['max'], 15)

    @unittest.skipUnless(comfyui_available(), "ComfyUI submodule not initialized")
    def test_extracts_named_dynamic_combo_option_inputs_for_textgen(self):
        script = os.path.join(SCRIPT_DIR, 'extract_v3_nodes.py')
        nodes = run_extractor(script, COMFYUI_ROOT)

        node = next(n for n in nodes if n['node_id'] == 'TextGenerate')
        sampling_mode = next(i for i in node['inputs'] if i['name'] == 'sampling_mode')

        self.assertEqual(sampling_mode['type'], 'COMFY_DYNAMICCOMBO_V3')
        self.assertEqual(sampling_mode['dynamic_options_source'], 'sampling_options')
        self.assertEqual([o['key'] for o in sampling_mode['dynamic_combo_options']], ['on', 'off'])
        self.assertEqual(
            [child['name'] for child in sampling_mode['dynamic_combo_options'][0]['inputs']],
            ['temperature', 'top_k', 'top_p', 'min_p', 'repetition_penalty', 'seed'],
        )
        self.assertEqual(sampling_mode['dynamic_combo_options'][1]['inputs'], [])

    @unittest.skipUnless(comfyui_available(), "ComfyUI submodule not initialized")
    def test_base_processing_nodes_do_not_expand_local_schema_lists(self):
        script = os.path.join(SCRIPT_DIR, 'extract_v3_nodes.py')
        nodes = run_extractor(script, COMFYUI_ROOT)

        image_processing = next(n for n in nodes if n['node_id'] == 'ImageProcessingNode')
        text_processing = next(n for n in nodes if n['node_id'] == 'TextProcessingNode')

        self.assertEqual(image_processing['inputs'], [])
        self.assertEqual(text_processing['inputs'], [])


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

    @unittest.skipUnless(os.path.exists(SPEC_PATH), "node_specs.json not found")
    def test_validation_kind_classification_for_static_enum(self):
        with open(SPEC_PATH) as f:
            spec = json.load(f)

        node = next(n for n in spec['nodes'] if n['node_id'] == 'ByteDanceTextToVideoNode')
        model_input = next(i for i in node['inputs'] if i['name'] == 'model')

        self.assertEqual(model_input['validation_kind'], 'static_enum')
        self.assertEqual(model_input['inventory_kind'], '')
        self.assertTrue(model_input['supports_strict_plan_validation'])

    @unittest.skipUnless(os.path.exists(SPEC_PATH), "node_specs.json not found")
    def test_validation_kind_classification_for_dynamic_inventory(self):
        with open(SPEC_PATH) as f:
            spec = json.load(f)

        checkpoint = next(n for n in spec['nodes'] if n['node_id'] == 'CheckpointLoaderSimple')
        ckpt_name = next(i for i in checkpoint['inputs'] if i['name'] == 'ckpt_name')
        self.assertEqual(ckpt_name['validation_kind'], 'dynamic_inventory')
        self.assertEqual(ckpt_name['inventory_kind'], 'checkpoints')
        self.assertTrue(ckpt_name['supports_strict_plan_validation'])

        lora = next(n for n in spec['nodes'] if n['node_id'] == 'LoraLoader')
        lora_name = next(i for i in lora['inputs'] if i['name'] == 'lora_name')
        self.assertEqual(lora_name['validation_kind'], 'dynamic_inventory')
        self.assertEqual(lora_name['inventory_kind'], 'loras')
        self.assertTrue(lora_name['supports_strict_plan_validation'])

    @unittest.skipUnless(os.path.exists(SPEC_PATH), "node_specs.json not found")
    def test_validation_kind_classification_for_unsupported_dynamic_expression(self):
        with open(SPEC_PATH) as f:
            spec = json.load(f)

        node = next(n for n in spec['nodes'] if n['node_id'] == 'BasicScheduler')
        moderation = next(i for i in node['inputs'] if i['name'] == 'scheduler')

        self.assertEqual(moderation['validation_kind'], 'dynamic_expression')
        self.assertEqual(moderation['inventory_kind'], '')
        self.assertFalse(moderation['supports_strict_plan_validation'])

    @unittest.skipUnless(os.path.exists(SPEC_PATH), "node_specs.json not found")
    def test_dynamic_combo_options_are_preserved_in_node_specs(self):
        with open(SPEC_PATH) as f:
            spec = json.load(f)

        wan = next(n for n in spec['nodes'] if n['node_id'] == 'Wan2ImageToVideoApi')
        wan_model = next(i for i in wan['inputs'] if i['name'] == 'model')
        self.assertEqual([o['key'] for o in wan_model['dynamic_combo_options']], ['wan2.7-i2v'])
        self.assertEqual(
            [child['name'] for child in wan_model['dynamic_combo_options'][0]['inputs']],
            ['prompt', 'negative_prompt', 'resolution', 'duration'],
        )

        textgen = next(n for n in spec['nodes'] if n['node_id'] == 'TextGenerate')
        sampling_mode = next(i for i in textgen['inputs'] if i['name'] == 'sampling_mode')
        self.assertEqual([o['key'] for o in sampling_mode['dynamic_combo_options']], ['on', 'off'])
        self.assertEqual(
            [child['name'] for child in sampling_mode['dynamic_combo_options'][0]['inputs']],
            ['temperature', 'top_k', 'top_p', 'min_p', 'repetition_penalty', 'seed'],
        )


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

    @unittest.skipUnless(comfyui_available(), "ComfyUI submodule not initialized")
    def test_ui_hints_capture_comfyui_version_metadata(self):
        with open(UI_HINTS_PATH) as f:
            hints = json.load(f)
        result = subprocess.run(
            ['git', '-C', COMFYUI_ROOT, 'describe', '--tags', '--always', '--dirty'],
            capture_output=True, text=True,
        )
        self.assertEqual(result.returncode, 0, f"git describe failed: {result.stderr}")
        expected_version = result.stdout.strip()
        self.assertNotEqual(hints['comfyui_version'], 'unknown')
        self.assertEqual(hints['comfyui_version'], expected_version)


if __name__ == '__main__':
    unittest.main(verbosity=2)
