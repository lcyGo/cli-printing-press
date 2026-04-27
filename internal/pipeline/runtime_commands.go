package pipeline

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

func discoverCommands(dir string, binaryPath string) []discoveredCommand {
	if binaryPath != "" {
		if cmds := discoverCommandsFromHelp(binaryPath); len(cmds) > 0 {
			return cmds
		}
	}

	return discoverCommandsFromSource(dir)
}

func discoverCommandsFromHelp(binaryPath string) []discoveredCommand {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	helpCmd := exec.CommandContext(ctx, binaryPath, "--help")
	out, err := helpCmd.CombinedOutput()
	if err != nil {
		return nil
	}

	return parseHelpCommands(string(out))
}

func parseHelpCommands(helpOutput string) []discoveredCommand {
	lines := strings.Split(helpOutput, "\n")
	inAvailable := false
	var commands []discoveredCommand
	seen := map[string]bool{}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "Available Commands:") {
			inAvailable = true
			continue
		}

		// An empty line or a new section header ends the Available Commands block.
		if inAvailable && (trimmed == "" || (len(trimmed) > 0 && trimmed[len(trimmed)-1] == ':' && !strings.Contains(trimmed, " "))) {
			break
		}

		if !inAvailable {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		name := fields[0]
		if seen[name] {
			continue
		}
		seen[name] = true

		switch name {
		case "version", "completion", "help":
			continue
		}
		commands = append(commands, discoveredCommand{Name: name})
	}
	return commands
}

func discoverCommandsFromSource(dir string) []discoveredCommand {
	rootPath := filepath.Join(dir, "internal", "cli", "root.go")
	data, err := os.ReadFile(rootPath)
	if err != nil {
		return nil
	}

	// Match: rootCmd.AddCommand(newXxxCmd(...))
	re := regexp.MustCompile(`rootCmd\.AddCommand\(new(\w+)Cmd\(`)
	matches := re.FindAllStringSubmatch(string(data), -1)

	var commands []discoveredCommand
	seen := map[string]bool{}
	for _, m := range matches {
		name := camelToKebab(m[1])
		if seen[name] {
			continue
		}
		seen[name] = true
		switch name {
		case "version-pp-cli", "version-cli", "version", "completion", "help":
			continue
		}
		commands = append(commands, discoveredCommand{Name: name})
	}
	return commands
}

type discoveredCommand struct {
	Name string
	Kind string // read, write, local, data-layer
	Args []string
}

// inferPositionalArgs runs `<binary> <cmd> --help`, parses the Usage line for
// positional arg placeholders like <region> or [price], and maps them to
// synthetic values. On any failure, it falls back to no extra args.
func inferPositionalArgs(binary string, cmd *discoveredCommand) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	helpCmd := exec.CommandContext(ctx, binary, cmd.Name, "--help")
	out, err := helpCmd.CombinedOutput()
	if err != nil {
		return // fall back to no extra args
	}

	// Find the Usage line, e.g. "Usage:\n  cli-name pulse <region> [flags]"
	usageRe := regexp.MustCompile(`(?m)^Usage:\s*\n\s+\S+\s+\S+(.*)$`)
	m := usageRe.FindSubmatch(out)
	if m == nil {
		return
	}

	for _, name := range extractPositionalPlaceholders(string(m[1])) {
		cmd.Args = append(cmd.Args, syntheticArgValue(name))
	}
}

// flagDescriptorRe matches a bracketed token whose body looks like a flag
// descriptor rather than an optional positional. The body starts with one
// or more leading dashes, or contains an `=` sign (e.g., `[--tags=<csv>]`,
// `[--stdin]`, `[-v]`). Without scrubbing these first, the placeholder
// regex picks up `<csv>` from such tokens as if it were a separate
// positional, which then gets passed to the binary and breaks
// `cobra.MaximumNArgs(1)` validators on commands that accept exactly one
// real positional. See retro #301 finding F2.
var flagDescriptorRe = regexp.MustCompile(`\[\s*-+[^\]]*\]|\[[^\]]*=[^\]]*\]`)

// positionalPlaceholderRe extracts <name> and [name] placeholders from the
// scrubbed Usage suffix. Runs after flagDescriptorRe.
var positionalPlaceholderRe = regexp.MustCompile(`[<\[]([a-zA-Z][\w-]*)[>\]]`)

// extractPositionalPlaceholders returns the placeholder names found in a
// cobra Usage suffix (the part after `Usage:\n  cli-name cmd-name`).
// It strips bracketed flag descriptors first so tokens like `[--tags=<csv>]`
// don't contribute `<csv>` as a phantom positional, then drops cobra's
// built-in `[flags]` / `[command]` placeholders.
//
// Returns lowercase placeholder names in source order.
func extractPositionalPlaceholders(usageSuffix string) []string {
	scrubbed := flagDescriptorRe.ReplaceAllString(usageSuffix, "")
	matches := positionalPlaceholderRe.FindAllStringSubmatch(scrubbed, -1)
	if len(matches) == 0 {
		return nil
	}
	var names []string
	for _, match := range matches {
		name := strings.ToLower(match[1])
		if name == "flags" || name == "command" {
			continue
		}
		names = append(names, name)
	}
	return names
}

