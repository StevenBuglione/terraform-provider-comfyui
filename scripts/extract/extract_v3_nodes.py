#!/usr/bin/env python3
"""Extract V3-pattern ComfyUI nodes from source code using AST parsing.

V3 nodes inherit from io.ComfyNode or IO.ComfyNode and use define_schema()
to declare their inputs, outputs, and metadata.

Usage:
    python3 extract_v3_nodes.py <comfyui_root>

Output:
    JSON array of node definitions to stdout.
    Summary statistics to stderr.
"""

import ast
import json
import os
import re
import sys
from pathlib import Path


# ---------------------------------------------------------------------------
# Type name mapping: Python class name -> ComfyUI io_type string
# Built from @comfytype(io_type=...) decorators in comfy_api/latest/_io.py
# ---------------------------------------------------------------------------
TYPE_NAME_MAP = {
    "Boolean": "BOOLEAN",
    "Int": "INT",
    "Float": "FLOAT",
    "String": "STRING",
    "Combo": "COMBO",
    "MultiCombo": "COMBO",
    "Image": "IMAGE",
    "WanCameraEmbedding": "WAN_CAMERA_EMBEDDING",
    "Webcam": "WEBCAM",
    "Mask": "MASK",
    "Latent": "LATENT",
    "Conditioning": "CONDITIONING",
    "Sampler": "SAMPLER",
    "Sigmas": "SIGMAS",
    "Noise": "NOISE",
    "Guider": "GUIDER",
    "Clip": "CLIP",
    "ControlNet": "CONTROL_NET",
    "Vae": "VAE",
    "Model": "MODEL",
    "ClipVision": "CLIP_VISION",
    "ClipVisionOutput": "CLIP_VISION_OUTPUT",
    "StyleModel": "STYLE_MODEL",
    "Gligen": "GLIGEN",
    "UpscaleModel": "UPSCALE_MODEL",
    "LatentUpscaleModel": "LATENT_UPSCALE_MODEL",
    "Audio": "AUDIO",
    "Video": "VIDEO",
    "SVG": "SVG",
    "LoraModel": "LORA_MODEL",
    "LossMap": "LOSS_MAP",
    "Voxel": "VOXEL",
    "Mesh": "MESH",
    "File3DAny": "FILE_3D",
    "File3DGLB": "FILE_3D_GLB",
    "File3DGLTF": "FILE_3D_GLTF",
    "File3DFBX": "FILE_3D_FBX",
    "File3DOBJ": "FILE_3D_OBJ",
    "File3DSTL": "FILE_3D_STL",
    "File3DUSDZ": "FILE_3D_USDZ",
    "Hooks": "HOOKS",
    "HookKeyframes": "HOOK_KEYFRAMES",
    "TimestepsRange": "TIMESTEPS_RANGE",
    "LatentOperation": "LATENT_OPERATION",
    "FlowControl": "FLOW_CONTROL",
    "Accumulation": "ACCUMULATION",
    "Load3DCamera": "LOAD3D_CAMERA",
    "Load3D": "LOAD_3D",
    "Load3DAnimation": "LOAD_3D_ANIMATION",
    "Photomaker": "PHOTOMAKER",
    "Point": "POINT",
    "FaceAnalysis": "FACE_ANALYSIS",
    "BBOX": "BBOX",
    "SEGS": "SEGS",
    "AnyType": "*",
    "ModelPatch": "MODEL_PATCH",
    "AudioEncoder": "AUDIO_ENCODER",
    "AudioEncoderOutput": "AUDIO_ENCODER_OUTPUT",
    "Tracks": "TRACKS",
    "MultiType": "COMFY_MULTITYPED_V3",
    "MatchType": "COMFY_MATCHTYPE_V3",
    "Autogrow": "COMFY_AUTOGROW_V3",
    "DynamicCombo": "COMFY_DYNAMICCOMBO_V3",
    "DynamicSlot": "COMFY_DYNAMICSLOT_V3",
    "ImageCompare": "IMAGECOMPARE",
    "Color": "COLOR",
    "BoundingBox": "BOUNDING_BOX",
    "Curve": "CURVE",
}

