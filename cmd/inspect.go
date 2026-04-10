package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/kream404/spoof/models"
	json_service "github.com/kream404/spoof/services/json"
	"github.com/olekukonko/tablewriter"
)

// lightweight model to unmarshal bundles and configs for inspection without needing to define all fields
type inspectConfig struct {
	Metadata  *models.ConfigMetadata        `json:"metadata,omitempty"`
	Variables map[string]models.VariableDoc `json:"variables,omitempty"`
}

type InspectResult struct {
	RootPath        string
	RootType        string // bundle | config | unknown
	ExecutionOrder  []ExecutionNode
	Variables       map[string]*VariableInfo
	Warnings        []string
	VisitedFiles    map[string]bool
	ResolvedVars    map[string]string
	DiscoveredFiles int
}

type ExecutionNode struct {
	Path         string
	Depth        int
	Kind         string // bundle-child | config | unresolved | candidate
	Resolved     bool
	Requires     []string
	SourceFrom   string
	Exists       bool
	IsSpeculated bool
}

type VariableInfo struct {
	Name          string
	Required      bool
	DefaultValue  string
	Type          string
	Description   string
	DefinedIn     []string
	UsedIn        []string
	Examples      []string
	AllowedValues []string
	Declared      bool
}

var tokenRegex = regexp.MustCompile(`\{\{\s*([A-Z0-9_]+)(?:\s*\|\s*([^}]+))?\s*\}\}`)

func runInspect() error {
	if strings.TrimSpace(configPath) == "" {
		return fmt.Errorf("config path is required for inspect mode")
	}

	varsMap := parseInjectedVars(injectVars)

	result := &InspectResult{
		RootPath:     configPath,
		Variables:    map[string]*VariableInfo{},
		Warnings:     []string{},
		VisitedFiles: map[string]bool{},
		ResolvedVars: varsMap,
	}

	if err := inspectPath(configPath, 0, "", varsMap, result); err != nil {
		return err
	}

	renderInspectResult(result)
	return nil
}

func inspectPath(path string, depth int, sourceFrom string, varsMap map[string]string, result *InspectResult) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to resolve path %q: %w", path, err)
	}

	if result.VisitedFiles[absPath] {
		return nil
	}
	result.VisitedFiles[absPath] = true
	result.DiscoveredFiles++

	raw, err := os.ReadFile(absPath)
	if err != nil {
		return fmt.Errorf("could not read %q: %w", path, err)
	}

	if depth == 0 {
		result.RootType = detectFileType(raw)
		if result.RootType == "" {
			result.RootType = "unknown"
		}
	}

	discoverVariables(absPath, raw, result)

	if isBundle(raw) {
		var bundle models.Bundle
		if err := json.Unmarshal(raw, &bundle); err != nil {
			return fmt.Errorf("failed to unmarshal bundle %q: %w", path, err)
		}

		mergeDeclaredVariables(absPath, bundle.Variables, result)

		for _, f := range bundle.Files {
			src := strings.TrimSpace(f.Source)
			if src == "" {
				continue
			}

			requiredVars := extractTokens(src)
			for _, name := range requiredVars {
				vi := ensureVariable(result, name)
				vi.UsedIn = appendUnique(vi.UsedIn, fmt.Sprintf("%s -> %s", rel(absPath), src))
			}

			resolvedSrc, unresolvedVars, err := tryResolveString(src, varsMap)
			if err != nil {
				result.Warnings = append(result.Warnings,
					fmt.Sprintf("failed resolving source %q in %s: %v", src, rel(absPath), err))
				continue
			}

			if len(unresolvedVars) > 0 {
				result.ExecutionOrder = append(result.ExecutionOrder, ExecutionNode{
					Path:       src,
					Depth:      depth + 1,
					Kind:       "unresolved",
					Resolved:   false,
					Requires:   requiredVars,
					SourceFrom: rel(absPath),
				})

				candidates := expandPathCandidates(src, result)
				if len(candidates) > 0 {
					for _, candidate := range candidates {
						candidatePath, _ := resolveChildPath(absPath, candidate)
						_, statErr := os.Stat(candidatePath)

						result.ExecutionOrder = append(result.ExecutionOrder, ExecutionNode{
							Path:         rel(candidatePath),
							Depth:        depth + 2,
							Kind:         "candidate",
							Resolved:     false,
							SourceFrom:   rel(absPath),
							Exists:       statErr == nil,
							IsSpeculated: true,
						})
					}
				} else {
					result.Warnings = append(result.Warnings,
						fmt.Sprintf("could not resolve %q from %s; missing vars: %s",
							src, rel(absPath), strings.Join(unresolvedVars, ", ")))
				}

				continue
			}

			childPath, resolveNote := resolveChildPath(absPath, resolvedSrc)
			if resolveNote != "" {
				result.Warnings = append(result.Warnings, resolveNote)
			}

			result.ExecutionOrder = append(result.ExecutionOrder, ExecutionNode{
				Path:       rel(childPath),
				Depth:      depth + 1,
				Kind:       "bundle-child",
				Resolved:   true,
				Requires:   requiredVars,
				SourceFrom: rel(absPath),
				Exists:     true,
			})

			if err := inspectPath(childPath, depth+1, absPath, varsMap, result); err != nil {
				result.Warnings = append(result.Warnings, err.Error())
			}
		}

		return nil
	}

	var config inspectConfig
	if err := json.Unmarshal(raw, &config); err == nil {
		mergeDeclaredVariables(absPath, config.Variables, result)
	}

	if depth == 0 || sourceFrom == "" {
		result.ExecutionOrder = append(result.ExecutionOrder, ExecutionNode{
			Path:       rel(absPath),
			Depth:      depth,
			Kind:       "config",
			Resolved:   true,
			SourceFrom: rel(sourceFrom),
			Exists:     true,
		})
	}

	return nil
}

