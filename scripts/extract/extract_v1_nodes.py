#!/usr/bin/env python3
"""Extract V1-pattern ComfyUI nodes from source code using AST parsing.

Parses nodes.py (core) and 10 V1 extras files in comfy_extras/ to extract
node specifications including INPUT_TYPES, RETURN_TYPES, FUNCTION, CATEGORY, etc.

Usage:
    python3 scripts/extract/extract_v1_nodes.py <comfyui_root>

Output: JSON array of node specs to stdout. Summary to stderr.
"""

import ast
import json
import os
import re
import sys
from typing import Any, Optional


# Types that represent runtime objects passed between nodes (not user-editable)
LINK_TYPES = frozenset({
    "MODEL", "CLIP", "VAE", "CONDITIONING", "LATENT", "IMAGE", "MASK",
    "CONTROL_NET", "UPSCALE_MODEL", "CLIP_VISION", "CLIP_VISION_OUTPUT",
    "STYLE_MODEL", "GLIGEN", "NOISE", "GUIDER", "SAMPLER", "SIGMAS",
    "HOOKS", "TIMESTEPS_RANGE", "HOOK_KEYFRAMES", "AUDIO", "WEBCAM",
    "LORA_MODEL", "LOSS_MAP", "POINT", "FACE_ANALYSIS", "BBOX", "SEGS",
    "VIDEO", "LATENT_UPSCALE_MODEL", "MODEL_PATCH",
})

# Primitive types that users enter directly
PRIMITIVE_TYPES = frozenset({"INT", "FLOAT", "STRING", "BOOLEAN"})

# Files to parse: (relative_path, pattern_label)
CORE_FILE = ("nodes.py", "v1_core")
EXTRAS_FILES = [
    ("comfy_extras/nodes_hooks.py", "v1_extras"),
    ("comfy_extras/nodes_lora_debug.py", "v1_extras"),
    ("comfy_extras/nodes_lt_upsampler.py", "v1_extras"),
    ("comfy_extras/nodes_model_advanced.py", "v1_extras"),
    ("comfy_extras/nodes_model_merging.py", "v1_extras"),
    ("comfy_extras/nodes_model_merging_model_specific.py", "v1_extras"),
    ("comfy_extras/nodes_model_patch.py", "v1_extras"),
    ("comfy_extras/nodes_preview_any.py", "v1_extras"),
    ("comfy_extras/nodes_video_model.py", "v1_extras"),
    ("comfy_extras/nodes_webcam.py", "v1_extras"),
]


def warn(msg: str) -> None:
    print(f"  WARNING: {msg}", file=sys.stderr)


def to_snake_case(name: str) -> str:
    """Convert PascalCase/camelCase to snake_case.

    KSampler -> ksampler
    CLIPTextEncode -> clip_text_encode
    CheckpointLoaderSimple -> checkpoint_loader_simple
    SVD_img2vid_Conditioning -> svd_img2vid_conditioning
    """
    # Insert underscore before uppercase letters preceded by lowercase/digit
    s = re.sub(r'([a-z0-9])([A-Z])', r'\1_\2', name)
    # Insert underscore between acronyms (2+ uppercase) and next word
    s = re.sub(r'([A-Z]{2,})([A-Z][a-z])', r'\1_\2', s)
    return s.lower()


def terraform_resource_name(node_id: str) -> str:
    return f"comfyui_{to_snake_case(node_id)}"


def ast_to_source(node: ast.AST) -> str:
    """Convert an AST node back to source code string."""
    try:
        return ast.unparse(node)
    except Exception:
        return "<unknown>"


def safe_literal_eval(node: ast.AST) -> Any:
    """Try to evaluate an AST expression as a literal. Returns None on failure."""
    try:
        return ast.literal_eval(node)
    except (ValueError, TypeError, SyntaxError, RecursionError):
        return None


def is_dynamic_expr(node: ast.AST) -> bool:
    """Check if an AST node represents a dynamic (non-literal) expression."""
    return isinstance(node, (ast.Call, ast.Attribute, ast.Subscript, ast.Name))


