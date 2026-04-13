package resources

type AuthResolverConfig struct {
	ComfyOrgAuthToken string
	ComfyOrgAPIKey    string
}

type ResolvedAuthRequirement struct {
	Family          string
	Mode            string
	RequiredFields  []string
	TriggeringNodes []string
}

type AuthResolution struct {
	Resolved    []ResolvedAuthRequirement
	Unsatisfied []AuthRequirement
	ExtraData   map[string]interface{}
}

func ResolveAuthRequirements(requirements []AuthRequirement, config AuthResolverConfig) AuthResolution {
	resolution := AuthResolution{
		ExtraData: map[string]interface{}{},
	}

	for _, requirement := range requirements {
		switch requirement.Family {
		case "comfy_org":
			if config.ComfyOrgAPIKey != "" {
				resolution.ExtraData["api_key_comfy_org"] = config.ComfyOrgAPIKey
				resolution.Resolved = append(resolution.Resolved, ResolvedAuthRequirement{
					Family:          requirement.Family,
					Mode:            "api_key",
					RequiredFields:  append([]string(nil), requirement.RequiredFields...),
					TriggeringNodes: append([]string(nil), requirement.TriggeringNodes...),
				})
				continue
			}

			if config.ComfyOrgAuthToken != "" {
				resolution.ExtraData["auth_token_comfy_org"] = config.ComfyOrgAuthToken
				resolution.Resolved = append(resolution.Resolved, ResolvedAuthRequirement{
					Family:          requirement.Family,
					Mode:            "auth_token",
					RequiredFields:  append([]string(nil), requirement.RequiredFields...),
					TriggeringNodes: append([]string(nil), requirement.TriggeringNodes...),
				})
				continue
			}
		}

		resolution.Unsatisfied = append(resolution.Unsatisfied, requirement)
	}

	return resolution
}