func detectFileType(raw []byte) string {
	if isBundle(raw) {
		return "bundle"
	}

	var fc models.FileConfig
	if err := json.Unmarshal(raw, &fc); err == nil && len(fc.Files) > 0 {
		return "config"
	}

	return ""
}

func isBundle(raw []byte) bool {
	var b models.Bundle
	if err := json.Unmarshal(raw, &b); err != nil {
		return false
	}
	if len(b.Files) == 0 {
		return false
	}
	for _, f := range b.Files {
		if strings.TrimSpace(f.Source) != "" {
			return true
		}
	}
	return false
}

func discoverVariables(path string, raw []byte, result *InspectResult) {
	matches := tokenRegex.FindAllSubmatch(raw, -1)
	for _, m := range matches {
		name := strings.TrimSpace(string(m[1]))
		def := ""
		if len(m) > 2 {
			def = strings.TrimSpace(string(m[2]))
		}

		vi := ensureVariable(result, name)
		vi.DefinedIn = appendUnique(vi.DefinedIn, rel(path))
		vi.UsedIn = appendUnique(vi.UsedIn, rel(path))

		if def != "" && vi.DefaultValue == "" {
			vi.DefaultValue = def
		}
		if def != "" && !vi.Declared {
			vi.Required = false
		}
	}
}

func mergeDeclaredVariables(path string, declared map[string]models.VariableDoc, result *InspectResult) {
	for name, doc := range declared {
		vi := ensureVariable(result, name)
		vi.DefinedIn = appendUnique(vi.DefinedIn, rel(path))

		if doc.Type != "" {
			vi.Type = doc.Type
		}
		if doc.Description != "" {
			vi.Description = doc.Description
		}
		if doc.Default != "" && vi.DefaultValue == "" {
			vi.DefaultValue = doc.Default
		}
		if doc.Example != "" {
			vi.Examples = appendUnique(vi.Examples, doc.Example)
		}
		if len(doc.AllowedValues) > 0 {
			for _, val := range doc.AllowedValues {
				vi.AllowedValues = appendUnique(vi.AllowedValues, val)
			}
		}

		if doc.Required {
			vi.Required = true
		}

		if doc.Default != "" {
			vi.DefaultValue = doc.Default
			vi.Required = false
		}

		vi.Declared = true
	}
}

func ensureVariable(result *InspectResult, name string) *VariableInfo {
	if vi, ok := result.Variables[name]; ok {
		return vi
	}

	vi := &VariableInfo{
		Name:          name,
		Required:      true,
		DefinedIn:     []string{},
		UsedIn:        []string{},
		Examples:      []string{},
		AllowedValues: []string{},
	}
	result.Variables[name] = vi
	return vi
}

func extractTokens(input string) []string {
	matches := tokenRegex.FindAllStringSubmatch(input, -1)
	out := make([]string, 0, len(matches))
	for _, m := range matches {
		if len(m) > 1 {
			out = appendUnique(out, strings.TrimSpace(m[1]))
		}
	}
	return out
}