def resolve_io_type(node: ast.AST) -> Optional[str]:
    """Resolve IO.STRING -> 'STRING', IO.CLIP -> 'CLIP', etc."""
    if isinstance(node, ast.Attribute) and isinstance(node.value, ast.Name):
        if node.value.id == "IO":
            return node.attr
    return None


def extract_type_from_tuple_element(node: ast.AST) -> tuple[str, Optional[list], bool, Optional[str]]:
    """Extract type info from the first element of an input tuple.

    Returns: (type_name, options_list_or_None, is_dynamic, dynamic_source_or_None)
    """
    # String literal: ("MODEL",) or ("INT",)
    if isinstance(node, ast.Constant) and isinstance(node.value, str):
        return (node.value, None, False, None)

    # IO.STRING, IO.MODEL, etc.
    io_type = resolve_io_type(node)
    if io_type is not None:
        return (io_type, None, False, None)

    # List literal: (["option1", "option2"],) - enum/combo
    if isinstance(node, ast.List):
        options = safe_literal_eval(node)
        if options is not None:
            return ("COMBO", options, False, None)
        # List with some dynamic elements
        return ("COMBO", None, True, ast_to_source(node))

    # Dynamic expression: function call, attribute access, etc.
    if is_dynamic_expr(node):
        source = ast_to_source(node)
        return ("COMBO", None, True, source)

    # Fallback
    return ("UNKNOWN", None, True, ast_to_source(node))


def extract_input_config(config_node: ast.AST) -> dict[str, Any]:
    """Extract configuration dict from the second element of an input tuple.

    E.g., {"default": 0, "min": 0, "max": 0xffffffffffffffff, "step": 1}
    """
    result: dict[str, Any] = {}
    if not isinstance(config_node, ast.Dict):
        return result

    for key_node, val_node in zip(config_node.keys, config_node.values):
        if key_node is None:
            continue
        key = safe_literal_eval(key_node)
        if key is None:
            continue

        val = safe_literal_eval(val_node)
        if val is None and isinstance(val_node, (ast.Name, ast.Attribute, ast.Call)):
            # Dynamic value like MAX_RESOLUTION or nodes.MAX_RESOLUTION
            result[key] = {"_dynamic": True, "_source": ast_to_source(val_node)}
        else:
            result[key] = val

    return result


def parse_input_entry(name: str, value_node: ast.AST, required: bool) -> dict:
    """Parse a single input entry from INPUT_TYPES dict.

    value_node is the tuple like ("MODEL",) or ("INT", {"default": 0, ...})
    """
    input_spec = {
        "name": name,
        "type": "UNKNOWN",
        "required": required,
        "is_link_type": False,
        "default": None,
        "min": None,
        "max": None,
        "step": None,
        "options": None,
        "multiline": None,
        "dynamic_options": None,
        "dynamic_options_source": None,
        "tooltip": None,
        "display_name": None,
    }

    if not isinstance(value_node, ast.Tuple) or len(value_node.elts) == 0:
        # Might be a bare expression - try to extract type
        type_name, options, is_dynamic, dyn_source = extract_type_from_tuple_element(value_node)
        input_spec["type"] = type_name
        if options:
            input_spec["options"] = options
        if is_dynamic:
            input_spec["dynamic_options"] = True
            input_spec["dynamic_options_source"] = dyn_source
        return input_spec

    # First element: type
    type_node = value_node.elts[0]
    type_name, options, is_dynamic, dyn_source = extract_type_from_tuple_element(type_node)
    input_spec["type"] = type_name

    if options is not None:
        input_spec["options"] = [str(o) for o in options]
    if is_dynamic:
        input_spec["dynamic_options"] = True
        input_spec["dynamic_options_source"] = dyn_source

    # Determine is_link_type
    if type_name in LINK_TYPES:
        input_spec["is_link_type"] = True
    elif type_name in PRIMITIVE_TYPES or type_name == "COMBO":
        input_spec["is_link_type"] = False
    elif type_name not in PRIMITIVE_TYPES and type_name != "COMBO" and type_name != "UNKNOWN":
        # Unknown types that aren't primitive are likely link types
        input_spec["is_link_type"] = True

    # Second element: config dict (if present)
    if len(value_node.elts) >= 2:
        config = extract_input_config(value_node.elts[1])
        if "default" in config:
            val = config["default"]
            if isinstance(val, dict) and val.get("_dynamic"):
                input_spec["default"] = None
            else:
                input_spec["default"] = val
        if "min" in config:
            val = config["min"]
            if isinstance(val, dict) and val.get("_dynamic"):
                input_spec["min"] = None
            else:
                input_spec["min"] = val
        if "max" in config:
            val = config["max"]
            if isinstance(val, dict) and val.get("_dynamic"):
                input_spec["max"] = None
            else:
                input_spec["max"] = val
        if "step" in config:
            val = config["step"]
            if isinstance(val, dict) and val.get("_dynamic"):
                input_spec["step"] = None
            else:
                input_spec["step"] = val
        if "multiline" in config:
            input_spec["multiline"] = config["multiline"]
        if "tooltip" in config:
            input_spec["tooltip"] = config["tooltip"]
        if "display_name" in config:
            input_spec["display_name"] = config["display_name"]

    return input_spec


