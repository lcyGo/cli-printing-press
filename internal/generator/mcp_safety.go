package generator

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/mvanhorn/cli-printing-press/v3/internal/spec"
)

const (
	mcpReadOnlyAnnotation         = "mcp:read-only"
	mcpDestructiveAnnotation      = "mcp:destructive"
	mcpPrivacySensitiveAnnotation = "mcp:privacy-sensitive"
	ppEndpointAnnotation          = "pp:endpoint"
	ppMethodAnnotation            = "pp:method"
	ppPathAnnotation              = "pp:path"
)

const unannotatedMutationWarning = "warning: command %s is an unannotated mutation; agents will not see destructive signal; annotate explicitly with mcp:destructive=true when the action is destructive.\n"

func commandAnnotationsLiteral(resourceName, endpointName, path string, ep spec.Endpoint, isReadOnly bool) string {
	method := strings.ToUpper(strings.TrimSpace(ep.Method))
	destructiveMeta := spec.EndpointMetaTrue(ep, mcpDestructiveAnnotation)
	parts := []string{
		fmt.Sprintf("%q: %q", ppEndpointAnnotation, resourceName+"."+endpointName),
		fmt.Sprintf("%q: %q", ppMethodAnnotation, method),
		fmt.Sprintf("%q: %q", ppPathAnnotation, path),
	}
	if isReadOnly && !destructiveMeta {
		parts = append(parts, fmt.Sprintf("%q: %q", mcpReadOnlyAnnotation, "true"))
	}
	if destructiveMeta || (!isReadOnly && method == "DELETE") {
		parts = append(parts, fmt.Sprintf("%q: %q", mcpDestructiveAnnotation, "true"))
	}
	if spec.EndpointMetaTrue(ep, mcpPrivacySensitiveAnnotation) {
		parts = append(parts, fmt.Sprintf("%q: %q", mcpPrivacySensitiveAnnotation, "true"))
	}
	return "map[string]string{" + strings.Join(parts, ", ") + "}"
}

func warnUnannotatedMutations(s *spec.APISpec, w io.Writer) {
	if s == nil || w == nil {
		return
	}
	for _, warning := range collectMutationWarnings(s) {
		fmt.Fprintf(w, unannotatedMutationWarning, warning)
	}
}

func collectMutationWarnings(s *spec.APISpec) []string {
	if s == nil {
		return nil
	}
	var warnings []string
	for resourceName, resource := range s.Resources {
		for endpointName, endpoint := range resource.Endpoints {
			if endpointNeedsMutationWarning(endpoint, endpointName) {
				warnings = append(warnings, formatMutationWarningCommand(resourceName, "", endpointName))
			}
		}
		for subResourceName, subResource := range resource.SubResources {
			for endpointName, endpoint := range subResource.Endpoints {
				if endpointNeedsMutationWarning(endpoint, endpointName) {
					warnings = append(warnings, formatMutationWarningCommand(resourceName, subResourceName, endpointName))
				}
			}
		}
	}
	sort.Strings(warnings)
	return warnings
}

func endpointNeedsMutationWarning(endpoint spec.Endpoint, opName string) bool {
	if !isNeutralMutationMethod(endpoint.Method) {
		return false
	}
	if spec.EndpointMetaTrue(endpoint, mcpDestructiveAnnotation) {
		return false
	}
	return endpointIsWriteCommand(endpoint, opName)
}

func isNeutralMutationMethod(method string) bool {
	switch strings.ToUpper(strings.TrimSpace(method)) {
	case "POST", "PUT", "PATCH":
		return true
	default:
		return false
	}
}

func formatMutationWarningCommand(resourceName, subResourceName, endpointName string) string {
	parts := []string{toKebab(resourceName)}
	if subResourceName != "" {
		parts = append(parts, toKebab(subResourceName))
	}
	parts = append(parts, toKebab(endpointName))
	return strings.Join(parts, " ")
}
