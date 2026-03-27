package pipeline

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

// Scorecard holds the auto-scored evaluation of a generated CLI against the Steinberger bar.
type Scorecard struct {
	APIName          string       `json:"api_name"`
	Steinberger      SteinerScore `json:"steinberger"`
	CompetitorScores []CompScore  `json:"competitor_scores"`
	OverallGrade     string       `json:"overall_grade"`
	GapReport        []string     `json:"gap_report"`
}

// SteinerScore breaks down the Steinberger bar into 11 dimensions, each 0-10.
type SteinerScore struct {
	OutputModes   int `json:"output_modes"`   // 0-10
	Auth          int `json:"auth"`           // 0-10
	ErrorHandling int `json:"error_handling"` // 0-10
	TerminalUX    int `json:"terminal_ux"`    // 0-10
	README        int `json:"readme"`         // 0-10
	Doctor        int `json:"doctor"`         // 0-10
	AgentNative   int `json:"agent_native"`   // 0-10
	LocalCache    int `json:"local_cache"`    // 0-10
	Breadth       int `json:"breadth"`        // 0-10: how many commands (penalizes empty CLIs)
	Vision        int `json:"vision"`         // 0-10
	Workflows     int `json:"workflows"`      // 0-10
	Insight       int `json:"insight"`        // 0-10
	// Tier 2: Domain Correctness (semantic checks)
	PathValidity          int `json:"path_validity"`           // 0-10
	AuthProtocol          int `json:"auth_protocol"`           // 0-10
	DataPipelineIntegrity int `json:"data_pipeline_integrity"` // 0-10
	SyncCorrectness       int `json:"sync_correctness"`        // 0-10
	TypeFidelity          int `json:"type_fidelity"`           // 0-5
	DeadCode              int `json:"dead_code"`               // 0-5
	Total                 int `json:"total"`                   // 0-100 (weighted: 50% infrastructure + 50% domain)
	Percentage            int `json:"percentage"`              // 0-100
}

// CompScore compares our score against a competitor on a single dimension.
type CompScore struct {
	Name       string `json:"name"`
	OurScore   int    `json:"our_score"`
	TheirScore int    `json:"their_score"`
	WeWin      bool   `json:"we_win"`
}

// RunScorecard evaluates generated CLI files and produces a scorecard.
func RunScorecard(outputDir, pipelineDir, specPath string) (*Scorecard, error) {
	sc := &Scorecard{}

	// Infer API name from outputDir basename
	sc.APIName = filepath.Base(outputDir)

	// Score each Steinberger dimension by inspecting generated files
	sc.Steinberger.OutputModes = scoreOutputModes(outputDir)
	sc.Steinberger.Auth = scoreAuth(outputDir)
	sc.Steinberger.ErrorHandling = scoreErrorHandling(outputDir)
	sc.Steinberger.TerminalUX = scoreTerminalUX(outputDir)
	sc.Steinberger.README = scoreREADME(outputDir)
	sc.Steinberger.Doctor = scoreDoctor(outputDir)
	sc.Steinberger.AgentNative = scoreAgentNative(outputDir)
	sc.Steinberger.LocalCache = scoreLocalCache(outputDir)
	sc.Steinberger.Breadth = scoreBreadth(outputDir)
	sc.Steinberger.Vision = scoreVision(outputDir)
	sc.Steinberger.Workflows = scoreWorkflows(outputDir)
	sc.Steinberger.Insight = scoreInsight(outputDir)
	sc.Steinberger.PathValidity = scorePathValidity(outputDir, specPath)
	sc.Steinberger.AuthProtocol = scoreAuthProtocol(outputDir, specPath)
	sc.Steinberger.DataPipelineIntegrity = scoreDataPipelineIntegrity(outputDir)
	sc.Steinberger.SyncCorrectness = scoreSyncCorrectness(outputDir)
	sc.Steinberger.TypeFidelity = scoreTypeFidelity(outputDir)
	sc.Steinberger.DeadCode = scoreDeadCode(outputDir)

	// Tier 1: Infrastructure (string-matching, 120 max)
	tier1Raw := sc.Steinberger.OutputModes +
		sc.Steinberger.Auth +
		sc.Steinberger.ErrorHandling +
		sc.Steinberger.TerminalUX +
		sc.Steinberger.README +
		sc.Steinberger.Doctor +
		sc.Steinberger.AgentNative +
		sc.Steinberger.LocalCache +
		sc.Steinberger.Breadth +
		sc.Steinberger.Vision +
		sc.Steinberger.Workflows +
		sc.Steinberger.Insight

	// Tier 2: Domain Correctness (semantic, 50 max)
	tier2Raw := sc.Steinberger.PathValidity +
		sc.Steinberger.AuthProtocol +
		sc.Steinberger.DataPipelineIntegrity +
		sc.Steinberger.SyncCorrectness +
		sc.Steinberger.TypeFidelity +
		sc.Steinberger.DeadCode

	// Weighted composite: Tier 1 = 50%, Tier 2 = 50% of final 100-point scale
	tier1Normalized := (tier1Raw * 50) / 120 // scale 0-120 to 0-50
	tier2Normalized := (tier2Raw * 50) / 50  // scale 0-50 to 0-50
	sc.Steinberger.Total = tier1Normalized + tier2Normalized

	if sc.Steinberger.Total > 0 {
		sc.Steinberger.Percentage = sc.Steinberger.Total // Total IS the percentage (0-100)
	}

	// Grade
	sc.OverallGrade = computeGrade(sc.Steinberger.Percentage)

	// Gap report for dimensions below 5
	sc.GapReport = buildGapReport(sc.Steinberger)

	// Competitor comparison from research.json
	sc.CompetitorScores = buildCompetitorScores(sc.Steinberger.Total, pipelineDir)

	// Write scorecard.md
	if err := writeScorecardMD(sc, pipelineDir); err != nil {
		return sc, fmt.Errorf("writing scorecard.md: %w", err)
	}

	return sc, nil
}