def parse_input_types_dict(
    dict_node: ast.Dict,
    method_node: Optional[ast.FunctionDef] = None,
) -> tuple[list[dict], list[dict]]:
    """Parse the full INPUT_TYPES return dict.

    Returns: (inputs_list, hidden_inputs_list)
    """
    inputs = []
    hidden_inputs = []

    for key_node, val_node in zip(dict_node.keys, dict_node.values):
        if key_node is None:
            continue
        section = safe_literal_eval(key_node)
        if section is None:
            continue

        # Handle variable references (e.g., return {"required": arg_dict})
        section_dict = val_node
        if isinstance(val_node, ast.Name) and method_node is not None:
            section_dict = reconstruct_dict_from_method(method_node, val_node.id)

        if not isinstance(section_dict, ast.Dict):
            if section in ("required", "optional"):
                warn(f"  Cannot statically resolve '{section}' section (dynamic variable)")
            continue

        if section in ("required", "optional"):
            is_required = section == "required"
            for inp_key, inp_val in zip(section_dict.keys, section_dict.values):
                if inp_key is None:
                    continue
                inp_name = safe_literal_eval(inp_key)
                if inp_name is None:
                    continue
                inp = parse_input_entry(inp_name, inp_val, is_required)
                inputs.append(inp)

        elif section == "hidden":
            for hid_key, hid_val in zip(section_dict.keys, section_dict.values):
                if hid_key is None:
                    continue
                hid_name = safe_literal_eval(hid_key)
                if hid_name is None:
                    continue
                hid_type = safe_literal_eval(hid_val)
                if hid_type is None:
                    hid_type = ast_to_source(hid_val)
                hidden_inputs.append({"name": hid_name, "type": str(hid_type)})

    return inputs, hidden_inputs


def reconstruct_dict_from_method(
    method_node: ast.FunctionDef, var_name: str
) -> Optional[ast.Dict]:
    """Reconstruct a dict built up via assignments in a method body.

    Handles patterns like:
        arg_dict = {"model1": ("MODEL",), "model2": ("MODEL",)}
        arg_dict["input_blocks.0."] = argument
        for i in range(12):
            arg_dict["input_blocks.{}.".format(i)] = argument
    """
    # Find the initial dict assignment
    base_dict = None
    extra_entries: list[tuple[ast.AST, ast.AST]] = []

    for stmt in method_node.body:
        # Direct assignment: arg_dict = {...}
        if isinstance(stmt, ast.Assign):
            for target in stmt.targets:
                if isinstance(target, ast.Name) and target.id == var_name:
                    if isinstance(stmt.value, ast.Dict):
                        base_dict = stmt.value

                # Subscript assignment: arg_dict["key"] = value
                if (isinstance(target, ast.Subscript) and
                    isinstance(target.value, ast.Name) and
                    target.value.id == var_name):
                    extra_entries.append((target.slice, stmt.value))

        # For loops that add entries: for i in range(N): arg_dict[expr] = val
        if isinstance(stmt, ast.For):
            for body_stmt in stmt.body:
                if isinstance(body_stmt, ast.Assign):
                    for target in body_stmt.targets:
                        if (isinstance(target, ast.Subscript) and
                            isinstance(target.value, ast.Name) and
                            target.value.id == var_name):
                            # Try to expand the loop
                            loop_entries = _expand_for_loop(stmt, target.slice, body_stmt.value)
                            extra_entries.extend(loop_entries)

    if base_dict is None:
        return None

    # Merge extra entries into the dict
    if extra_entries:
        new_keys = list(base_dict.keys) + [e[0] for e in extra_entries]
        new_values = list(base_dict.values) + [e[1] for e in extra_entries]
        merged = ast.Dict(keys=new_keys, values=new_values)
        return merged

    return base_dict