func syntheticArgValue(name string) string {
	switch name {
	case "region", "location", "city":
		return "mock-city"
	case "id", "property-id", "listing-id":
		return "12345"
	case "price", "amount":
		return "500000"
	case "zip", "zipcode":
		return "94102"
	case "url", "path":
		return "/mock/path"
	case "query", "search", "name":
		return "mock-query"
	case "type", "entity-type", "entity", "kind":
		return "collection"
	case "resource", "resource-type":
		return "items"
	case "format", "output-format":
		return "json"
	case "category", "slug":
		return "general"
	case "action", "command", "operation":
		return "list"
	case "status", "state":
		return "active"
	default:
		return "mock-value"
	}
}

func classifyCommandKind(cmd *discoveredCommand, spec *openAPISpec) {
	name := cmd.Name
	switch name {
	case "sync", "search", "sql", "health", "trends", "patterns", "analytics",
		"export", "import", "stale", "no-show", "today", "busy", "diff",
		"noshow", "velocity", "popular":
		cmd.Kind = "data-layer"
		return
	case "doctor", "auth", "api", "completion":
		cmd.Kind = "local"
		return
	case "tail":
		cmd.Kind = "data-layer"
		return
	}

	if spec != nil && len(spec.Paths) > 0 {
		cmd.Kind = "read"
		return
	}

	cmd.Kind = "read"
}

// workflowTestFlags returns flags needed for workflow commands that require --org or --repo.
// Retained for explicit positional-arg patterns (e.g., changelog takes two positional
// args, not flags — cobra won't surface them through the "required flag(s) not set"
// error). Flag-shaped requirements are now discovered dynamically via inferRequiredFlags.
func workflowTestFlags(cmdName string) []string {
	switch cmdName {
	case "changelog":
		return []string{"mock-owner", "mock-repo", "--since", "v0.0.1"}
	default:
		return nil
	}
}

// requiredFlagsRe matches cobra's standard "required flag(s) ... not set" error.
// Cobra emits the flag names quoted, comma-separated: required flag(s) "event", "year" not set
var requiredFlagsRe = regexp.MustCompile(`required flag\(s\) ((?:"[^"]+"(?:, )?)+) not set`)

var flagNameRe = regexp.MustCompile(`"([^"]+)"`)

// inferRequiredFlags probes a command by running it with no args, parses cobra's
// "required flag(s) ... not set" error if present, and returns synthetic --flag value
// pairs the verifier can use to exercise the command. Returns nil when the command
// has no required flags (or when probing fails — the caller falls back gracefully).
func inferRequiredFlags(binary, cmdName string) []string {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	probe := exec.CommandContext(ctx, binary, cmdName)
	out, _ := probe.CombinedOutput() // error expected when flags are missing

	m := requiredFlagsRe.FindSubmatch(out)
	if m == nil {
		return nil
	}

	nameMatches := flagNameRe.FindAllSubmatch(m[1], -1)
	if len(nameMatches) == 0 {
		return nil
	}

	var args []string
	for _, nm := range nameMatches {
		flag := string(nm[1])
		args = append(args, "--"+flag, syntheticFlagValue(flag))
	}
	return args
}

// syntheticFlagValue maps a required flag name to a synthetic test value. Shares
// its philosophy with syntheticArgValue but keyed on flag names that appear in
// "required flag(s)" errors. The mock server doesn't validate values, so any
// non-empty string of the right shape works.
func syntheticFlagValue(name string) string {
	n := strings.ToLower(name)
	switch n {
	case "org", "organization", "owner":
		return "mock-owner"
	case "repo", "repository":
		return "mock-owner/mock-repo"
	case "team", "workspace", "project", "workspace-id", "project-id":
		return "mock-project"
	case "user", "username", "user-id", "account", "account-id":
		return "mock-user"
	case "event", "event-id", "game", "game-id", "match", "match-id":
		return "mock-event-123"
	case "season", "year":
		return "2026"
	case "sport", "league", "competition":
		return "mock-league"
	case "id", "uid", "uuid":
		return "mock-id-123"
	case "ticker", "symbol":
		return "MOCK"
	case "region", "location", "city":
		return "mock-city"
	case "date", "day":
		return "2026-04-11"
	case "since", "from", "start", "start-date":
		return "2026-01-01"
	case "until", "to", "end", "end-date":
		return "2026-12-31"
	case "query", "q", "search", "term":
		return "mock-query"
	case "name", "slug", "key":
		return "mock-name"
	case "type", "kind", "category":
		return "mock-type"
	case "status", "state":
		return "active"
	case "limit", "count", "size":
		return "10"
	case "format", "output":
		return "json"
	case "url", "endpoint", "base-url":
		return "https://mock.example.com"
	case "path", "file", "output-file":
		return "/tmp/mock-file"
	case "token", "api-key", "key-id", "secret":
		return "mock-secret"
	default:
		return "mock-value"
	}
}
