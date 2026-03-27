package pipeline

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// leafCmd is a minimal cobra command fixture with RunE (makes it a leaf command).
const leafCmdPrefix = `
package cli

var cmdX = &cobra.Command{
	RunE: func(cmd *cobra.Command, args []string) error { return nil },
`

func TestScoreAgentNative(t *testing.T) {
	t.Run("scores zero for empty directory", func(t *testing.T) {
		dir := t.TempDir()
		assert.Equal(t, 0, scoreAgentNative(dir))
	})

	t.Run("scores core flags only", func(t *testing.T) {
		dir := t.TempDir()
		writeScorecardFixture(t, dir, "internal/cli/root.go", `
package cli

func init() {
	rootCmd.PersistentFlags().StringVar(&flags.outputFormat, "json", "", "JSON output")
	rootCmd.PersistentFlags().StringVar(&flags.selectFields, "select", "", "Select fields")
	rootCmd.PersistentFlags().BoolVar(&flags.dryRun, "dry-run", false, "Dry run")
	rootCmd.PersistentFlags().BoolVar(&flags.stdin, "stdin", false, "Read from stdin")
	rootCmd.PersistentFlags().BoolVar(&flags.yes, "yes", false, "Skip confirmation")
}
`)
		writeScorecardFixture(t, dir, "internal/cli/helpers.go", `
package cli
`)
		assert.Equal(t, 5, scoreAgentNative(dir))
	})

	t.Run("scores max with all checks passing", func(t *testing.T) {
		dir := t.TempDir()
		writeScorecardFixture(t, dir, "internal/cli/root.go", `
package cli

func init() {
	rootCmd.PersistentFlags().StringVar(&flags.outputFormat, "json", "", "JSON output")
	rootCmd.PersistentFlags().StringVar(&flags.selectFields, "select", "", "Select fields")
	rootCmd.PersistentFlags().BoolVar(&flags.dryRun, "dry-run", false, "Dry run")
	rootCmd.PersistentFlags().BoolVar(&flags.stdin, "stdin", false, "Read from stdin")
	rootCmd.PersistentFlags().BoolVar(&flags.yes, "yes", false, "Skip confirmation")
	rootCmd.PersistentFlags().IntVar(&flags.limit, "limit", 100, "Max results")
}

const defaultLimit = 100
`)
		writeScorecardFixture(t, dir, "internal/cli/helpers.go", `
package cli

func handleError(err error) {
	hint := "try running the doctor command"
	hint2 := "use --json for machine-readable output"
	hint3 := "run 'myapp auth login' to authenticate"
	hint4 := "check your network connection"
	_ = hint
	_ = hint2
	_ = hint3
	_ = hint4
}

var exitCodes = map[string]int{
	"code: 1": 1,
	"code: 2": 2,
	"code: 3": 3,
}
`)
		// Create 5 leaf command files with valid examples and flag declarations
		for _, name := range []string{"list_users.go", "get_user.go", "create_user.go", "delete_user.go", "update_user.go"} {
			writeScorecardFixture(t, dir, "internal/cli/"+name, `
package cli

func init() {
	cmd.Flags().StringVar(&flagID, "id", "", "Resource ID")
	cmd.Flags().BoolVar(&flagVerbose, "verbose", false, "Verbose output")
}

var cmdX = &cobra.Command{
	RunE: func(cmd *cobra.Command, args []string) error { return nil },
	Example: `+"`"+`myapp users --id usr_123 --verbose`+"`"+`,
}
`)
		}
		writeScorecardFixture(t, dir, "internal/client/client.go", `
package client
`)
		assert.Equal(t, 10, scoreAgentNative(dir))
	})
}

func TestExtractExampleField(t *testing.T) {
	t.Run("backtick delimited", func(t *testing.T) {
		content := "var cmd = &cobra.Command{\n\tExample: `mycli list --json`,\n}"
		assert.Equal(t, "mycli list --json", extractExampleField(content))
	})

	t.Run("double-quote delimited", func(t *testing.T) {
		content := `var cmd = &cobra.Command{
	Example: "mycli list --json",
}`
		assert.Equal(t, "mycli list --json", extractExampleField(content))
	})

	t.Run("empty when missing", func(t *testing.T) {
		content := `var cmd = &cobra.Command{
	Short: "List things",
}`
		assert.Equal(t, "", extractExampleField(content))
	})

	t.Run("empty string value", func(t *testing.T) {
		content := `var cmd = &cobra.Command{
	Example: "",
}`
		assert.Equal(t, "", extractExampleField(content))
	})
}