def _expand_for_loop(
    for_node: ast.For,
    key_expr: ast.AST,
    value_expr: ast.AST,
) -> list[tuple[ast.AST, ast.AST]]:
    """Try to expand a for loop that builds dict entries.

    Handles: for i in range(N): dict[f"prefix.{i}."] = value
    """
    entries = []

    # Try to determine the range
    if not isinstance(for_node.iter, ast.Call):
        return entries
    if not (isinstance(for_node.iter.func, ast.Name) and for_node.iter.func.id == "range"):
        return entries
    if len(for_node.iter.args) < 1:
        return entries

    range_end = safe_literal_eval(for_node.iter.args[0])
    if not isinstance(range_end, int):
        return entries

    range_start = 0
    if len(for_node.iter.args) >= 2:
        range_start = safe_literal_eval(for_node.iter.args[0]) or 0
        range_end = safe_literal_eval(for_node.iter.args[1]) or range_end

    loop_var = for_node.target
    if not isinstance(loop_var, ast.Name):
        return entries

    for i in range(range_start, range_end):
        # Create a constant key by substituting the loop variable
        try:
            key_source = ast_to_source(key_expr)
            # Replace .format(i) patterns
            key_str = key_source.replace(f"{loop_var.id}", str(i))
            # Try to evaluate the string expression
            if ".format(" in key_source:
                # Pattern: "prefix.{}.".format(i)
                template = None
                if isinstance(key_expr, ast.Call) and isinstance(key_expr.func, ast.Attribute):
                    if key_expr.func.attr == "format":
                        template = safe_literal_eval(key_expr.func.value)
                if template:
                    key_str = template.format(i)
                else:
                    key_str = f"_dynamic_{i}"
            elif isinstance(key_expr, ast.JoinedStr):
                # f-string pattern
                key_str = f"_dynamic_{i}"

            key_node = ast.Constant(value=key_str)
            entries.append((key_node, value_expr))
        except Exception:
            key_node = ast.Constant(value=f"_dynamic_{i}")
            entries.append((key_node, value_expr))

    return entries


def find_input_types_return(method_node: ast.FunctionDef) -> Optional[tuple[ast.Dict, ast.FunctionDef]]:
    """Find and return the dict node and method from INPUT_TYPES return statement."""
    for node in ast.walk(method_node):
        if isinstance(node, ast.Return) and node.value is not None:
            if isinstance(node.value, ast.Dict):
                return (node.value, method_node)
    return None


def get_class_attr_str(cls_node: ast.ClassDef, attr_name: str) -> Optional[str]:
    """Get a string class attribute value."""
    for item in cls_node.body:
        if isinstance(item, ast.Assign):
            for target in item.targets:
                if isinstance(target, ast.Name) and target.id == attr_name:
                    val = safe_literal_eval(item.value)
                    if isinstance(val, str):
                        return val
    return None


def get_class_attr_bool(cls_node: ast.ClassDef, attr_name: str, default: bool = False) -> bool:
    """Get a boolean class attribute value."""
    for item in cls_node.body:
        if isinstance(item, ast.Assign):
            for target in item.targets:
                if isinstance(target, ast.Name) and target.id == attr_name:
                    val = safe_literal_eval(item.value)
                    if isinstance(val, bool):
                        return val
                    # Handle NameConstant for older Python
                    if isinstance(item.value, ast.Constant):
                        return bool(item.value.value)
    return default