func tryResolveString(input string, varsMap map[string]string) (string, []string, error) {
	unresolved := []string{}

	matches := tokenRegex.FindAllStringSubmatch(input, -1)
	for _, m := range matches {
		name := strings.TrimSpace(m[1])
		def := ""
		if len(m) > 2 {
			def = strings.TrimSpace(m[2])
		}

		if _, ok := varsMap[name]; !ok && def == "" {
			unresolved = appendUnique(unresolved, name)
		}
	}

	if len(unresolved) > 0 {
		return input, unresolved, nil
	}

	out, err := json_service.PerformTokenReplacement([]byte(input), varsMap)
	if err != nil {
		return "", nil, err
	}

	return string(out), nil, nil
}

// rendering functions
func renderInspectResult(r *InspectResult) {
	fmt.Println()
	fmt.Printf("Inspect: %s\n", rel(r.RootPath))
	fmt.Printf("Type: %s\n", r.RootType)
	fmt.Printf("Discovered files: %d\n", r.DiscoveredFiles)

	if len(r.ResolvedVars) > 0 {
		fmt.Println("\nInjected values")
		keys := make([]string, 0, len(r.ResolvedVars))
		for k := range r.ResolvedVars {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			fmt.Printf("  %s = %s\n", k, r.ResolvedVars[k])
		}
	}

	renderExecutionTable(r)
	renderVariablesTable(r)
	renderMissingVariables(r)
	renderWarnings(r)
	renderSampleCommand(r)

	fmt.Println()
}

func renderExecutionTable(r *InspectResult) {
	if len(r.ExecutionOrder) == 0 {
		return
	}

	headers := []string{"#", "Path", "Status", "Requires"}
	printSectionHeader("Execution Plan")

	table := tablewriter.NewWriter(os.Stdout)
	table.Header(headers)

	for i, n := range r.ExecutionOrder {
		status := "resolved"
		switch n.Kind {
		case "candidate":
			if n.Exists {
				status = "candidate exists"
			} else {
				status = "candidate missing"
			}
		case "unresolved":
			status = "unresolved"
		default:
			if !n.Resolved {
				status = "unresolved"
			}
		}

		path := fmt.Sprintf("%s%s", strings.Repeat("  ", n.Depth), n.Path)

		table.Append([]string{
			fmt.Sprintf("%d", i+1),
			path,
			status,
			strings.Join(n.Requires, ", "),
		})
	}

	table.Render()
}

func renderVariablesTable(r *InspectResult) {
	if len(r.Variables) == 0 {
		return
	}

	printSectionHeader("Variables")
	table := tablewriter.NewWriter(os.Stdout)
	table.Header("Name", "Required", "Type", "Default", "Description")

	keys := make([]string, 0, len(r.Variables))
	for k := range r.Variables {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		v := r.Variables[k]

		required := "no"
		if v.Required {
			required = "yes"
		}

		notes := ""

		if v.Required {
			notes = "required"
		}

		if v.Description != "" {
			desc := truncate(v.Description, 100)
			if notes == "" {
				notes = desc
			} else {
				notes += " | " + desc
			}
		}

		if v.Required {
			if _, ok := r.ResolvedVars[v.Name]; !ok {
				if notes == "" {
					notes = "missing"
				} else {
					notes += " | missing"
				}
			}
		}

		table.Append([]string{
			v.Name,
			required,
			v.Type,
			truncate(v.DefaultValue, 32),
			notes,
		})
	}

	table.Render()
}

func renderMissingVariables(r *InspectResult) {
	missing := missingRequiredVars(r)
	if len(missing) == 0 {
		return
	}

	printSectionHeader("Missing required variables")
	table := tablewriter.NewWriter(os.Stdout)
	table.Header("Name", "Example", "Allowed Values")

	for _, m := range missing {
		v := r.Variables[m]
		if v == nil {
			table.Append([]string{m, "", ""})
			continue
		}

		example := ""
		if len(v.Examples) > 0 {
			example = v.Examples[0]
		}

		table.Append([]string{
			v.Name,
			truncate(example, 24),
			truncate(strings.Join(v.AllowedValues, ", "), 72),
		})
	}

	_ = table.Render()

	for _, m := range missing {
		v := r.Variables[m]
		if v == nil || strings.TrimSpace(v.Description) == "" {
			continue
		}
		fmt.Printf("  %s: %s\n", v.Name, truncate(v.Description, 100))
	}
}