func scoreOutputModes(dir string) int {
	rootContent := readFileContent(filepath.Join(dir, "internal", "cli", "root.go"))
	helpersContent := readFileContent(filepath.Join(dir, "internal", "cli", "helpers.go"))
	score := 0
	// Presence tier (max 5)
	if strings.Contains(rootContent, `"json"`) {
		score += 1
	}
	if strings.Contains(rootContent, `"plain"`) {
		score += 1
	}
	if strings.Contains(rootContent, `"select"`) {
		score += 1
	}
	if strings.Contains(rootContent, `"csv"`) {
		score += 1
	}
	if strings.Contains(rootContent, `"quiet"`) {
		score += 1
	}
	// Quality tier: field-aware select (real JSON parsing, not string ops)
	if strings.Contains(helpersContent, "filterFields") && strings.Contains(helpersContent, "json.Unmarshal") {
		score += 2
	}
	// Quality tier: pagination progress events
	if strings.Contains(helpersContent, "page_fetch") || strings.Contains(helpersContent, "ndjson") {
		score += 1
	}
	// Quality tier: tabwriter for aligned output
	if strings.Contains(helpersContent, "tabwriter") {
		score += 2
	}
	if score > 10 {
		score = 10
	}
	return score
}

func scoreAuth(dir string) int {
	configContent := readFileContent(filepath.Join(dir, "internal", "config", "config.go"))
	authContent := readFileContent(filepath.Join(dir, "internal", "cli", "auth.go"))
	clientContent := readFileContent(filepath.Join(dir, "internal", "client", "client.go"))
	score := 0
	// Presence: at least one env var
	if strings.Count(configContent, "os.Getenv") >= 1 {
		score += 2
	}
	// Presence: auth file exists
	if authContent != "" {
		score += 1
	}
	// Quality: secure config file permissions (0o600 or 0600)
	if strings.Contains(configContent, "0o600") || strings.Contains(configContent, "0600") || strings.Contains(configContent, "0o700") || strings.Contains(configContent, "0700") {
		score += 2
	}
	// Quality: token masking in output (showing partial token)
	if strings.Contains(clientContent, "mask") || strings.Contains(clientContent, "***") || strings.Contains(clientContent, "last 4") || (strings.Contains(clientContent, "Authorization") && strings.Contains(clientContent, "[:")) {
		score += 2
	}
	// Quality: multiple auth methods (env var + config + flag)
	authSources := 0
	if strings.Contains(configContent, "os.Getenv") {
		authSources++
	}
	if strings.Contains(configContent, "ReadFile") || strings.Contains(configContent, "Load") {
		authSources++
	}
	if authSources >= 2 {
		score += 1
	}
	// Excellence: OAuth2 browser flow with refresh
	if strings.Contains(authContent, "oauth2") || strings.Contains(authContent, "OAuth2") {
		if strings.Contains(authContent, "refresh") || strings.Contains(authContent, "Refresh") {
			score += 2
		}
	}
	if score > 10 {
		score = 10
	}
	return score
}

func scoreErrorHandling(dir string) int {
	helpersContent := readFileContent(filepath.Join(dir, "internal", "cli", "helpers.go"))
	clientContent := readFileContent(filepath.Join(dir, "internal", "client", "client.go"))
	score := 0
	// Presence: error hints
	if strings.Contains(helpersContent, "hint:") || strings.Contains(helpersContent, "Hint:") {
		score += 1
	}
	// Presence: at least 3 distinct exit codes
	exitCount := strings.Count(helpersContent, "code:")
	if exitCount >= 3 {
		score += 2
	} else if exitCount >= 1 {
		score += 1
	}
	// Quality: rate limit handling (429 + retry)
	if strings.Contains(clientContent, "429") && (strings.Contains(clientContent, "Retry-After") || strings.Contains(clientContent, "backoff") || strings.Contains(clientContent, "retry")) {
		score += 2
	}
	// Quality: idempotency (409 = already exists = success)
	if strings.Contains(helpersContent, "409") && strings.Contains(helpersContent, "already exists") {
		score += 2
	}
	// Quality: 404 with specific exit code
	if strings.Contains(helpersContent, "404") {
		score += 1
	}
	// Excellence: actionable suggestions in errors (not just codes)
	if (strings.Contains(helpersContent, "Run") || strings.Contains(helpersContent, "try")) && strings.Contains(helpersContent, "doctor") {
		score += 2
	}
	if score > 10 {
		score = 10
	}
	return score
}

func scoreTerminalUX(dir string) int {
	helpersContent := readFileContent(filepath.Join(dir, "internal", "cli", "helpers.go"))
	rootContent := readFileContent(filepath.Join(dir, "internal", "cli", "root.go"))
	score := 0
	// Presence: NO_COLOR support
	if strings.Contains(helpersContent, "NO_COLOR") {
		score += 1
	}
	// Presence: TTY detection
	if strings.Contains(helpersContent, "isatty") {
		score += 1
	}
	// Presence: no-color flag
	if strings.Contains(rootContent, "no-color") {
		score += 1
	}
	// Quality: tabwriter for aligned columns
	if strings.Contains(helpersContent, "tabwriter") {
		score += 2
	}
	// Quality: help text descriptions are meaningful (not just verb names)
	cmdFiles := sampleCommandFiles(dir, 5)
	goodDescs := 0
	for _, content := range cmdFiles {
		if hasQualityDescription(content) {
			goodDescs++
		}
	}
	if goodDescs >= 4 {
		score += 2
	} else if goodDescs >= 2 {
		score += 1
	}
	// Quality: example values are realistic (not abc123 or bare "value")
	goodExamples := 0
	for _, content := range cmdFiles {
		if !hasPlaceholderValues(content) {
			goodExamples++
		}
	}
	if goodExamples >= 4 {
		score += 3
	} else if goodExamples >= 2 {
		score += 1
	}
	if score > 10 {
		score = 10
	}
	return score
}

func scoreREADME(dir string) int {
	content := readFileContent(filepath.Join(dir, "README.md"))
	score := 0
	// Presence: key sections exist (1pt each, max 4)
	for _, section := range []string{"Quick Start", "Agent Usage", "Doctor", "Troubleshooting"} {
		if strings.Contains(content, section) {
			score++
		}
	}
	// Quality: Quick Start has no placeholder values
	qsIdx := strings.Index(content, "Quick Start")
	if qsIdx >= 0 {
		qsSection := content[qsIdx:min(qsIdx+500, len(content))]
		if !strings.Contains(qsSection, "your-key-here") && !strings.Contains(qsSection, "USER/tap") && !strings.Contains(qsSection, "abc123") {
			score += 2
		}
	}
	// Quality: has Cookbook or Recipes with 3+ code blocks
	if strings.Contains(content, "Cookbook") || strings.Contains(content, "Recipes") {
		codeBlocks := strings.Count(content, "```")
		if codeBlocks >= 6 { // 3+ examples = 6+ backtick pairs
			score += 2
		} else {
			score += 1
		}
	}
	// Quality: README describes the API in human terms (not raw spec text)
	lines := strings.SplitN(content, "\n", 5)
	if len(lines) >= 3 {
		header := strings.Join(lines[:3], " ")
		if !strings.Contains(header, "Preview of") && !strings.Contains(header, "specification") && len(header) > 20 {
			score += 2
		}
	}
	if score > 10 {
		score = 10
	}
	return score
}