def get_class_attr_tuple(cls_node: ast.ClassDef, attr_name: str) -> Optional[tuple]:
    """Get a tuple class attribute value."""
    for item in cls_node.body:
        if isinstance(item, ast.Assign):
            for target in item.targets:
                if isinstance(target, ast.Name) and target.id == attr_name:
                    val = safe_literal_eval(item.value)
                    if isinstance(val, tuple):
                        return val
                    # Handle IO.TYPE references in tuples
                    if isinstance(item.value, ast.Tuple):
                        result = []
                        for elt in item.value.elts:
                            io_type = resolve_io_type(elt)
                            if io_type:
                                result.append(io_type)
                            else:
                                lit = safe_literal_eval(elt)
                                if lit is not None:
                                    result.append(lit)
                                else:
                                    result.append(ast_to_source(elt))
                        return tuple(result)
    return None


def get_class_attr_tuple_raw(cls_node: ast.ClassDef, attr_name: str) -> Optional[ast.Tuple]:
    """Get the raw AST Tuple node for a class attribute."""
    for item in cls_node.body:
        if isinstance(item, ast.Assign):
            for target in item.targets:
                if isinstance(target, ast.Name) and target.id == attr_name:
                    if isinstance(item.value, ast.Tuple):
                        return item.value
    return None


def find_input_types_method(cls_node: ast.ClassDef) -> Optional[ast.FunctionDef]:
    """Find the INPUT_TYPES classmethod on a class."""
    for item in cls_node.body:
        if isinstance(item, ast.FunctionDef) and item.name == "INPUT_TYPES":
            return item
    return None


def extract_class_info(
    cls_node: ast.ClassDef,
    parent_classes: dict[str, ast.ClassDef],
) -> Optional[dict]:
    """Extract all node info from a class definition.

    Uses parent_classes dict to resolve inherited attributes.
    """
    class_name = cls_node.name

    # Resolve inheritance chain for attribute lookup
    def _resolve_base_name(base: ast.AST) -> Optional[str]:
        if isinstance(base, ast.Name):
            return base.id
        elif isinstance(base, ast.Attribute):
            return ast_to_source(base)
        return None

    def get_attr_str(name: str) -> Optional[str]:
        val = get_class_attr_str(cls_node, name)
        if val is not None:
            return val
        for base in cls_node.bases:
            base_name = _resolve_base_name(base)
            if base_name and base_name in parent_classes:
                val = get_class_attr_str(parent_classes[base_name], name)
                if val is not None:
                    return val
        return None

    def get_attr_bool(name: str, default: bool = False) -> bool:
        val = get_class_attr_bool(cls_node, name, default=None)
        if val is not None:
            return val
        for base in cls_node.bases:
            base_name = _resolve_base_name(base)
            if base_name and base_name in parent_classes:
                val = get_class_attr_bool(parent_classes[base_name], name, default=None)
                if val is not None:
                    return val
        return default

    def get_attr_tuple(name: str) -> Optional[tuple]:
        val = get_class_attr_tuple(cls_node, name)
        if val is not None:
            return val
        for base in cls_node.bases:
            base_name = _resolve_base_name(base)
            if base_name and base_name in parent_classes:
                val = get_class_attr_tuple(parent_classes[base_name], name)
                if val is not None:
                    return val
        return None

    # FUNCTION is required
    function_name = get_attr_str("FUNCTION")
    if function_name is None:
        return None

    # CATEGORY is required
    category = get_attr_str("CATEGORY")
    if category is None:
        return None

    # RETURN_TYPES
    return_types = get_attr_tuple("RETURN_TYPES")
    if return_types is None:
        # Check if parent has it
        return_types = ()

    # RETURN_NAMES
    return_names = get_attr_tuple("RETURN_NAMES")

    # OUTPUT_IS_LIST
    output_is_list = get_attr_tuple("OUTPUT_IS_LIST")

    # Boolean flags
    is_output_node = get_attr_bool("OUTPUT_NODE", False)
    is_deprecated = get_attr_bool("DEPRECATED", False)
    is_experimental = get_attr_bool("EXPERIMENTAL", False)

    # DESCRIPTION
    description = get_attr_str("DESCRIPTION")

    # Parse INPUT_TYPES
    inputs = []
    hidden_inputs = []
    input_types_method = find_input_types_method(cls_node)
    if input_types_method is None:
        for base in cls_node.bases:
            base_name = _resolve_base_name(base)
            if base_name and base_name in parent_classes:
                input_types_method = find_input_types_method(parent_classes[base_name])
                if input_types_method is not None:
                    break

    if input_types_method is not None:
        result = find_input_types_return(input_types_method)
        if result is not None:
            return_dict, method_ref = result
            inputs, hidden_inputs = parse_input_types_dict(return_dict, method_ref)
        else:
            warn(f"{class_name}: Could not find return dict in INPUT_TYPES")
    else:
        warn(f"{class_name}: No INPUT_TYPES method found")

    # Build outputs
    outputs = []
    for i, rtype in enumerate(return_types):
        out_name = return_names[i] if return_names and i < len(return_names) else str(rtype)
        is_list = False
        if output_is_list and i < len(output_is_list):
            is_list = bool(output_is_list[i])
        outputs.append({
            "name": out_name,
            "type": str(rtype),
            "slot_index": i,
            "is_list": is_list,
        })

    return {
        "class_name": class_name,
        "description": description,
        "category": category,
        "function_name": function_name,
        "is_output_node": is_output_node,
        "is_deprecated": is_deprecated,
        "is_experimental": is_experimental,
        "inputs": inputs,
        "hidden_inputs": hidden_inputs,
        "outputs": outputs,
        "line_number": cls_node.lineno,
    }