# Types that represent connectable link types (not widget types)
LINK_TYPES = {
    "IMAGE", "MASK", "LATENT", "MODEL", "CLIP", "VAE", "CONDITIONING",
    "CONTROL_NET", "CLIP_VISION", "CLIP_VISION_OUTPUT", "STYLE_MODEL",
    "GLIGEN", "UPSCALE_MODEL", "LATENT_UPSCALE_MODEL", "SAMPLER",
    "SIGMAS", "NOISE", "GUIDER", "AUDIO", "VIDEO", "SVG",
    "LORA_MODEL", "LOSS_MAP", "VOXEL", "MESH",
    "FILE_3D", "FILE_3D_GLB", "FILE_3D_GLTF", "FILE_3D_FBX",
    "FILE_3D_OBJ", "FILE_3D_STL", "FILE_3D_USDZ",
    "HOOKS", "HOOK_KEYFRAMES", "TIMESTEPS_RANGE", "LATENT_OPERATION",
    "FLOW_CONTROL", "ACCUMULATION", "LOAD3D_CAMERA", "LOAD_3D",
    "LOAD_3D_ANIMATION", "PHOTOMAKER", "POINT", "FACE_ANALYSIS",
    "BBOX", "SEGS", "*", "MODEL_PATCH",
    "AUDIO_ENCODER", "AUDIO_ENCODER_OUTPUT", "TRACKS",
    "WAN_CAMERA_EMBEDDING", "BOUNDING_BOX",
}

# Hidden input enum values
HIDDEN_VALUES = {
    "unique_id": "UNIQUE_ID",
    "prompt": "PROMPT",
    "extra_pnginfo": "EXTRA_PNGINFO",
    "dynprompt": "DYNPROMPT",
    "auth_token_comfy_org": "AUTH_TOKEN_COMFY_ORG",
    "api_key_comfy_org": "API_KEY_COMFY_ORG",
}


def warn(msg: str):
    print(f"  WARNING: {msg}", file=sys.stderr)


def to_snake_case(name: str) -> str:
    """Convert CamelCase/PascalCase to snake_case."""
    # Insert underscore before uppercase letters that follow lowercase/digits
    s = re.sub(r"([a-z0-9])([A-Z])", r"\1_\2", name)
    # Insert underscore between consecutive uppercase and uppercase+lowercase
    s = re.sub(r"([A-Z]+)([A-Z][a-z])", r"\1_\2", s)
    return s.lower()


def class_name_to_type(class_name: str) -> str:
    """Convert a Python io class name to its ComfyUI type string."""
    if class_name in TYPE_NAME_MAP:
        return TYPE_NAME_MAP[class_name]
    # Fallback: UPPER_SNAKE_CASE
    return to_snake_case(class_name).upper()


def is_link_type(type_name: str) -> bool:
    """Determine if a type is a link type (connectable) vs widget type."""
    if type_name in LINK_TYPES:
        return True
    # Custom types are generally link types
    if type_name not in ("INT", "FLOAT", "STRING", "BOOLEAN", "COMBO",
                         "COMFY_MULTITYPED_V3", "COMFY_MATCHTYPE_V3",
                         "COMFY_AUTOGROW_V3", "COMFY_DYNAMICCOMBO_V3",
                         "COMFY_DYNAMICSLOT_V3", "IMAGECOMPARE", "COLOR",
                         "CURVE", "WEBCAM"):
        return True
    return False


def ast_to_source(node: ast.AST) -> str:
    """Convert an AST node back to source string representation."""
    try:
        return ast.unparse(node)
    except Exception:
        return "<unknown>"


def extract_constant(node: ast.AST):
    """Extract a Python constant value from an AST node."""
    if isinstance(node, ast.Constant):
        return node.value
    if isinstance(node, ast.UnaryOp) and isinstance(node.op, ast.USub):
        val = extract_constant(node.operand)
        if val is not None:
            return -val
    if isinstance(node, ast.List):
        items = []
        for elt in node.elts:
            v = extract_constant(elt)
            if v is None:
                return None
            items.append(v)
        return items
    if isinstance(node, ast.Tuple):
        items = []
        for elt in node.elts:
            v = extract_constant(elt)
            if v is None:
                return None
            items.append(v)
        return tuple(items)
    if isinstance(node, (ast.Name, ast.Attribute, ast.Call, ast.BinOp,
                         ast.Subscript)):
        return None  # Dynamic/variable reference
    return None