func scoreDoctor(dir string) int {
	content := readFileContent(filepath.Join(dir, "internal", "cli", "doctor.go"))
	if content == "" {
		return 0
	}
	score := 0
	// Presence: doctor command exists
	score += 2
	// Quality: checks auth/token validity
	if strings.Contains(content, "auth") || strings.Contains(content, "token") || strings.Contains(content, "Token") {
		score += 2
	}
	// Quality: checks API connectivity (makes an HTTP request)
	hasHTTP := strings.Contains(content, "http.Get") || strings.Contains(content, "http.Head") ||
		strings.Contains(content, "http.NewRequest") || strings.Contains(content, "httpClient")
	if hasHTTP {
		score += 2
	}
	// Quality: checks config file
	if strings.Contains(content, "config") || strings.Contains(content, "Config") {
		score += 2
	}
	// Excellence: checks version or API compatibility
	if strings.Contains(content, "version") || strings.Contains(content, "Version") {
		score += 2
	}
	if score > 10 {
		score = 10
	}
	return score
}

func scoreAgentNative(dir string) int {
	rootContent := readFileContent(filepath.Join(dir, "internal", "cli", "root.go"))
	helpersContent := readFileContent(filepath.Join(dir, "internal", "cli", "helpers.go"))
	clientContent := readFileContent(filepath.Join(dir, "internal", "client", "client.go"))
	score := 0
	// Presence: core agent flags (1pt each, max 5)
	if strings.Contains(rootContent, `"json"`) {
		score++
	}
	if strings.Contains(rootContent, `"select"`) {
		score++
	}
	if strings.Contains(rootContent, "dry-run") {
		score++
	}
	if strings.Contains(rootContent, "stdin") {
		score++
	}
	if strings.Contains(rootContent, `"yes"`) {
		score++
	}
	// Quality: typed exit codes (3+ distinct) — +1
	exitCount := strings.Count(helpersContent, "code:")
	if exitCount >= 3 {
		score++
	}
	// Quality: help example validity (+2)
	// Sample leaf command files (with RunE/Run:) and verify Example: fields are non-empty with valid flags
	allSampled := sampleCommandFiles(dir, 10)
	var cmdFiles []string
	for _, content := range allSampled {
		if strings.Contains(content, "RunE") || strings.Contains(content, "Run:") {
			cmdFiles = append(cmdFiles, content)
		}
	}
	nonEmptyExamples := 0
	validFlagExamples := 0
	flagDeclRe := regexp.MustCompile(`(?:StringVar|BoolVar|IntVar|Float64Var|StringSliceVar|Int64Var|UintVar)\b[^"]*"([a-z][-a-z0-9]*)"`)
	exampleFlagRe := regexp.MustCompile(`--([a-z][-a-z0-9]*)`)
	rootFlags := map[string]bool{}
	for _, m := range flagDeclRe.FindAllStringSubmatch(rootContent, -1) {
		rootFlags[m[1]] = true
	}
	for _, content := range cmdFiles {
		exampleText := extractExampleField(content)
		if strings.TrimSpace(exampleText) != "" {
			nonEmptyExamples++
		}
		// Collect flag declarations from this file
		localFlags := map[string]bool{}
		for k, v := range rootFlags {
			localFlags[k] = v
		}
		for _, m := range flagDeclRe.FindAllStringSubmatch(content, -1) {
			localFlags[m[1]] = true
		}
		// Check flags used in examples exist as declarations
		usedFlags := exampleFlagRe.FindAllStringSubmatch(exampleText, -1)
		if len(usedFlags) > 0 {
			allValid := true
			for _, m := range usedFlags {
				if !localFlags[m[1]] {
					allValid = false
					break
				}
			}
			if allValid {
				validFlagExamples++
			}
		} else if strings.TrimSpace(exampleText) != "" {
			// No flags in example is still valid (e.g. positional args only)
			validFlagExamples++
		}
	}
	if len(cmdFiles) > 0 {
		if nonEmptyExamples*100/len(cmdFiles) >= 80 {
			score++
		}
		if validFlagExamples*100/len(cmdFiles) >= 80 {
			score++
		}
	}
	// Quality: error message actionability (+1)
	// Check that helpers.go contains actionable suggestion patterns
	suggestionRe := regexp.MustCompile(`(?i)(try |use |run |check |see |ensure |verify )`)
	suggestions := suggestionRe.FindAllString(helpersContent, -1)
	distinctSuggestions := map[string]bool{}
	for _, s := range suggestions {
		distinctSuggestions[strings.ToLower(strings.TrimSpace(s))] = true
	}
	if len(distinctSuggestions) >= 3 {
		score++
	}
	// Quality: bounded output defaults (+1)
	// Check for default page/result limits that prevent unbounded pagination.
	// Scan root.go, helpers.go, client.go, and all CLI command files (including
	// infra files like search.go and analytics.go that define their own limits).
	hasBound := false
	boundPatterns := []string{"defaultLimit", "maxPages", "MaxResults", "DefaultPageSize"}
	boundSources := []string{rootContent, helpersContent, clientContent}
	// Also read all .go files in internal/cli/ for command-level limits
	cliDir := filepath.Join(dir, "internal", "cli")
	if entries, err := os.ReadDir(cliDir); err == nil {
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".go") &&
				e.Name() != "root.go" && e.Name() != "helpers.go" {
				boundSources = append(boundSources, readFileContent(filepath.Join(cliDir, e.Name())))
			}
		}
	}
	for _, src := range boundSources {
		for _, p := range boundPatterns {
			if strings.Contains(src, p) {
				hasBound = true
				break
			}
		}
		if hasBound {
			break
		}
	}
	if !hasBound {
		// Also check for --limit flag with a non-zero default (e.g. IntVar(&x, "limit", 100, ...))
		// The default value follows "limit", so match: "limit", <non-zero int>,
		// This excludes "limit", 0, ... which means unlimited.
		limitDefaultRe := regexp.MustCompile(`(?:Int|Uint|Int64)(?:Var|VarP)?\([^)]*"limit"\s*,\s*[1-9][0-9]*\s*,`)
		for _, src := range boundSources {
			if limitDefaultRe.MatchString(src) {
				hasBound = true
				break
			}
		}
	}
	if hasBound {
		score++
	}
	if score > 10 {
		score = 10
	}
	return score
}

