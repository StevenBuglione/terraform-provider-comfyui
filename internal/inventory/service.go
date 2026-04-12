package inventory

import (
	"context"
	"fmt"
	"sort"

	"github.com/StevenBuglione/terraform-provider-comfyui/internal/client"
	"github.com/StevenBuglione/terraform-provider-comfyui/internal/nodeschema"
)

type objectInfoReader interface {
	GetObjectInfo() (map[string]client.NodeInfo, error)
}

type representative struct {
	NodeType  string
	InputName string
}

type Service struct {
	reader           objectInfoReader
	objectInfo       map[string]client.NodeInfo
	objectInfoLoaded bool
	cache            map[Kind][]string
}

func NewService(reader objectInfoReader) *Service {
	return &Service{
		reader: reader,
		cache:  map[Kind][]string{},
	}
}

func (s *Service) GetInventory(ctx context.Context, kind Kind) ([]string, error) {
	_ = ctx
	if values, ok := s.cache[kind]; ok {
		return append([]string(nil), values...), nil
	}

	if !s.objectInfoLoaded {
		if s.reader == nil {
			return nil, fmt.Errorf("inventory lookup requires a configured ComfyUI client")
		}
		info, err := s.reader.GetObjectInfo()
		if err != nil {
			return nil, fmt.Errorf("failed to read live ComfyUI object_info for inventory %q: %w", kind, err)
		}
		s.objectInfo = info
		s.objectInfoLoaded = true
	}

	rep, ok := representativeForKind(kind, s.objectInfo)
	if !ok {
		return nil, fmt.Errorf("unsupported inventory kind %q", kind)
	}

	nodeInfo, ok := s.objectInfo[rep.NodeType]
	if !ok {
		return nil, fmt.Errorf("live ComfyUI object_info is missing representative node %q for inventory %q", rep.NodeType, kind)
	}

	values, err := extractInputOptions(nodeInfo, rep.InputName)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve live inventory %q from %s.%s: %w", kind, rep.NodeType, rep.InputName, err)
	}
	sort.Strings(values)
	s.cache[kind] = append([]string(nil), values...)
	return append([]string(nil), values...), nil
}

func representativeForKind(kind Kind, objectInfo map[string]client.NodeInfo) (representative, bool) {
	candidates := make([]representative, 0)
	for _, schema := range nodeschema.All() {
		for _, input := range schema.RequiredInputs {
			if input.InventoryKind == string(kind) && input.SupportsStrictPlanValidation {
				candidates = append(candidates, representative{NodeType: schema.NodeType, InputName: input.Name})
			}
		}
		for _, input := range schema.OptionalInputs {
			if input.InventoryKind == string(kind) && input.SupportsStrictPlanValidation {
				candidates = append(candidates, representative{NodeType: schema.NodeType, InputName: input.Name})
			}
		}
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].NodeType == candidates[j].NodeType {
			return candidates[i].InputName < candidates[j].InputName
		}
		return candidates[i].NodeType < candidates[j].NodeType
	})
	for _, candidate := range candidates {
		if _, ok := objectInfo[candidate.NodeType]; ok {
			return candidate, true
		}
	}
	return representative{}, false
}

func extractInputOptions(info client.NodeInfo, inputName string) ([]string, error) {
	for _, section := range []map[string]interface{}{info.Input.Required, info.Input.Optional} {
		raw, ok := section[inputName]
		if !ok {
			continue
		}
		values := optionsFromDefinition(raw)
		return values, nil
	}
	return nil, fmt.Errorf("input %q not found in live object_info", inputName)
}

func optionsFromDefinition(raw interface{}) []string {
	def, ok := raw.([]interface{})
	if !ok || len(def) == 0 {
		return nil
	}

	switch first := def[0].(type) {
	case []interface{}:
		return interfaceSliceToStrings(first)
	case []string:
		return append([]string(nil), first...)
	}

	if len(def) > 1 {
		if meta, ok := def[1].(map[string]interface{}); ok {
			switch options := meta["options"].(type) {
			case []interface{}:
				return interfaceSliceToStrings(options)
			case []string:
				return append([]string(nil), options...)
			}
		}
	}

	return nil
}

func interfaceSliceToStrings(values []interface{}) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value == nil {
			continue
		}
		result = append(result, fmt.Sprintf("%v", value))
	}
	return result
}