def extract_attribute_chain(node: ast.AST) -> str | None:
    """Extract dotted attribute name, e.g. 'io.Model' -> 'Model'."""
    if isinstance(node, ast.Name):
        return node.id
    if isinstance(node, ast.Attribute):
        parent = extract_attribute_chain(node.value)
        if parent is not None:
            return f"{parent}.{node.attr}"
        return node.attr
    return None


def resolve_type_from_ast(node: ast.AST) -> tuple[str, str | None]:
    """Resolve the ComfyUI type from an AST expression like io.Model, IO.Custom(...), etc.

    Returns (type_name, custom_type_expr_or_None).
    """
    # Handle: io.Custom("SOME_TYPE") or IO.Custom(SomeVar.SOME_TYPE)
    if isinstance(node, ast.Call):
        func_name = extract_attribute_chain(node.func)
        if func_name and "Custom" in func_name:
            if node.args:
                arg = node.args[0]
                if isinstance(arg, ast.Constant) and isinstance(arg.value, str):
                    return (arg.value, arg.value)
                else:
                    expr = ast_to_source(arg)
                    return (expr, expr)
            return ("CUSTOM", "CUSTOM")
        # Fallback for other callable type constructors
        if func_name:
            parts = func_name.split(".")
            class_name = parts[-1]
            return (class_name_to_type(class_name), None)

    # Handle: io.Model, io.Int, IO.Image, etc.
    if isinstance(node, ast.Attribute):
        class_name = node.attr
        return (class_name_to_type(class_name), None)

    if isinstance(node, ast.Name):
        return (class_name_to_type(node.id), None)

    return (ast_to_source(node), None)


def parse_input_call(call_node: ast.Call, io_prefix_node: ast.AST) -> dict | None:
    """Parse an io.Type.Input(...) call into an input dict.

    call_node: the Call AST node for .Input(...)
    io_prefix_node: the AST for the type prefix (before .Input)
    """
    type_name, custom_type = resolve_type_from_ast(io_prefix_node)

    # First positional arg is the input name
    name = None
    if call_node.args:
        name_node = call_node.args[0]
        if isinstance(name_node, ast.Constant) and isinstance(name_node.value, str):
            name = name_node.value
        else:
            name = ast_to_source(name_node)

    if name is None:
        return None

    # Extract keyword arguments
    kwargs = {}
    for kw in call_node.keywords:
        if kw.arg is not None:
            kwargs[kw.arg] = kw.value

    default = None
    if "default" in kwargs:
        default = extract_constant(kwargs["default"])

    min_val = None
    if "min" in kwargs:
        min_val = extract_constant(kwargs["min"])

    max_val = None
    if "max" in kwargs:
        max_val = extract_constant(kwargs["max"])

    step = None
    if "step" in kwargs:
        step = extract_constant(kwargs["step"])

    tooltip = None
    if "tooltip" in kwargs:
        tooltip = extract_constant(kwargs["tooltip"])

    display_name = None
    if "display_name" in kwargs:
        display_name = extract_constant(kwargs["display_name"])

    multiline = None
    if "multiline" in kwargs:
        multiline = extract_constant(kwargs["multiline"])

    optional = False
    if "optional" in kwargs:
        opt_val = extract_constant(kwargs["optional"])
        if opt_val is True:
            optional = True

    # Options for Combo inputs
    options = None
    dynamic_options = None
    dynamic_options_source = None
    if "options" in kwargs:
        opt_node = kwargs["options"]
        const_opts = extract_constant(opt_node)
        if const_opts is not None and isinstance(const_opts, list):
            options = const_opts
        else:
            # Dynamic options - variable reference
            dynamic_options = True
            dynamic_options_source = ast_to_source(opt_node)

    required = not optional

    return {
        "name": name,
        "type": type_name,
        "required": required,
        "is_link_type": is_link_type(type_name),
        "default": default,
        "min": min_val,
        "max": max_val,
        "step": step,
        "options": options,
        "multiline": multiline,
        "dynamic_options": dynamic_options,
        "dynamic_options_source": dynamic_options_source,
        "tooltip": tooltip,
        "display_name": display_name,
    }