// extractExampleField extracts the Example: field value from a cobra command file.
func extractExampleField(content string) string {
	idx := strings.Index(content, "Example:")
	if idx < 0 {
		return ""
	}
	rest := content[idx+len("Example:"):]
	rest = strings.TrimSpace(rest)
	if len(rest) == 0 {
		return ""
	}
	// Handle backtick-delimited strings
	if rest[0] == '`' {
		end := strings.Index(rest[1:], "`")
		if end < 0 {
			return ""
		}
		return rest[1 : 1+end]
	}
	// Handle double-quoted strings
	if rest[0] == '"' {
		end := strings.Index(rest[1:], `"`)
		if end < 0 {
			return ""
		}
		return rest[1 : 1+end]
	}
	return ""
}

func scoreLocalCache(dir string) int {
	clientContent := readFileContent(filepath.Join(dir, "internal", "client", "client.go"))
	score := 0
	// Presence: GET response caching
	if strings.Contains(clientContent, "readCache") || strings.Contains(clientContent, "writeCache") || strings.Contains(clientContent, "cacheDir") {
		score += 2
	}
	// Presence: --no-cache bypass
	if strings.Contains(clientContent, "no-cache") || strings.Contains(clientContent, "NoCache") {
		score += 1
	}
	// Quality: cache has TTL (time-based expiry)
	if strings.Contains(clientContent, "time.Duration") || strings.Contains(clientContent, "ModTime") || strings.Contains(clientContent, "TTL") || strings.Contains(clientContent, "ttl") {
		score += 2
	}
	// Quality: XDG or standard cache directory
	if strings.Contains(clientContent, ".cache") || strings.Contains(clientContent, "XDG_CACHE_HOME") || strings.Contains(clientContent, "UserCacheDir") {
		score += 2
	}
	// Excellence: SQLite or embedded DB
	for _, name := range []string{"internal/cache/cache.go", "internal/store/store.go"} {
		content := readFileContent(filepath.Join(dir, name))
		if strings.Contains(content, "sqlite") || strings.Contains(content, "bolt") || strings.Contains(content, "badger") {
			score += 3
			break
		}
	}
	if score > 10 {
		score = 10
	}
	return score
}

func scoreBreadth(dir string) int {
	cliDir := filepath.Join(dir, "internal", "cli")
	entries, err := os.ReadDir(cliDir)
	if err != nil {
		return 0
	}
	infra := map[string]bool{
		"helpers.go": true, "root.go": true, "doctor.go": true, "auth.go": true,
		"export.go": true, "import.go": true, "search.go": true, "sync.go": true,
		"tail.go": true, "analytics.go": true,
	}
	commandFiles := 0
	lazyDescs := 0
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") {
			continue
		}
		if infra[e.Name()] {
			continue
		}
		commandFiles++
		// Check for lazy 1-word descriptions
		content := readFileContent(filepath.Join(cliDir, e.Name()))
		if hasLazyDescription(content) {
			lazyDescs++
		}
	}

	var score int
	switch {
	case commandFiles >= 60:
		score = 8
	case commandFiles >= 41:
		score = 7
	case commandFiles >= 21:
		score = 5
	case commandFiles >= 11:
		score = 4
	case commandFiles >= 5:
		score = 2
	default:
		return 0
	}
	// Penalty: if more than 50% of commands have lazy 1-word descriptions
	if commandFiles > 0 && lazyDescs*2 > commandFiles {
		score -= 2
	}
	// Bonus: if descriptions are mostly quality (< 20% lazy)
	if commandFiles > 0 && lazyDescs*5 < commandFiles {
		score += 2
	}
	if score > 10 {
		score = 10
	}
	if score < 0 {
		score = 0
	}
	return score
}

func scoreVision(dir string) int {
	cliDir := filepath.Join(dir, "internal", "cli")

	// Tier 1: Feature Presence (0-5 points)
	tier1 := 0.0
	if fileExists(filepath.Join(cliDir, "export.go")) {
		tier1 += 1.0
	}
	if fileExists(filepath.Join(dir, "internal", "store", "store.go")) {
		tier1 += 1.0
	}
	if fileExists(filepath.Join(cliDir, "search.go")) {
		tier1 += 1.0
	}
	if fileExists(filepath.Join(cliDir, "sync.go")) {
		tier1 += 0.5
	}
	if fileExists(filepath.Join(cliDir, "tail.go")) {
		tier1 += 0.5
	}
	if fileExists(filepath.Join(cliDir, "import.go")) {
		tier1 += 0.5
	}
	// Workflow or compound command files
	entries, err := os.ReadDir(cliDir)
	if err == nil {
		for _, e := range entries {
			name := e.Name()
			if strings.Contains(name, "_workflow") || strings.Contains(name, "_compound") {
				if strings.HasSuffix(name, ".go") {
					tier1 += 0.5
					break
				}
			}
		}
	}
	if tier1 > 5 {
		tier1 = 5
	}

	// Tier 2: Feature Intelligence (0-5 points)
	tier2 := 0.0

	// Schema depth (0-1.5): check if store.go has domain-specific tables
	storePath := filepath.Join(dir, "internal", "store", "store.go")
	if fileExists(storePath) {
		storeContent := readFileContent(storePath)
		tableCount := strings.Count(storeContent, "CREATE TABLE")
		syncStateCount := strings.Count(storeContent, "sync_state")
		domainTables := tableCount
		if syncStateCount > 0 {
			domainTables-- // Don't count sync_state as a domain table
		}
		if domainTables >= 3 {
			tier2 += 1.5
		} else if domainTables >= 2 {
			tier2 += 1.0
		} else if domainTables >= 1 {
			tier2 += 0.5
		}
	}

	// Wiring check (0-1.5): are vision commands registered in root.go?
	rootPath := filepath.Join(cliDir, "root.go")
	if fileExists(rootPath) {
		rootContent := readFileContent(rootPath)
		visionFuncs := []string{"newSyncCmd", "newSearchCmd", "newExportCmd", "newTailCmd", "newImportCmd", "newAnalyticsCmd"}
		wired := 0
		for _, fn := range visionFuncs {
			if strings.Contains(rootContent, fn) {
				wired++
			}
		}
		tier2 += float64(wired) * 0.25
		if tier2 > 3.0 { // cap wiring contribution
			tier2 = 3.0
		}
	}

	// FTS5 check (0-1.0): does the store have full-text search?
	if fileExists(storePath) {
		storeContent := readFileContent(storePath)
		if strings.Contains(storeContent, "fts5") || strings.Contains(storeContent, "FTS5") {
			tier2 += 1.0
		}
	}

	// Search uses store (0-0.5): does search.go reference the store package?
	searchPath := filepath.Join(cliDir, "search.go")
	if fileExists(searchPath) {
		searchContent := readFileContent(searchPath)
		if strings.Contains(searchContent, "store.") || strings.Contains(searchContent, "/store") {
			tier2 += 0.5
		}
	}

	if tier2 > 5 {
		tier2 = 5
	}

	score := int(tier1 + tier2)
	if score > 10 {
		score = 10
	}
	return score
}