def extract_node_class_mappings(tree: ast.Module) -> dict[str, str]:
    """Extract NODE_CLASS_MAPPINGS from the module AST.

    Returns dict of node_id -> class_name.
    """
    mappings = {}
    for node in ast.walk(tree):
        if not isinstance(node, ast.Assign):
            continue
        for target in node.targets:
            if not (isinstance(target, ast.Name) and target.id == "NODE_CLASS_MAPPINGS"):
                continue
            if isinstance(node.value, ast.Dict):
                for k, v in zip(node.value.keys, node.value.values):
                    if k is None:
                        continue
                    key = safe_literal_eval(k)
                    if key is None:
                        continue
                    # Value is a Name reference to a class
                    if isinstance(v, ast.Name):
                        mappings[key] = v.id
                    elif isinstance(v, ast.Attribute):
                        mappings[key] = ast_to_source(v)
    return mappings


def extract_node_display_name_mappings(tree: ast.Module) -> dict[str, str]:
    """Extract NODE_DISPLAY_NAME_MAPPINGS from the module AST."""
    mappings = {}
    for node in ast.walk(tree):
        if not isinstance(node, ast.Assign):
            continue
        for target in node.targets:
            if not (isinstance(target, ast.Name) and target.id == "NODE_DISPLAY_NAME_MAPPINGS"):
                continue
            if isinstance(node.value, ast.Dict):
                for k, v in zip(node.value.keys, node.value.values):
                    if k is None:
                        continue
                    key = safe_literal_eval(k)
                    val = safe_literal_eval(v)
                    if key and val:
                        mappings[key] = val
    return mappings