def parse_output_call(call_node: ast.Call, io_prefix_node: ast.AST,
                      slot_index: int) -> dict:
    """Parse an io.Type.Output(...) call into an output dict."""
    type_name, _ = resolve_type_from_ast(io_prefix_node)

    # Extract keyword args
    display_name = None
    for kw in call_node.keywords:
        if kw.arg == "display_name":
            display_name = extract_constant(kw.value)

    # Positional arg can also be a display name for outputs
    if call_node.args and display_name is None:
        first_arg = call_node.args[0]
        if isinstance(first_arg, ast.Constant) and isinstance(first_arg.value, str):
            display_name = first_arg.value

    name = display_name or type_name

    return {
        "name": name,
        "type": type_name,
        "slot_index": slot_index,
        "is_list": False,
    }


def find_input_output_calls(node: ast.AST):
    """Walk an AST node to find .Input(...) and .Output(...) calls.

    Yields tuples of (kind, call_node, type_prefix_node) where:
      kind is "Input" or "Output"
      call_node is the Call AST node
      type_prefix_node is the AST for the type prefix
    """
    if isinstance(node, ast.Call):
        func = node.func
        if isinstance(func, ast.Attribute) and func.attr in ("Input", "Output"):
            yield (func.attr, node, func.value)
            return
    for child in ast.iter_child_nodes(node):
        yield from find_input_output_calls(child)


def parse_schema_call(call_node: ast.Call, file_path: str,
                      class_name: str) -> dict | None:
    """Parse a Schema(...) call to extract node metadata, inputs, and outputs."""
    kwargs = {}
    for kw in call_node.keywords:
        if kw.arg is not None:
            kwargs[kw.arg] = kw.value

    # Extract simple string/bool fields
    node_id = None
    if "node_id" in kwargs:
        node_id = extract_constant(kwargs["node_id"])

    display_name = None
    if "display_name" in kwargs:
        display_name = extract_constant(kwargs["display_name"])

    category = None
    if "category" in kwargs:
        category = extract_constant(kwargs["category"])

    description = None
    if "description" in kwargs:
        description = extract_constant(kwargs["description"])

    is_output_node = False
    if "is_output_node" in kwargs:
        v = extract_constant(kwargs["is_output_node"])
        if v is True:
            is_output_node = True
    if "output_node" in kwargs:
        v = extract_constant(kwargs["output_node"])
        if v is True:
            is_output_node = True

    is_deprecated = False
    if "is_deprecated" in kwargs:
        v = extract_constant(kwargs["is_deprecated"])
        if v is True:
            is_deprecated = True

    is_experimental = False
    if "is_experimental" in kwargs:
        v = extract_constant(kwargs["is_experimental"])
        if v is True:
            is_experimental = True

    # Parse inputs list
    inputs = []
    hidden_inputs = []
    if "inputs" in kwargs:
        inputs_node = kwargs["inputs"]
        if isinstance(inputs_node, ast.List):
            for elt in inputs_node.elts:
                parsed = _parse_io_element(elt, "Input", file_path, class_name)
                if parsed is not None:
                    inputs.append(parsed)
        else:
            # Could be a variable reference; walk it for Input calls
            for kind, call, prefix in find_input_output_calls(inputs_node):
                if kind == "Input":
                    parsed = parse_input_call(call, prefix)
                    if parsed is not None:
                        inputs.append(parsed)

    # Parse outputs list
    outputs = []
    if "outputs" in kwargs:
        outputs_node = kwargs["outputs"]
        if isinstance(outputs_node, ast.List):
            for idx, elt in enumerate(outputs_node.elts):
                parsed = _parse_io_element_output(elt, idx, file_path,
                                                  class_name)
                if parsed is not None:
                    outputs.append(parsed)
        else:
            for kind, call, prefix in find_input_output_calls(outputs_node):
                if kind == "Output":
                    parsed = parse_output_call(call, prefix, len(outputs))
                    if parsed is not None:
                        outputs.append(parsed)

    # Parse hidden inputs
    if "hidden" in kwargs:
        hidden_node = kwargs["hidden"]
        if isinstance(hidden_node, ast.List):
            for elt in hidden_node.elts:
                hi = _parse_hidden_input(elt)
                if hi is not None:
                    hidden_inputs.append(hi)

    return {
        "node_id": node_id,
        "display_name": display_name,
        "category": category,
        "description": description,
        "is_output_node": is_output_node,
        "is_deprecated": is_deprecated,
        "is_experimental": is_experimental,
        "inputs": inputs,
        "outputs": outputs,
        "hidden_inputs": hidden_inputs,
    }