func renderWarnings(r *InspectResult) {
	if len(r.Warnings) == 0 {
		return
	}

	fmt.Println("\nWarnings")
	for _, w := range r.Warnings {
		fmt.Printf("  - %s\n", w)
	}
}

func renderSampleCommand(r *InspectResult) {
	merged := make(map[string]string)

	// Keep already supplied vars
	for k, v := range r.ResolvedVars {
		merged[k] = v
	}

	// Fill in any missing required vars
	for _, name := range missingRequiredVars(r) {
		if _, exists := merged[name]; exists {
			continue
		}

		v := r.Variables[name]
		value := "<value>"

		if v != nil {
			if len(v.Examples) > 0 && strings.TrimSpace(v.Examples[0]) != "" {
				value = v.Examples[0]
			} else if len(v.AllowedValues) > 0 && strings.TrimSpace(v.AllowedValues[0]) != "" {
				value = v.AllowedValues[0]
			} else if strings.TrimSpace(v.DefaultValue) != "" {
				value = v.DefaultValue
			}
		}

		merged[name] = value
	}

	if len(merged) == 0 {
		return
	}

	keys := make([]string, 0, len(merged))
	for k := range merged {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%s", k, merged[k]))
	}

	fmt.Println("\nSample command\n")
	fmt.Printf("  spoof -c %s -i %s -f -p <CONNECTION_PROFILE> ", rel(r.RootPath), strings.Join(parts, ","))
	fmt.Println("\n")
}

func estimateTableWidth(headers []string) int {
	width := 1 // borders
	for _, h := range headers {
		width += len(h) + 3 // padding + column separators
	}
	return width
}

func missingRequiredVars(r *InspectResult) []string {
	out := []string{}
	for name, info := range r.Variables {
		if info.Required {
			if _, ok := r.ResolvedVars[name]; !ok {
				out = append(out, name)
			}
		}
	}
	sort.Strings(out)
	return out
}

func printSectionHeader(title string) {
	fmt.Printf("\n%s\n", title)
}

func center(text string, width int) string {
	if len(text) >= width {
		return text
	}
	padding := (width - len(text)) / 2
	return strings.Repeat(" ", padding) + text
}

func appendUnique(in []string, value string) []string {
	for _, v := range in {
		if v == value {
			return in
		}
	}
	return append(in, value)
}

func rel(path string) string {
	if strings.TrimSpace(path) == "" {
		return ""
	}
	cwd, err := os.Getwd()
	if err != nil {
		return path
	}
	r, err := filepath.Rel(cwd, path)
	if err != nil {
		return path
	}
	return r
}

func resolveChildPath(parentAbsPath, src string) (string, string) {
	if filepath.IsAbs(src) {
		return src, ""
	}

	if _, err := os.Stat(src); err == nil {
		abs, err := filepath.Abs(src)
		if err == nil {
			return abs, ""
		}
		return src, ""
	}

	joined := filepath.Join(filepath.Dir(parentAbsPath), src)
	if _, err := os.Stat(joined); err == nil {
		return joined, ""
	}

	abs, err := filepath.Abs(src)
	if err == nil {
		return abs, fmt.Sprintf("could not confirm path %q from cwd or relative to %s", src, rel(parentAbsPath))
	}

	return joined, fmt.Sprintf("could not confirm path %q from cwd or relative to %s", src, rel(parentAbsPath))
}

func expandPathCandidates(input string, result *InspectResult) []string {
	candidates := []string{input}

	matches := tokenRegex.FindAllStringSubmatch(input, -1)
	for _, m := range matches {
		if len(m) < 2 {
			continue
		}

		name := strings.TrimSpace(m[1])
		vi, ok := result.Variables[name]
		if !ok || len(vi.AllowedValues) == 0 {
			return nil
		}

		next := []string{}
		fullMatch := m[0]
		for _, candidate := range candidates {
			for _, allowed := range vi.AllowedValues {
				next = append(next, strings.ReplaceAll(candidate, fullMatch, allowed))
			}
		}
		candidates = next
	}

	return uniqueStrings(candidates)
}

func uniqueStrings(in []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(in))
	for _, v := range in {
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	sort.Strings(out)
	return out
}

func truncate(s string, max int) string {
	s = strings.TrimSpace(s)
	if len(s) <= max {
		return s
	}
	if max <= 3 {
		return s[:max]
	}
	return s[:max-3] + "..."
}