def extract_hooks_dynamic_mappings(tree: ast.Module, all_classes: dict[str, ast.ClassDef]) -> tuple[dict[str, str], dict[str, str]]:
    """Handle nodes_hooks.py dynamic NODE_CLASS_MAPPINGS built via loop.

    Looks for the pattern:
        node_list = [ClassA, ClassB, ...]
        for node in node_list:
            NODE_CLASS_MAPPINGS[node.NodeId] = node
            NODE_DISPLAY_NAME_MAPPINGS[node.NodeId] = node.NodeName
    """
    class_mappings = {}
    display_mappings = {}

    # Find node_list assignment
    node_list_classes = []
    for node in ast.walk(tree):
        if isinstance(node, ast.Assign):
            for target in node.targets:
                if isinstance(target, ast.Name) and target.id == "node_list":
                    if isinstance(node.value, ast.List):
                        for elt in node.value.elts:
                            if isinstance(elt, ast.Name):
                                node_list_classes.append(elt.id)

    # For each class in node_list, get NodeId and NodeName from class attrs
    for cls_name in node_list_classes:
        if cls_name not in all_classes:
            warn(f"hooks: class {cls_name} in node_list not found")
            continue
        cls_node = all_classes[cls_name]
        node_id = get_class_attr_str(cls_node, "NodeId")
        node_name = get_class_attr_str(cls_node, "NodeName")
        if node_id:
            class_mappings[node_id] = cls_name
            if node_name:
                display_mappings[node_id] = node_name

    return class_mappings, display_mappings


def parse_file_classes(filepath: str, rel_path: str) -> dict[str, ast.ClassDef]:
    """Parse a Python file and return all class definitions."""
    try:
        with open(filepath, "r", encoding="utf-8") as f:
            source = f.read()
        tree = ast.parse(source, filename=rel_path)
        classes = {}
        for node in ast.walk(tree):
            if isinstance(node, ast.ClassDef):
                classes[node.name] = node
        return classes
    except (SyntaxError, OSError):
        return {}


def resolve_dotted_base(base_node: ast.AST) -> Optional[str]:
    """Resolve a dotted base class reference to module.ClassName form.

    E.g., comfy_extras.nodes_model_merging.ModelMergeBlocks
    """
    if isinstance(base_node, ast.Attribute):
        return ast_to_source(base_node)
    if isinstance(base_node, ast.Name):
        return base_node.id
    return None


def process_file(
    filepath: str,
    rel_path: str,
    pattern: str,
    external_classes: dict[str, ast.ClassDef],
) -> list[dict]:
    """Process a single Python file and extract all V1 nodes.

    external_classes: dict mapping class names (both short and dotted) to AST
    class nodes from other files, for resolving cross-file inheritance.
    """
    print(f"  Parsing {rel_path}...", file=sys.stderr)

    with open(filepath, "r", encoding="utf-8") as f:
        source = f.read()

    try:
        tree = ast.parse(source, filename=rel_path)
    except SyntaxError as e:
        warn(f"SyntaxError parsing {rel_path}: {e}")
        return []

    # Collect all class definitions (local + external for inheritance)
    all_classes: dict[str, ast.ClassDef] = {}
    all_classes.update(external_classes)
    for node in ast.walk(tree):
        if isinstance(node, ast.ClassDef):
            all_classes[node.name] = node

    # Also map dotted base class references. When a class inherits
    # comfy_extras.nodes_model_merging.ModelMergeBlocks, we need to resolve that
    # to the actual class node.
    for node in ast.walk(tree):
        if isinstance(node, ast.ClassDef):
            for base in node.bases:
                dotted = resolve_dotted_base(base)
                if dotted and "." in dotted:
                    # Extract the simple class name from the end
                    simple_name = dotted.rsplit(".", 1)[-1]
                    if simple_name in all_classes and dotted not in all_classes:
                        all_classes[dotted] = all_classes[simple_name]

    # Extract mappings
    class_mappings = extract_node_class_mappings(tree)
    display_mappings = extract_node_display_name_mappings(tree)

    # Handle nodes_hooks.py dynamic pattern
    is_hooks = "nodes_hooks" in rel_path
    if is_hooks:
        dyn_class, dyn_display = extract_hooks_dynamic_mappings(tree, all_classes)
        class_mappings.update(dyn_class)
        display_mappings.update(dyn_display)

    if not class_mappings:
        warn(f"No NODE_CLASS_MAPPINGS found in {rel_path}")
        return []

    nodes = []
    for node_id, class_name in class_mappings.items():
        if class_name not in all_classes:
            warn(f"{rel_path}: class '{class_name}' referenced in NODE_CLASS_MAPPINGS not found")
            continue

        cls_node = all_classes[class_name]
        info = extract_class_info(cls_node, all_classes)
        if info is None:
            warn(f"{rel_path}: Could not extract info from class '{class_name}' (missing FUNCTION or CATEGORY)")
            continue

        display_name = display_mappings.get(node_id, node_id)

        node_spec = {
            "node_id": node_id,
            "class_name": info["class_name"],
            "display_name": display_name,
            "description": info["description"],
            "category": info["category"],
            "function_name": info["function_name"],
            "is_output_node": info["is_output_node"],
            "is_deprecated": info["is_deprecated"],
            "is_experimental": info["is_experimental"],
            "inputs": info["inputs"],
            "hidden_inputs": info["hidden_inputs"],
            "outputs": info["outputs"],
            "source": {
                "file": rel_path,
                "pattern": pattern,
                "line_number": info["line_number"],
            },
            "terraform_resource_name": terraform_resource_name(node_id),
        }
        nodes.append(node_spec)

    return nodes