def _parse_io_element(elt: ast.AST, kind: str, file_path: str,
                      class_name: str) -> dict | None:
    """Parse a single element from the inputs list in Schema."""
    # Direct .Input(...) call: io.Type.Input(...)
    if isinstance(elt, ast.Call):
        func = elt.func
        if isinstance(func, ast.Attribute) and func.attr == "Input":
            return parse_input_call(elt, func.value)
        # Could be a variable call - try to parse anyway
        # e.g. crop_combo which is defined as io.Combo.Input(...) elsewhere
        # We can't resolve variables, so emit a warning
        func_name = extract_attribute_chain(func) if isinstance(func, (ast.Attribute, ast.Name)) else None
        if func_name and "Input" in func_name:
            return parse_input_call(elt, func.value)
    # Might be a Name reference to a variable defined outside
    if isinstance(elt, ast.Name):
        return {
            "name": elt.id,
            "type": "UNKNOWN",
            "required": True,
            "is_link_type": False,
            "default": None, "min": None, "max": None, "step": None,
            "options": None, "multiline": None,
            "dynamic_options": None,
            "dynamic_options_source": f"variable:{elt.id}",
            "tooltip": None, "display_name": None,
        }
    return None


def _parse_io_element_output(elt: ast.AST, idx: int, file_path: str,
                             class_name: str) -> dict | None:
    """Parse a single element from the outputs list in Schema."""
    if isinstance(elt, ast.Call):
        func = elt.func
        if isinstance(func, ast.Attribute) and func.attr == "Output":
            return parse_output_call(elt, func.value, idx)
        func_name = extract_attribute_chain(func) if isinstance(func, (ast.Attribute, ast.Name)) else None
        if func_name and "Output" in func_name:
            return parse_output_call(elt, func.value, idx)
    return None


def _parse_hidden_input(elt: ast.AST) -> dict | None:
    """Parse a hidden input like io.Hidden.prompt."""
    if isinstance(elt, ast.Attribute):
        attr_name = elt.attr
        if attr_name in HIDDEN_VALUES:
            return {
                "name": attr_name,
                "type": HIDDEN_VALUES[attr_name],
            }
        return {
            "name": attr_name,
            "type": attr_name.upper(),
        }
    return None


def find_schema_call(func_body: list[ast.stmt]) -> ast.Call | None:
    """Find the Schema(...) call in a define_schema method body.

    Handles:
      return io.Schema(...)
      return IO.Schema(...)
      schema = io.Schema(...); return schema
    """
    # Walk all statements looking for Schema(...) calls
    for stmt in func_body:
        if isinstance(stmt, ast.Return) and stmt.value is not None:
            call = _find_schema_in_expr(stmt.value)
            if call is not None:
                return call
        # Assignment: schema = io.Schema(...)
        if isinstance(stmt, ast.Assign):
            if isinstance(stmt.value, ast.Call):
                call = _find_schema_in_expr(stmt.value)
                if call is not None:
                    return call
    return None


def _find_schema_in_expr(node: ast.AST) -> ast.Call | None:
    """Find Schema(...) call within an expression."""
    if isinstance(node, ast.Call):
        func_name = extract_attribute_chain(node.func)
        if func_name and "Schema" in func_name:
            return node
    return None