func scoreWorkflows(dir string) int {
	cliDir := filepath.Join(dir, "internal", "cli")
	entries, err := os.ReadDir(cliDir)
	if err != nil {
		return 0
	}

	workflowPrefixes := []string{"stale", "orphan", "triage", "load", "overdue", "standup", "deps", "workflow"}

	compoundCommands := 0
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") {
			continue
		}

		name := strings.ToLower(e.Name())

		// Detect workflow commands by filename pattern
		isWorkflowFile := false
		for _, prefix := range workflowPrefixes {
			if strings.HasPrefix(name, prefix) {
				isWorkflowFile = true
				break
			}
		}
		if isWorkflowFile {
			compoundCommands++
			continue
		}

		content := readFileContent(filepath.Join(cliDir, e.Name()))

		// Count files that make 2+ different API calls in a single RunE.
		apiCalls := 0
		if strings.Contains(content, "c.Get(") || strings.Contains(content, "c.Get (") {
			apiCalls++
		}
		if strings.Contains(content, "c.Post(") || strings.Contains(content, "c.Post (") {
			apiCalls++
		}
		if strings.Contains(content, "c.Put(") || strings.Contains(content, "c.Put (") {
			apiCalls++
		}
		if strings.Contains(content, "c.Delete(") || strings.Contains(content, "c.Delete (") {
			apiCalls++
		}
		// Also count store operations as compound behavior.
		if strings.Contains(content, "store.") || strings.Contains(content, "/store") {
			apiCalls++
		}
		if apiCalls >= 2 {
			compoundCommands++
		}
	}

	switch {
	case compoundCommands >= 7:
		return 10
	case compoundCommands >= 5:
		return 8
	case compoundCommands >= 3:
		return 6
	case compoundCommands >= 2:
		return 4
	case compoundCommands >= 1:
		return 2
	default:
		return 0
	}
}

func scoreInsight(dir string) int {
	cliDir := filepath.Join(dir, "internal", "cli")
	entries, err := os.ReadDir(cliDir)
	if err != nil {
		return 0
	}

	insightPrefixes := []string{"health", "similar", "bottleneck", "trends", "patterns", "forecast"}
	found := 0
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") {
			continue
		}
		name := strings.ToLower(e.Name())
		for _, prefix := range insightPrefixes {
			if strings.HasPrefix(name, prefix) {
				found++
				break
			}
		}
	}

	switch {
	case found >= 6:
		return 10
	case found >= 5:
		return 9
	case found >= 4:
		return 8
	case found >= 3:
		return 6
	case found >= 2:
		return 4
	case found >= 1:
		return 2
	default:
		return 0
	}
}

type openAPISecurityScheme struct {
	Name string
	Type string
	In   string
}

type openAPISpecInfo struct {
	Paths           []string
	SecuritySchemes []openAPISecurityScheme
}

func loadOpenAPISpec(specPath string) *openAPISpecInfo {
	if specPath == "" {
		return nil
	}

	data, err := os.ReadFile(specPath)
	if err != nil {
		return nil
	}

	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil
	}

	info := &openAPISpecInfo{}

	if paths, ok := raw["paths"].(map[string]any); ok {
		for path := range paths {
			info.Paths = append(info.Paths, path)
		}
		slices.Sort(info.Paths)
	}

	if components, ok := raw["components"].(map[string]any); ok {
		if securitySchemes, ok := components["securitySchemes"].(map[string]any); ok {
			for schemeName, value := range securitySchemes {
				scheme := openAPISecurityScheme{Name: schemeName}
				if fields, ok := value.(map[string]any); ok {
					scheme.Type = strings.ToLower(asString(fields["scheme"]))
					if scheme.Type == "" {
						scheme.Type = strings.ToLower(asString(fields["type"]))
					}
					scheme.In = strings.ToLower(asString(fields["in"]))
				}
				info.SecuritySchemes = append(info.SecuritySchemes, scheme)
			}
		}
	}

	return info
}

func scorePathValidity(dir, specPath string) int {
	if specPath == "" {
		return 5
	}

	spec := loadOpenAPISpec(specPath)
	if spec == nil || len(spec.Paths) == 0 {
		return 5
	}

	pathRe := regexp.MustCompile(`\bpath\s*[:=]\s*"([^"]+)"`)
	cmdFiles := sampleCommandFiles(dir, 10)
	if len(cmdFiles) == 0 {
		return 0
	}

	total := 0
	matches := 0
	for _, content := range cmdFiles {
		match := pathRe.FindStringSubmatch(content)
		if len(match) < 2 {
			continue
		}
		total++
		if specPathExists(spec.Paths, match[1]) {
			matches++
		}
	}

	if total == 0 {
		return 0
	}
	return (matches * 10) / total
}