def main() -> None:
    if len(sys.argv) < 2:
        print("Usage: extract_v1_nodes.py <comfyui_root>", file=sys.stderr)
        sys.exit(1)

    comfyui_root = os.path.abspath(sys.argv[1])
    if not os.path.isdir(comfyui_root):
        print(f"Error: {comfyui_root} is not a directory", file=sys.stderr)
        sys.exit(1)

    print(f"Extracting V1 nodes from {comfyui_root}", file=sys.stderr)

    all_nodes = []
    file_counts = {}

    # Pre-parse files that serve as cross-file base classes.
    # These are referenced by dotted imports in child files.
    external_base_files = [
        "comfy_extras/nodes_model_merging.py",
    ]
    external_classes: dict[str, ast.ClassDef] = {}
    for rel_path in external_base_files:
        filepath = os.path.join(comfyui_root, rel_path)
        if os.path.isfile(filepath):
            classes = parse_file_classes(filepath, rel_path)
            external_classes.update(classes)

    # Process all files
    files_to_process = [CORE_FILE] + EXTRAS_FILES
    for rel_path, pattern in files_to_process:
        filepath = os.path.join(comfyui_root, rel_path)
        if not os.path.isfile(filepath):
            warn(f"File not found: {filepath}")
            continue

        nodes = process_file(filepath, rel_path, pattern, external_classes)
        file_counts[rel_path] = len(nodes)
        all_nodes.extend(nodes)

    # Sort by node_id for stable output
    all_nodes.sort(key=lambda n: n["node_id"])

    # Output JSON to stdout
    json.dump(all_nodes, sys.stdout, indent=2, default=str)
    sys.stdout.write("\n")

    # Print summary to stderr
    print(f"\n{'='*60}", file=sys.stderr)
    print(f"V1 Node Extraction Summary", file=sys.stderr)
    print(f"{'='*60}", file=sys.stderr)
    print(f"Total nodes extracted: {len(all_nodes)}", file=sys.stderr)
    print(f"\nPer-file counts:", file=sys.stderr)
    for rel_path, count in sorted(file_counts.items()):
        print(f"  {rel_path}: {count}", file=sys.stderr)

    # Count by category
    categories = {}
    for node in all_nodes:
        cat = node["category"]
        categories[cat] = categories.get(cat, 0) + 1
    print(f"\nBy category:", file=sys.stderr)
    for cat, count in sorted(categories.items()):
        print(f"  {cat}: {count}", file=sys.stderr)

    # Count special flags
    deprecated = sum(1 for n in all_nodes if n["is_deprecated"])
    experimental = sum(1 for n in all_nodes if n["is_experimental"])
    output_nodes = sum(1 for n in all_nodes if n["is_output_node"])
    print(f"\nSpecial flags:", file=sys.stderr)
    print(f"  Deprecated: {deprecated}", file=sys.stderr)
    print(f"  Experimental: {experimental}", file=sys.stderr)
    print(f"  Output nodes: {output_nodes}", file=sys.stderr)
    print(f"{'='*60}", file=sys.stderr)


if __name__ == "__main__":
    main()