func TestAgentNativeHelpExampleValidity(t *testing.T) {
	t.Run("awards 2 when all examples valid", func(t *testing.T) {
		dir := t.TempDir()
		writeScorecardFixture(t, dir, "internal/cli/root.go", `package cli`)
		writeScorecardFixture(t, dir, "internal/cli/helpers.go", `package cli`)
		writeScorecardFixture(t, dir, "internal/client/client.go", `package client`)

		for _, name := range []string{"cmd_a.go", "cmd_b.go", "cmd_c.go", "cmd_d.go", "cmd_e.go"} {
			writeScorecardFixture(t, dir, "internal/cli/"+name, `
package cli

func init() {
	cmd.Flags().StringVar(&flagName, "name", "", "Name")
	cmd.Flags().BoolVar(&flagForce, "force", false, "Force")
}

var cmdX = &cobra.Command{
	RunE: func(cmd *cobra.Command, args []string) error { return nil },
	Example: `+"`"+`mycli do --name foo --force`+"`"+`,
}
`)
		}

		score := scoreAgentNative(dir)
		// 0 flags + 0 exit codes + 2 examples + 0 actionability + 0 bounded = 2
		assert.Equal(t, 2, score)
	})

	t.Run("awards 1 when examples present but flags invalid", func(t *testing.T) {
		dir := t.TempDir()
		writeScorecardFixture(t, dir, "internal/cli/root.go", `package cli`)
		writeScorecardFixture(t, dir, "internal/cli/helpers.go", `package cli`)
		writeScorecardFixture(t, dir, "internal/client/client.go", `package client`)

		for _, name := range []string{"cmd_a.go", "cmd_b.go", "cmd_c.go", "cmd_d.go", "cmd_e.go"} {
			writeScorecardFixture(t, dir, "internal/cli/"+name, `
package cli

func init() {
	cmd.Flags().StringVar(&flagName, "name", "", "Name")
}

var cmdX = &cobra.Command{
	RunE: func(cmd *cobra.Command, args []string) error { return nil },
	Example: `+"`"+`mycli do --nonexistent-flag`+"`"+`,
}
`)
		}

		score := scoreAgentNative(dir)
		// 0 flags + 0 exit codes + 1 (non-empty) + 0 (invalid flags) + 0 + 0 = 1
		assert.Equal(t, 1, score)
	})

	t.Run("awards 0 when examples empty", func(t *testing.T) {
		dir := t.TempDir()
		writeScorecardFixture(t, dir, "internal/cli/root.go", `package cli`)
		writeScorecardFixture(t, dir, "internal/cli/helpers.go", `package cli`)
		writeScorecardFixture(t, dir, "internal/client/client.go", `package client`)

		for _, name := range []string{"cmd_a.go", "cmd_b.go", "cmd_c.go", "cmd_d.go", "cmd_e.go"} {
			writeScorecardFixture(t, dir, "internal/cli/"+name, `
package cli

var cmdX = &cobra.Command{
	RunE: func(cmd *cobra.Command, args []string) error { return nil },
	Example: "",
}
`)
		}

		score := scoreAgentNative(dir)
		assert.Equal(t, 0, score)
	})

	t.Run("counts root.go flags as valid", func(t *testing.T) {
		dir := t.TempDir()
		writeScorecardFixture(t, dir, "internal/cli/root.go", `
package cli

func init() {
	rootCmd.PersistentFlags().StringVar(&flags.outputFormat, "output", "", "Output format")
}
`)
		writeScorecardFixture(t, dir, "internal/cli/helpers.go", `package cli`)
		writeScorecardFixture(t, dir, "internal/client/client.go", `package client`)

		for _, name := range []string{"cmd_a.go", "cmd_b.go", "cmd_c.go", "cmd_d.go", "cmd_e.go"} {
			writeScorecardFixture(t, dir, "internal/cli/"+name, `
package cli

var cmdX = &cobra.Command{
	RunE: func(cmd *cobra.Command, args []string) error { return nil },
	Example: `+"`"+`mycli do --output json`+"`"+`,
}
`)
		}

		score := scoreAgentNative(dir)
		// 0 flags + 0 exit codes + 1 (non-empty) + 1 (valid, flag from root) + 0 + 0 = 2
		assert.Equal(t, 2, score)
	})

	t.Run("skips parent commands without RunE", func(t *testing.T) {
		dir := t.TempDir()
		writeScorecardFixture(t, dir, "internal/cli/root.go", `package cli`)
		writeScorecardFixture(t, dir, "internal/cli/helpers.go", `package cli`)
		writeScorecardFixture(t, dir, "internal/client/client.go", `package client`)

		// 3 parent command files (no RunE, no Example — like command_parent.go.tmpl)
		for _, name := range []string{"users.go", "channels.go", "messages.go"} {
			writeScorecardFixture(t, dir, "internal/cli/"+name, `
package cli

func newCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "resource",
		Short: "Manage resources",
	}
	cmd.AddCommand(newListCmd())
	return cmd
}
`)
		}
		// 5 leaf command files with valid examples
		for _, name := range []string{"list_users.go", "get_user.go", "create_user.go", "delete_user.go", "update_user.go"} {
			writeScorecardFixture(t, dir, "internal/cli/"+name, `
package cli

func init() {
	cmd.Flags().StringVar(&flagID, "id", "", "Resource ID")
}

var cmdX = &cobra.Command{
	RunE: func(cmd *cobra.Command, args []string) error { return nil },
	Example: `+"`"+`mycli users --id usr_123`+"`"+`,
}
`)
		}

		score := scoreAgentNative(dir)
		// Parent files are excluded from example scoring; all 5 leaf commands have valid examples.
		// 0 flags + 0 exit codes + 2 examples + 0 actionability + 0 bounded = 2
		assert.Equal(t, 2, score)
	})
}

