package cli

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBrowserSniffCmdRejectsDomainMismatchOnAuthFrom(t *testing.T) {
	t.Parallel()

	cmd := newBrowserSniffCmd()
	outputPath := filepath.Join(t.TempDir(), "spec.yaml")
	cmd.SetArgs([]string{
		"--har", filepath.Join("..", "..", "testdata", "sniff", "sample-enriched.json"),
		"--auth-from", filepath.Join("..", "..", "testdata", "sniff", "sample-auth-capture-mismatch.json"),
		"--output", outputPath,
	})

	err := cmd.Execute()
	require.Error(t, err)
	assert.EqualError(t, err, "auth captured for other.example.com cannot be used with hn.algolia.com (domain mismatch)")
}

// newRootCmdForTest mirrors Execute()'s command tree construction for test-level
// command dispatch assertions.
func newRootCmdForTest() *cobra.Command {
	root := &cobra.Command{Use: "printing-press", SilenceUsage: true, SilenceErrors: true}
	root.AddCommand(newBrowserSniffCmd())
	root.AddCommand(newCrowdSniffCmd())
	return root
}

func TestLegacySniffCommandReturnsUnknownCommand(t *testing.T) {
	t.Parallel()

	root := newRootCmdForTest()
	root.SetArgs([]string{"sniff", "--har", "/tmp/whatever.har"})
	root.SetOut(new(bytes.Buffer))
	root.SetErr(new(bytes.Buffer))

	err := root.Execute()
	require.Error(t, err, "invoking legacy 'sniff' must fail after the rename")
	assert.Contains(t, err.Error(), "unknown command", "cobra should surface an unknown-command error")
}

func TestBrowserSniffAppearsInHelp(t *testing.T) {
	t.Parallel()

	root := newRootCmdForTest()
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetArgs([]string{"--help"})

	require.NoError(t, root.Execute())
	out := buf.String()
	assert.Contains(t, out, "browser-sniff", "browser-sniff should be listed in help")
	assert.NotContains(t, lineWithToken(out, "sniff"), "\n  sniff ", "bare 'sniff' should not appear as a top-level command in help")
}

// lineWithToken is a trivial helper — the NotContains check above looks for the
// subcommand indent pattern cobra uses when listing commands.
func lineWithToken(s, _ string) string {
	// Normalize to make the NotContains assertion robust across cobra versions.
	return "\n" + strings.ReplaceAll(s, "\r\n", "\n")
}

func TestCrowdSniffStillWorksAfterBrowserSniffRename(t *testing.T) {
	t.Parallel()

	root := newRootCmdForTest()
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetArgs([]string{"crowd-sniff", "--help"})

	require.NoError(t, root.Execute(), "crowd-sniff --help must still succeed after browser-sniff rename")
	out := buf.String()
	assert.Contains(t, out, "crowd-sniff", "crowd-sniff help output should reference the command name")
}
