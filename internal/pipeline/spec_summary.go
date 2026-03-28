package pipeline

import (
	"encoding/json"
	"sort"
	"strings"

	apispec "github.com/mvanhorn/cli-printing-press/internal/spec"
)

type specSummary struct {
	Paths []string
	Auth  apispec.AuthConfig
}

func loadSpecSummary(specPath string) (*specSummary, error) {
	if specPath == "" {
		return nil, nil
	}

	data, err := readSpecBytes(specPath)
	if err != nil {
		return nil, err
	}

	if summary, err := summarizeOpenAPILike(data); err == nil && summary != nil {
		return summary, nil
	}

	if parsed, err := apispec.ParseBytes(data); err == nil {
		return summarizeSpec(parsed), nil
	}

	return nil, nil
}

func summarizeSpec(parsed *apispec.APISpec) *specSummary {
	if parsed == nil {
		return nil
	}

	paths := collectSpecPaths(parsed.Resources)
	sort.Strings(paths)

	return &specSummary{
		Paths: paths,
		Auth:  parsed.Auth,
	}
}

func collectSpecPaths(resources map[string]apispec.Resource) []string {
	if len(resources) == 0 {
		return nil
	}

	paths := make(map[string]struct{})
	var walk func(map[string]apispec.Resource)
	walk = func(resources map[string]apispec.Resource) {
		for _, resource := range resources {
			for _, endpoint := range resource.Endpoints {
				if endpoint.Path != "" {
					paths[endpoint.Path] = struct{}{}
				}
			}
			if len(resource.SubResources) > 0 {
				walk(resource.SubResources)
			}
		}
	}
	walk(resources)

	out := make([]string, 0, len(paths))
	for path := range paths {
		out = append(out, path)
	}
	return out
}

func summarizeOpenAPILike(data []byte) (*specSummary, error) {
	data, err := ensureJSON(data)
	if err != nil {
		return nil, err
	}

	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	pathsRaw, _ := raw["paths"].(map[string]any)
	paths := make([]string, 0, len(pathsRaw))
	for path := range pathsRaw {
		paths = append(paths, path)
	}
	sort.Strings(paths)

	auth := summarizeSecuritySchemes(raw)
	if len(paths) == 0 && auth.Type == "" && auth.Format == "" && auth.Header == "" && auth.Scheme == "" && auth.In == "" && len(auth.EnvVars) == 0 {
		return nil, nil
	}

	return &specSummary{
		Paths: paths,
		Auth:  auth,
	}, nil
}

func summarizeSecuritySchemes(raw map[string]any) apispec.AuthConfig {
	components, ok := raw["components"].(map[string]any)
	if !ok {
		return apispec.AuthConfig{}
	}
	schemes, ok := components["securitySchemes"].(map[string]any)
	if !ok {
		return apispec.AuthConfig{}
	}

	for schemeName, value := range schemes {
		scheme, ok := value.(map[string]any)
		if !ok {
			continue
		}

		auth := apispec.AuthConfig{
			Scheme: schemeName,
			Type:   stringValue(scheme["type"]),
			Header: stringValue(scheme["name"]),
			In:     stringValue(scheme["in"]),
		}
		schemeType := stringValue(scheme["scheme"])

		switch {
		case strings.EqualFold(auth.Type, "http") && strings.EqualFold(schemeType, "bearer"):
			auth.Type = "bearer_token"
			auth.Header = "Authorization"
			auth.Format = "Bearer "
			if strings.Contains(strings.ToLower(schemeName), "bot") {
				auth.Format = "Bot {bot_token}"
			}
		case strings.EqualFold(auth.Type, "http") && strings.EqualFold(schemeType, "basic"):
			auth.Type = "api_key"
			auth.Header = "Authorization"
			auth.Format = "Basic "
		case strings.EqualFold(auth.Type, "apikey"):
			if auth.Header == "" {
				auth.Header = "Authorization"
			}
			if strings.Contains(strings.ToLower(schemeName), "bot") && strings.EqualFold(auth.Header, "Authorization") {
				auth.Format = "Bot {bot_token}"
			}
		}

		return auth
	}

	return apispec.AuthConfig{}
}

func stringValue(value any) string {
	if s, ok := value.(string); ok {
		return s
	}
	return ""
}