func TestAgentNativeErrorActionability(t *testing.T) {
	t.Run("awards 1 with 3+ distinct patterns", func(t *testing.T) {
		dir := t.TempDir()
		writeScorecardFixture(t, dir, "internal/cli/root.go", `package cli`)
		writeScorecardFixture(t, dir, "internal/client/client.go", `package client`)
		writeScorecardFixture(t, dir, "internal/cli/helpers.go", `
package cli

func handleError(err error) {
	hint1 := "try running doctor"
	hint2 := "use --json for machine output"
	hint3 := "check your API key"
	_ = hint1
	_ = hint2
	_ = hint3
}
`)

		score := scoreAgentNative(dir)
		// 0 flags + 0 exit codes + 0 examples (no cmd files) + 1 actionability + 0 bounded = 1
		assert.Equal(t, 1, score)
	})

	t.Run("awards 0 with fewer than 3 patterns", func(t *testing.T) {
		dir := t.TempDir()
		writeScorecardFixture(t, dir, "internal/cli/root.go", `package cli`)
		writeScorecardFixture(t, dir, "internal/client/client.go", `package client`)
		writeScorecardFixture(t, dir, "internal/cli/helpers.go", `
package cli

func handleError(err error) {
	hint1 := "try running doctor"
	hint2 := "run the auth command"
	_ = hint1
	_ = hint2
}
`)

		score := scoreAgentNative(dir)
		// "try " and "run " = 2 distinct, below threshold
		assert.Equal(t, 0, score)
	})

	t.Run("deduplicates same pattern", func(t *testing.T) {
		dir := t.TempDir()
		writeScorecardFixture(t, dir, "internal/cli/root.go", `package cli`)
		writeScorecardFixture(t, dir, "internal/client/client.go", `package client`)
		writeScorecardFixture(t, dir, "internal/cli/helpers.go", `
package cli

func handleError(err error) {
	hint1 := "try this"
	hint2 := "try that"
	hint3 := "try again"
	hint4 := "try once more"
	hint5 := "try everything"
	_ = hint1
	_ = hint2
	_ = hint3
	_ = hint4
	_ = hint5
}
`)

		score := scoreAgentNative(dir)
		// All "try " = 1 distinct pattern, below threshold
		assert.Equal(t, 0, score)
	})
}