func scoreAuthProtocol(dir, specPath string) int {
	if specPath == "" {
		return 5
	}

	spec := loadOpenAPISpec(specPath)
	if spec == nil || len(spec.SecuritySchemes) == 0 {
		return 5
	}

	clientContent := readFileContent(filepath.Join(dir, "internal", "client", "client.go"))
	configContent := readFileContent(filepath.Join(dir, "internal", "config", "config.go"))
	if clientContent == "" {
		return 0
	}

	score := 0
	authHeaderMatched := false
	headerNameMatched := false
	queryMatched := false
	envMatched := false

	for _, scheme := range spec.SecuritySchemes {
		nameLower := strings.ToLower(scheme.Name)
		switch {
		case strings.Contains(nameLower, "bot"):
			if strings.Contains(clientContent, `"Bot "`) || strings.Contains(clientContent, "`Bot `") {
				authHeaderMatched = true
			}
		case strings.Contains(nameLower, "bearer") || scheme.Type == "bearer" || scheme.Type == "http":
			if strings.Contains(clientContent, `"Bearer "`) || strings.Contains(clientContent, "`Bearer `") {
				authHeaderMatched = true
			}
		case strings.Contains(nameLower, "basic") || scheme.Type == "basic":
			if strings.Contains(clientContent, `"Basic "`) || strings.Contains(clientContent, "`Basic `") {
				authHeaderMatched = true
			}
		}

		headerName := "Authorization"
		if strings.Contains(nameLower, "bot") {
			headerName = "Authorization"
		}
		if strings.Contains(clientContent, `Header.Set("`+headerName+`"`) ||
			strings.Contains(clientContent, `Header.Add("`+headerName+`"`) {
			headerNameMatched = true
		}

		if scheme.In == "query" && (strings.Contains(clientContent, ".Query()") || strings.Contains(clientContent, "url.Values") || strings.Contains(clientContent, "RawQuery")) {
			queryMatched = true
		}

		envNeedle := sanitizeEnvName(scheme.Name)
		if envNeedle != "" && strings.Contains(strings.ToUpper(configContent), envNeedle) {
			envMatched = true
		}
	}

	if authHeaderMatched {
		score += 3
	}
	if headerNameMatched {
		score += 3
	}
	if queryMatched {
		score += 2
	}
	if envMatched {
		score += 2
	}
	if score > 10 {
		score = 10
	}
	return score
}

func scoreDataPipelineIntegrity(dir string) int {
	score := 0
	syncContent := readFileContent(filepath.Join(dir, "internal", "cli", "sync.go"))
	searchContent := readFileContent(filepath.Join(dir, "internal", "cli", "search.go"))
	storeContent := readFileContent(filepath.Join(dir, "internal", "store", "store.go"))

	if syncContent != "" && (strings.Contains(syncContent, "/store") || strings.Contains(syncContent, "store.")) {
		score++
	}

	domainUpsertRe := regexp.MustCompile(`\.Upsert[A-Z]\w*\(`)
	genericUpsertRe := regexp.MustCompile(`\.Upsert\(`)
	if domainUpsertRe.MatchString(syncContent) {
		score += 3
	} else if genericUpsertRe.MatchString(syncContent) {
		score += 0
	}

	domainSearchRe := regexp.MustCompile(`\.Search[A-Z]\w*\(`)
	genericSearchRe := regexp.MustCompile(`\.Search\(`)
	if domainSearchRe.MatchString(searchContent) {
		score += 3
	} else if genericSearchRe.MatchString(searchContent) {
		score += 0
	}

	score += scoreDomainTables(storeContent)
	if score > 10 {
		score = 10
	}
	return score
}

func scoreSyncCorrectness(dir string) int {
	content := readFileContent(filepath.Join(dir, "internal", "cli", "sync.go"))
	if content == "" {
		return 0
	}

	score := 0
	if hasNonEmptySyncResources(content) {
		score += 2
	}
	if strings.Contains(content, "{") {
		score += 3
	}
	if strings.Contains(content, "GetSyncState") || strings.Contains(content, "sync_state") {
		score += 2
	}
	if strings.Contains(content, "SaveSyncState") {
		score++
	}
	if strings.Contains(content, "paginatedGet") || strings.Contains(content, "hasNextPage") || strings.Contains(content, "endCursor") || strings.Contains(content, "cursor") {
		score += 2
	}
	if score > 10 {
		score = 10
	}
	return score
}

func scoreTypeFidelity(dir string) int {
	score := 0
	cmdFiles := sampleCommandFiles(dir, 10)
	if len(cmdFiles) == 0 {
		return 0
	}

	flagDeclRe := regexp.MustCompile(`Flags\(\)\.(StringVar|IntVar|StringVarP|IntVarP)\(&[^,]+,\s*"([^"]+)"(?:,\s*[^,]+){1,2},\s*"([^"]*)"`)
	requiredRe := regexp.MustCompile(`MarkFlagRequired\("([^"]+)"\)`)

	totalIDFlags := 0
	stringIDFlags := 0
	requiredCount := 0
	descWordCount := 0
	descCount := 0

	for _, content := range cmdFiles {
		for _, match := range flagDeclRe.FindAllStringSubmatch(content, -1) {
			name := strings.ToLower(match[2])
			if strings.Contains(name, "id") {
				totalIDFlags++
				if strings.HasPrefix(match[1], "StringVar") {
					stringIDFlags++
				}
			}
			descWordCount += len(strings.Fields(match[3]))
			descCount++
		}
		requiredCount += len(requiredRe.FindAllStringSubmatch(content, -1))
	}

	if totalIDFlags == 0 || stringIDFlags == totalIDFlags {
		score += 2
	}
	if requiredCount >= 3 {
		score++
	}
	if descCount > 0 && descWordCount/descCount > 5 {
		score++
	}

	allCLI := ""
	for _, content := range sampleCommandFiles(dir, 0) {
		allCLI += content
	}
	allCLI += readFileContent(filepath.Join(dir, "internal", "cli", "helpers.go"))
	allCLI += readFileContent(filepath.Join(dir, "internal", "cli", "root.go"))
	if !strings.Contains(allCLI, "var _ = strings.ReplaceAll") && !strings.Contains(allCLI, "var _ = fmt.Sprintf") {
		score++
	}

	if score > 5 {
		score = 5
	}
	return score
}