def has_execute_method(class_node: ast.ClassDef) -> bool:
    """Check if a class has an execute method."""
    for item in class_node.body:
        if isinstance(item, ast.FunctionDef) and item.name == "execute":
            return True
        if isinstance(item, ast.AsyncFunctionDef) and item.name == "execute":
            return True
    return False


def is_comfy_node_class(class_node: ast.ClassDef) -> bool:
    """Check if a class inherits from ComfyNode (io.ComfyNode or IO.ComfyNode)."""
    for base in class_node.bases:
        base_name = extract_attribute_chain(base)
        if base_name and "ComfyNode" in base_name:
            return True
    return False


def extract_nodes_from_file(file_path: str, comfyui_root: str,
                            pattern: str) -> list[dict]:
    """Extract all V3 nodes from a single Python file."""
    try:
        with open(file_path, "r", encoding="utf-8") as f:
            source = f.read()
    except Exception as e:
        warn(f"Could not read {file_path}: {e}")
        return []

    try:
        tree = ast.parse(source, filename=file_path)
    except SyntaxError as e:
        warn(f"Syntax error in {file_path}: {e}")
        return []

    rel_path = os.path.relpath(file_path, comfyui_root)
    nodes = []

    for node in ast.walk(tree):
        if not isinstance(node, ast.ClassDef):
            continue
        if not is_comfy_node_class(node):
            continue

        class_name = node.name
        line_number = node.lineno

        # Find define_schema method
        schema_data = None
        for item in node.body:
            if isinstance(item, ast.FunctionDef) and item.name == "define_schema":
                schema_call = find_schema_call(item.body)
                if schema_call is not None:
                    schema_data = parse_schema_call(schema_call, file_path,
                                                    class_name)
                break

        if schema_data is None:
            warn(f"No parseable define_schema in {class_name} "
                 f"({rel_path}:{line_number})")
            continue

        node_id = schema_data["node_id"] or class_name
        function_name = "execute" if has_execute_method(node) else "execute"

        terraform_name = "comfyui_" + to_snake_case(node_id)

        nodes.append({
            "node_id": node_id,
            "class_name": class_name,
            "display_name": schema_data["display_name"],
            "description": schema_data["description"],
            "category": schema_data["category"],
            "function_name": function_name,
            "is_output_node": schema_data["is_output_node"],
            "is_deprecated": schema_data["is_deprecated"],
            "is_experimental": schema_data["is_experimental"],
            "inputs": schema_data["inputs"],
            "hidden_inputs": schema_data["hidden_inputs"],
            "outputs": schema_data["outputs"],
            "source": {
                "file": rel_path,
                "pattern": pattern,
                "line_number": line_number,
            },
            "terraform_resource_name": terraform_name,
        })

    return nodes


def find_v3_files(comfyui_root: str) -> list[tuple[str, str]]:
    """Find all files containing V3 node classes.

    Returns list of (file_path, pattern) tuples.
    """
    files = []

    # Scan comfy_extras/
    extras_dir = os.path.join(comfyui_root, "comfy_extras")
    if os.path.isdir(extras_dir):
        for fname in sorted(os.listdir(extras_dir)):
            if not fname.endswith(".py"):
                continue
            fpath = os.path.join(extras_dir, fname)
            try:
                with open(fpath, "r", encoding="utf-8") as f:
                    content = f.read(8192)
                if re.search(r"class\s+\w+\s*\(\s*(?:io|IO)\.ComfyNode\s*\)",
                             content):
                    files.append((fpath, "v3_extras"))
            except Exception:
                pass

    # Scan comfy_api_nodes/
    api_dir = os.path.join(comfyui_root, "comfy_api_nodes")
    if os.path.isdir(api_dir):
        for fname in sorted(os.listdir(api_dir)):
            if not fname.endswith(".py"):
                continue
            fpath = os.path.join(api_dir, fname)
            try:
                with open(fpath, "r", encoding="utf-8") as f:
                    content = f.read(8192)
                if re.search(r"class\s+\w+\s*\(\s*(?:io|IO)\.ComfyNode\s*\)",
                             content):
                    files.append((fpath, "v3_api"))
            except Exception:
                pass

    return files


