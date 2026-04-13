package generator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mvanhorn/cli-printing-press/internal/spec"
	"github.com/stretchr/testify/require"
)

// TestEnumParamEmitsValidation ensures that params declared with enum constraints
// cause the generated command to (a) emit runtime validation that warns on
// unknown values and (b) include a "(one of: ...)" hint in the flag description.
// Regression guard for #205.
func TestEnumParamEmitsValidation(t *testing.T) {
	t.Parallel()

	apiSpec := minimalSpec("enum-test")
	// Two endpoints so sibling "search" renders to its own file rather than
	// getting consolidated into the promoted parent. The enum check fires in
	// either file path, but `widgets_search.go` is easier to assert against.
	apiSpec.Resources["widgets"] = spec.Resource{
		Description: "Widgets",
		Endpoints: map[string]spec.Endpoint{
			"list": {
				Method:      "GET",
				Path:        "/widgets",
				Description: "List widgets",
			},
			"search": {
				Method:      "GET",
				Path:        "/widgets/search",
				Description: "Search widgets filtered by status",
				Params: []spec.Param{
					{
						Name:        "status",
						Type:        "string",
						Required:    false,
						Description: "Widget status",
						Enum:        []string{"active", "archived", "pending"},
					},
				},
			},
		},
	}

	outputDir := filepath.Join(t.TempDir(), "enum-test-pp-cli")
	require.NoError(t, New(apiSpec, outputDir).Generate())

	src, err := os.ReadFile(filepath.Join(outputDir, "internal", "cli", "widgets_search.go"))
	require.NoError(t, err)
	code := string(src)

	// Flag description includes the enum hint.
	require.Contains(t, code, `(one of: active, archived, pending)`,
		"flag description must include enum values")

	// Runtime validation block emitted.
	require.Contains(t, code, `allowedStatus := []string{ "active", "archived", "pending" }`,
		"runtime validation must declare the allowed set")
	require.Contains(t, code, `warning: --%s %q not in allowed set %v`,
		"runtime validation must warn on unknown value")
}

// TestNonEnumParamDoesNotEmitValidation ensures the enum block is gated
// on the Enum slice being non-empty — plain params stay untouched.
func TestNonEnumParamDoesNotEmitValidation(t *testing.T) {
	t.Parallel()

	apiSpec := minimalSpec("no-enum")
	apiSpec.Resources["items"] = spec.Resource{
		Description: "Items",
		Endpoints: map[string]spec.Endpoint{
			"list": {
				Method:      "GET",
				Path:        "/items",
				Description: "List items",
			},
			"search": {
				Method:      "GET",
				Path:        "/items/search",
				Description: "Search items",
				Params: []spec.Param{
					{Name: "query", Type: "string", Required: false, Description: "Search query"},
				},
			},
		},
	}

	outputDir := filepath.Join(t.TempDir(), "no-enum-pp-cli")
	require.NoError(t, New(apiSpec, outputDir).Generate())

	src, err := os.ReadFile(filepath.Join(outputDir, "internal", "cli", "items_search.go"))
	require.NoError(t, err)
	code := string(src)

	require.NotContains(t, code, `allowedQuery`,
		"params without Enum must not emit validation code")
	require.NotContains(t, code, `(one of:`,
		"params without Enum must not get a description hint")
}