func scoreDeadCode(dir string) int {
	deadFlags := 0
	deadFunctions := 0
	cliDir := filepath.Join(dir, "internal", "cli")
	rootContent := readFileContent(filepath.Join(cliDir, "root.go"))
	helpersContent := readFileContent(filepath.Join(cliDir, "helpers.go"))
	if rootContent == "" && helpersContent == "" {
		return 0
	}

	flagRe := regexp.MustCompile(`&flags\.(\w+)`)
	flagNames := uniqueMatches(flagRe, rootContent)
	otherCLI := readOtherGoFiles(cliDir, map[string]bool{"root.go": true})
	for _, name := range flagNames {
		if !strings.Contains(otherCLI, "flags."+name) {
			deadFlags++
		}
	}

	funcRe := regexp.MustCompile(`(?m)^func\s+([A-Za-z_]\w*)\s*\(`)
	funcNames := uniqueMatches(funcRe, helpersContent)
	otherHelpers := readOtherGoFiles(cliDir, map[string]bool{"helpers.go": true})
	for _, name := range funcNames {
		if !strings.Contains(otherHelpers, name+"(") {
			deadFunctions++
		}
	}

	score := 5 - (deadFlags + deadFunctions)
	if score < 0 {
		return 0
	}
	return score
}

// sampleCommandFiles reads up to n command files from internal/cli/.
// If n <= 0, reads all command files.
func sampleCommandFiles(dir string, n int) []string {
	cliDir := filepath.Join(dir, "internal", "cli")
	entries, err := os.ReadDir(cliDir)
	if err != nil {
		return nil
	}
	infra := map[string]bool{
		"helpers.go": true, "root.go": true, "doctor.go": true, "auth.go": true,
		"export.go": true, "import.go": true, "search.go": true, "sync.go": true,
		"tail.go": true, "analytics.go": true,
	}
	var files []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") {
			continue
		}
		if infra[e.Name()] {
			continue
		}
		content := readFileContent(filepath.Join(cliDir, e.Name()))
		if content != "" {
			files = append(files, content)
		}
		if n > 0 && len(files) >= n {
			break
		}
	}
	return files
}

func specPathExists(specPaths []string, actual string) bool {
	for _, candidate := range specPaths {
		if matchSpecPath(candidate, actual) || matchSpecPath(actual, candidate) {
			return true
		}
	}
	return false
}

func matchSpecPath(pattern, actual string) bool {
	patternParts := splitPath(pattern)
	actualParts := splitPath(actual)
	if len(patternParts) != len(actualParts) {
		return false
	}
	for i := range patternParts {
		part := patternParts[i]
		if strings.HasPrefix(part, "{") && strings.HasSuffix(part, "}") {
			continue
		}
		if part != actualParts[i] {
			return false
		}
	}
	return true
}

func splitPath(path string) []string {
	trimmed := strings.Trim(path, "/")
	if trimmed == "" {
		return nil
	}
	return strings.Split(trimmed, "/")
}

func sanitizeEnvName(name string) string {
	name = strings.ToUpper(name)
	var b strings.Builder
	lastUnderscore := false
	for _, r := range name {
		if (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			lastUnderscore = false
			continue
		}
		if !lastUnderscore {
			b.WriteByte('_')
			lastUnderscore = true
		}
	}
	return strings.Trim(b.String(), "_")
}

func scoreDomainTables(storeContent string) int {
	if storeContent == "" {
		return 0
	}
	createTableRe := regexp.MustCompile(`(?is)CREATE TABLE[^()]*\((.*?)\)`)
	columnTables := 0
	for _, match := range createTableRe.FindAllStringSubmatch(storeContent, -1) {
		columnCount := 0
		for _, line := range strings.Split(match[1], "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "--") {
				continue
			}
			upper := strings.ToUpper(line)
			if strings.HasPrefix(upper, "PRIMARY KEY") || strings.HasPrefix(upper, "FOREIGN KEY") || strings.HasPrefix(upper, "UNIQUE") || strings.HasPrefix(upper, "CONSTRAINT") {
				continue
			}
			columnCount++
		}
		if columnCount >= 5 {
			columnTables++
		}
	}
	if columnTables > 0 {
		return 3
	}
	return 0
}

func hasNonEmptySyncResources(content string) bool {
	if strings.Contains(content, "[]string{}") || strings.Contains(content, "return nil") {
		return false
	}
	if strings.Contains(content, "defaultSyncResources()") || strings.Contains(content, "syncResources") {
		listRe := regexp.MustCompile(`\[\]string\{([^}]*)\}`)
		for _, match := range listRe.FindAllStringSubmatch(content, -1) {
			if strings.TrimSpace(match[1]) != "" {
				return true
			}
		}
		if strings.Contains(content, "defaultSyncResources") {
			return true
		}
	}
	return false
}

func uniqueMatches(re *regexp.Regexp, content string) []string {
	seen := map[string]bool{}
	var out []string
	for _, match := range re.FindAllStringSubmatch(content, -1) {
		if len(match) < 2 || seen[match[1]] {
			continue
		}
		seen[match[1]] = true
		out = append(out, match[1])
	}
	return out
}

func readOtherGoFiles(dir string, skip map[string]bool) string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}
	var b strings.Builder
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || skip[entry.Name()] {
			continue
		}
		b.WriteString(readFileContent(filepath.Join(dir, entry.Name())))
		b.WriteByte('\n')
	}
	return b.String()
}

func asString(v any) string {
	switch value := v.(type) {
	case string:
		return value
	case fmt.Stringer:
		return value.String()
	case float64:
		return strconv.FormatFloat(value, 'f', -1, 64)
	default:
		return ""
	}
}

// hasPlaceholderValues checks if file content contains common placeholder values
// that indicate unpolished examples.
func hasPlaceholderValues(content string) bool {
	placeholders := []string{"abc123", `"value"`, "my-resource", "your-key-here", "USER/tap"}
	for _, p := range placeholders {
		if strings.Contains(content, p) {
			return true
		}
	}
	return false
}

// hasQualityDescription checks if a command file has a meaningful Short description.
// Returns true if the description is multi-word and doesn't just repeat the verb.
func hasQualityDescription(content string) bool {
	idx := strings.Index(content, "Short:")
	if idx < 0 {
		return false
	}
	// Extract the Short value (between quotes)
	rest := content[idx:]
	q1 := strings.Index(rest, `"`)
	if q1 < 0 {
		return false
	}
	q2 := strings.Index(rest[q1+1:], `"`)
	if q2 < 0 {
		return false
	}
	desc := rest[q1+1 : q1+1+q2]
	// Quality: must be > 10 chars and contain a space (multi-word)
	return len(desc) > 10 && strings.Contains(desc, " ")
}