def find_v3_files_recursive(directory: str, pattern: str,
                            comfyui_root: str) -> list[tuple[str, str]]:
    """Recursively find V3 node files in a directory."""
    files = []
    if not os.path.isdir(directory):
        return files
    for root, _dirs, fnames in os.walk(directory):
        for fname in sorted(fnames):
            if not fname.endswith(".py"):
                continue
            fpath = os.path.join(root, fname)
            try:
                with open(fpath, "r", encoding="utf-8") as f:
                    content = f.read()
                if re.search(
                    r"class\s+\w+\s*\(\s*(?:io|IO)\.ComfyNode\s*\)",
                    content
                ):
                    files.append((fpath, pattern))
            except Exception:
                pass
    return files


def main():
    if len(sys.argv) < 2:
        print("Usage: extract_v3_nodes.py <comfyui_root>", file=sys.stderr)
        sys.exit(1)

    comfyui_root = os.path.abspath(sys.argv[1])
    if not os.path.isdir(comfyui_root):
        print(f"Error: {comfyui_root} is not a directory", file=sys.stderr)
        sys.exit(1)

    print(f"Scanning {comfyui_root} for V3 nodes...", file=sys.stderr)

    # Find files: use recursive scan to catch all files (including subdirs)
    files = []
    extras_dir = os.path.join(comfyui_root, "comfy_extras")
    api_dir = os.path.join(comfyui_root, "comfy_api_nodes")

    files.extend(find_v3_files_recursive(extras_dir, "v3_extras", comfyui_root))
    files.extend(find_v3_files_recursive(api_dir, "v3_api", comfyui_root))

    print(f"Found {len(files)} files with V3 nodes", file=sys.stderr)

    all_nodes = []
    file_counts = {}
    warnings_count = 0

    for fpath, pattern in files:
        rel = os.path.relpath(fpath, comfyui_root)
        nodes = extract_nodes_from_file(fpath, comfyui_root, pattern)
        if nodes:
            file_counts[rel] = len(nodes)
            all_nodes.extend(nodes)

    # Summary to stderr
    extras_count = sum(1 for n in all_nodes
                       if n["source"]["pattern"] == "v3_extras")
    api_count = sum(1 for n in all_nodes
                    if n["source"]["pattern"] == "v3_api")

    print(f"\n{'='*60}", file=sys.stderr)
    print(f"V3 Node Extraction Summary", file=sys.stderr)
    print(f"{'='*60}", file=sys.stderr)
    print(f"Total nodes extracted: {len(all_nodes)}", file=sys.stderr)
    print(f"  V3 extras (comfy_extras): {extras_count}", file=sys.stderr)
    print(f"  V3 API (comfy_api_nodes): {api_count}", file=sys.stderr)
    print(f"Files processed: {len(files)}", file=sys.stderr)
    print(f"{'='*60}", file=sys.stderr)

    # Per-file counts
    print(f"\nPer-file counts:", file=sys.stderr)
    for fpath, count in sorted(file_counts.items()):
        print(f"  {fpath}: {count}", file=sys.stderr)

    # Count nodes with various features
    output_nodes = sum(1 for n in all_nodes if n["is_output_node"])
    deprecated = sum(1 for n in all_nodes if n["is_deprecated"])
    experimental = sum(1 for n in all_nodes if n["is_experimental"])
    with_hidden = sum(1 for n in all_nodes if n["hidden_inputs"])

    print(f"\nFeature breakdown:", file=sys.stderr)
    print(f"  Output nodes: {output_nodes}", file=sys.stderr)
    print(f"  Deprecated: {deprecated}", file=sys.stderr)
    print(f"  Experimental: {experimental}", file=sys.stderr)
    print(f"  With hidden inputs: {with_hidden}", file=sys.stderr)
    print(f"{'='*60}", file=sys.stderr)

    # Output JSON to stdout
    json.dump(all_nodes, sys.stdout, indent=2, ensure_ascii=False)
    print("", file=sys.stdout)  # trailing newline


if __name__ == "__main__":
    main()