func TestAgentNativeBoundedOutput(t *testing.T) {
	t.Run("awards 1 for defaultLimit in helpers", func(t *testing.T) {
		dir := t.TempDir()
		writeScorecardFixture(t, dir, "internal/cli/root.go", `package cli`)
		writeScorecardFixture(t, dir, "internal/cli/helpers.go", `
package cli

const defaultLimit = 100

func paginate() {
	limit := defaultLimit
	_ = limit
}
`)
		writeScorecardFixture(t, dir, "internal/client/client.go", `package client`)

		score := scoreAgentNative(dir)
		assert.Equal(t, 1, score)
	})

	t.Run("awards 1 for maxPages in client", func(t *testing.T) {
		dir := t.TempDir()
		writeScorecardFixture(t, dir, "internal/cli/root.go", `package cli`)
		writeScorecardFixture(t, dir, "internal/cli/helpers.go", `package cli`)
		writeScorecardFixture(t, dir, "internal/client/client.go", `
package client

const maxPages = 50
`)

		score := scoreAgentNative(dir)
		assert.Equal(t, 1, score)
	})

	t.Run("awards 1 for limit flag in root", func(t *testing.T) {
		dir := t.TempDir()
		writeScorecardFixture(t, dir, "internal/cli/root.go", `
package cli

func init() {
	rootCmd.PersistentFlags().IntVar(&flags.limit, "limit", 100, "Max results per page")
}
`)
		writeScorecardFixture(t, dir, "internal/cli/helpers.go", `package cli`)
		writeScorecardFixture(t, dir, "internal/client/client.go", `package client`)

		score := scoreAgentNative(dir)
		assert.Equal(t, 1, score)
	})

	t.Run("awards 1 for limit flag in command file", func(t *testing.T) {
		dir := t.TempDir()
		writeScorecardFixture(t, dir, "internal/cli/root.go", `package cli`)
		writeScorecardFixture(t, dir, "internal/cli/helpers.go", `package cli`)
		writeScorecardFixture(t, dir, "internal/client/client.go", `package client`)
		// Like search.go.tmpl: IntVar(&limit, "limit", 50, ...)
		writeScorecardFixture(t, dir, "internal/cli/find_things.go", `
package cli

func newSearchCmd() *cobra.Command {
	var limit int
	cmd := &cobra.Command{
		RunE: func(cmd *cobra.Command, args []string) error { return nil },
	}
	cmd.Flags().IntVar(&limit, "limit", 50, "Maximum results to return")
	return cmd
}
`)

		score := scoreAgentNative(dir)
		assert.Equal(t, 1, score)
	})

	t.Run("awards 1 for defaultLimit in infra command file", func(t *testing.T) {
		dir := t.TempDir()
		writeScorecardFixture(t, dir, "internal/cli/root.go", `package cli`)
		writeScorecardFixture(t, dir, "internal/cli/helpers.go", `package cli`)
		writeScorecardFixture(t, dir, "internal/client/client.go", `package client`)
		// analytics.go is in the infra exclusion list for sampleCommandFiles,
		// but bounded-output should still find limits declared there.
		writeScorecardFixture(t, dir, "internal/cli/analytics.go", `
package cli

func newAnalyticsCmd() *cobra.Command {
	var limit int
	cmd := &cobra.Command{
		RunE: func(cmd *cobra.Command, args []string) error { return nil },
	}
	cmd.Flags().IntVar(&limit, "limit", 25, "Max groups to show")
	return cmd
}
`)

		score := scoreAgentNative(dir)
		assert.Equal(t, 1, score)
	})

	t.Run("awards 0 for limit flag with zero default", func(t *testing.T) {
		dir := t.TempDir()
		writeScorecardFixture(t, dir, "internal/cli/root.go", `package cli`)
		writeScorecardFixture(t, dir, "internal/cli/helpers.go", `package cli`)
		writeScorecardFixture(t, dir, "internal/client/client.go", `package client`)
		// Like export.go.tmpl: IntVar(&limit, "limit", 0, "... 0 = unlimited")
		writeScorecardFixture(t, dir, "internal/cli/export_data.go", `
package cli

func newExportCmd() *cobra.Command {
	var limit int
	cmd := &cobra.Command{
		RunE: func(cmd *cobra.Command, args []string) error { return nil },
	}
	cmd.Flags().IntVar(&limit, "limit", 0, "Maximum records to export (0 = unlimited)")
	return cmd
}
`)

		score := scoreAgentNative(dir)
		assert.Equal(t, 0, score)
	})

	t.Run("awards 0 when no bounds", func(t *testing.T) {
		dir := t.TempDir()
		writeScorecardFixture(t, dir, "internal/cli/root.go", `package cli`)
		writeScorecardFixture(t, dir, "internal/cli/helpers.go", `package cli`)
		writeScorecardFixture(t, dir, "internal/client/client.go", `package client`)

		score := scoreAgentNative(dir)
		assert.Equal(t, 0, score)
	})
}

func TestAgentNativeExitCodeReduced(t *testing.T) {
	t.Run("awards 1 for 3+ exit codes", func(t *testing.T) {
		dir := t.TempDir()
		writeScorecardFixture(t, dir, "internal/cli/root.go", `package cli`)
		writeScorecardFixture(t, dir, "internal/client/client.go", `package client`)
		writeScorecardFixture(t, dir, "internal/cli/helpers.go", `
package cli

var exitCodes = map[string]int{
	"code: auth":    1,
	"code: network": 2,
	"code: input":   3,
	"code: server":  4,
	"code: timeout": 5,
}
`)

		score := scoreAgentNative(dir)
		// 0 flags + 1 exit code (reduced from 2) + 0 examples + 0 actionability + 0 bounded = 1
		assert.Equal(t, 1, score)
	})

	t.Run("awards 0 for fewer than 3 exit codes", func(t *testing.T) {
		dir := t.TempDir()
		writeScorecardFixture(t, dir, "internal/cli/root.go", `package cli`)
		writeScorecardFixture(t, dir, "internal/client/client.go", `package client`)
		writeScorecardFixture(t, dir, "internal/cli/helpers.go", `
package cli

var exitCodes = map[string]int{
	"code: auth":    1,
	"code: network": 2,
}
`)

		score := scoreAgentNative(dir)
		assert.Equal(t, 0, score)
	})
}