// hasLazyDescription checks if a command has a 1-word or very short description.
func hasLazyDescription(content string) bool {
	idx := strings.Index(content, "Short:")
	if idx < 0 {
		return false
	}
	rest := content[idx:]
	q1 := strings.Index(rest, `"`)
	if q1 < 0 {
		return false
	}
	q2 := strings.Index(rest[q1+1:], `"`)
	if q2 < 0 {
		return false
	}
	desc := rest[q1+1 : q1+1+q2]
	words := strings.Fields(desc)
	return len(words) <= 2
}

func readFileContent(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(data)
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func computeGrade(percentage int) string {
	switch {
	case percentage >= 80:
		return "A"
	case percentage >= 65:
		return "B"
	case percentage >= 50:
		return "C"
	case percentage >= 35:
		return "D"
	default:
		return "F"
	}
}

func buildGapReport(s SteinerScore) []string {
	var gaps []string
	dimensions := []struct {
		name  string
		score int
	}{
		{"output_modes", s.OutputModes},
		{"auth", s.Auth},
		{"error_handling", s.ErrorHandling},
		{"terminal_ux", s.TerminalUX},
		{"readme", s.README},
		{"doctor", s.Doctor},
		{"agent_native", s.AgentNative},
		{"local_cache", s.LocalCache},
		{"breadth", s.Breadth},
		{"vision", s.Vision},
		{"workflows", s.Workflows},
		{"insight", s.Insight},
		{"path_validity", s.PathValidity},
		{"auth_protocol", s.AuthProtocol},
		{"data_pipeline_integrity", s.DataPipelineIntegrity},
		{"sync_correctness", s.SyncCorrectness},
		{"type_fidelity", s.TypeFidelity},
		{"dead_code", s.DeadCode},
	}
	for _, d := range dimensions {
		max := 10
		if d.name == "type_fidelity" || d.name == "dead_code" {
			max = 5
		}
		if d.score < max/2 {
			gaps = append(gaps, fmt.Sprintf("%s scored %d/%d - needs improvement", d.name, d.score, max))
		}
	}
	return gaps
}

func buildCompetitorScores(ourTotal int, pipelineDir string) []CompScore {
	research, err := LoadResearch(pipelineDir)
	if err != nil {
		return nil
	}
	var scores []CompScore
	for _, alt := range research.Alternatives {
		theirScore := estimateCompetitorTotal(alt)
		scores = append(scores, CompScore{
			Name:       alt.Name,
			OurScore:   ourTotal,
			TheirScore: theirScore,
			WeWin:      ourTotal > theirScore,
		})
	}
	return scores
}

func estimateCompetitorTotal(alt Alternative) int {
	score := 0
	if alt.HasJSON {
		score += 6 // output_modes partial credit
	}
	if alt.HasAuth {
		score += 5 // auth partial credit
	}
	// Assume basic error handling and terminal UX
	score += 3
	score += 3
	// README and doctor are unknowns - give partial credit
	score += 4
	score += 2
	// Agent native: partial if they have JSON
	if alt.HasJSON {
		score += 3
	}
	return score
}

func writeScorecardMD(sc *Scorecard, pipelineDir string) error {
	if err := os.MkdirAll(pipelineDir, 0o755); err != nil {
		return err
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("# Scorecard: %s\n\n", sc.APIName))
	b.WriteString(fmt.Sprintf("**Overall Grade: %s** (%d%%)\n\n", sc.OverallGrade, sc.Steinberger.Percentage))

	// Steinberger dimensions table
	b.WriteString("## Quality Dimensions\n\n")
	b.WriteString("| Dimension | Score |\n")
	b.WriteString("|-----------|-------|\n")
	s := sc.Steinberger
	dimensions := []struct {
		name  string
		score int
	}{
		{"Output Modes", s.OutputModes},
		{"Auth", s.Auth},
		{"Error Handling", s.ErrorHandling},
		{"Terminal UX", s.TerminalUX},
		{"README", s.README},
		{"Doctor", s.Doctor},
		{"Agent Native", s.AgentNative},
		{"Local Cache", s.LocalCache},
		{"Breadth", s.Breadth},
		{"Vision", s.Vision},
		{"Workflows", s.Workflows},
		{"Insight", s.Insight},
		{"Path Validity", s.PathValidity},
		{"Auth Protocol", s.AuthProtocol},
		{"Data Pipeline Integrity", s.DataPipelineIntegrity},
		{"Sync Correctness", s.SyncCorrectness},
	}
	for _, d := range dimensions {
		bar := strings.Repeat("#", d.score) + strings.Repeat(".", 10-d.score)
		b.WriteString(fmt.Sprintf("| %s | %d/10 %s |\n", d.name, d.score, bar))
	}
	typeDimensions := []struct {
		name  string
		score int
	}{
		{"Type Fidelity", s.TypeFidelity},
		{"Dead Code", s.DeadCode},
	}
	for _, d := range typeDimensions {
		bar := strings.Repeat("#", d.score) + strings.Repeat(".", 5-d.score)
		b.WriteString(fmt.Sprintf("| %s | %d/5 %s |\n", d.name, d.score, bar))
	}
	b.WriteString(fmt.Sprintf("| **Total** | **%d/100** |\n\n", s.Total))

	// Competitor comparison
	if len(sc.CompetitorScores) > 0 {
		b.WriteString("## Competitor Comparison\n\n")
		b.WriteString("| Competitor | Ours | Theirs | Winner |\n")
		b.WriteString("|------------|------|--------|--------|\n")
		for _, cs := range sc.CompetitorScores {
			winner := "Them"
			if cs.WeWin {
				winner = "Us"
			}
			b.WriteString(fmt.Sprintf("| %s | %d | %d | %s |\n", cs.Name, cs.OurScore, cs.TheirScore, winner))
		}
		b.WriteString("\n")
	}

	// Gap report
	if len(sc.GapReport) > 0 {
		b.WriteString("## Gaps\n\n")
		for _, g := range sc.GapReport {
			b.WriteString(fmt.Sprintf("- %s\n", g))
		}
		b.WriteString("\n")
	}

	return os.WriteFile(filepath.Join(pipelineDir, "scorecard.md"), []byte(b.String()), 0o644)
}

// LoadScorecard reads a scorecard from a pipeline directory's scorecard.json.
func LoadScorecard(pipelineDir string) (*Scorecard, error) {
	data, err := os.ReadFile(filepath.Join(pipelineDir, "scorecard.json"))
	if err != nil {
		return nil, err
	}
	var sc Scorecard
	if err := json.Unmarshal(data, &sc); err != nil {
		return nil, err
	}
	return &sc, nil
}
