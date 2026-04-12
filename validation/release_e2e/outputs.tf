locals {
  scenario_payloads = {
    assembled_resource  = data.comfyui_prompt_to_workspace.assembled_resource.workspace_json
    raw_import          = data.comfyui_prompt_to_workspace.raw_import.workspace_json
    assembled_roundtrip = data.comfyui_prompt_to_workspace.assembled_roundtrip.workspace_json
    release_gallery     = comfyui_workspace.release_gallery.workspace_json
  }

  scenario_require_groups = {
    assembled_resource  = false
    raw_import          = false
    assembled_roundtrip = false
    release_gallery     = true
  }

  scenario_expectations = {
    for name, payload in local.scenario_payloads : name => {
      workspace_name = name
      node_count     = length(try(jsondecode(payload).nodes, []))
      group_count    = length(try(jsondecode(payload).groups, []))
      link_count     = length(try(jsondecode(payload).links, []))
      max_in_degree = max([
        for node in try(jsondecode(payload).nodes, []) : sum(concat([0], [
          for input in try(node.inputs, []) : try(input.link, null) != null ? 1 : 0
        ]))
      ]...)
      max_out_degree = max([
        for node in try(jsondecode(payload).nodes, []) : sum(concat([0], [
          for output in try(node.outputs, []) : length(try(output.links, []))
        ]))
      ]...)
      require_groups = local.scenario_require_groups[name]
    }
  }
}

output "scenario_payloads" {
  value = local.scenario_payloads
}

output "scenario_expectations" {
  value = local.scenario_expectations
}

output "translation_assertions" {
  value = {
    assembled_prompt_nodes           = data.comfyui_prompt_json.assembled_resource.node_count
    assembled_roundtrip_prompt_nodes = data.comfyui_prompt_json.assembled_roundtrip.node_count
    assembled_workspace_fidelity     = data.comfyui_prompt_to_workspace.assembled_resource.fidelity
    raw_import_workspace_fidelity    = data.comfyui_prompt_to_workspace.raw_import.fidelity
    roundtrip_prompt_fidelity        = data.comfyui_workspace_to_prompt.assembled_roundtrip.fidelity
    roundtrip_workspace_fidelity     = data.comfyui_prompt_to_workspace.assembled_roundtrip.fidelity
  }
}
