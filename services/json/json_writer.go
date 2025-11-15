package json

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

func WriteTemplate(path string, contents string, overwrite bool) (string, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create dirs for %s: %w", path, err)
	}

	finalPath := path
	if !overwrite {
		if _, err := os.Stat(finalPath); err == nil {
			base := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
			ext := filepath.Ext(path)
			for i := 2; ; i++ {
				try := filepath.Join(dir, fmt.Sprintf("%s_%d%s", base, i, ext))
				if _, err := os.Stat(try); os.IsNotExist(err) {
					finalPath = try
					break
				}
			}
		}
	}

	if err := os.WriteFile(finalPath, []byte(contents), 0o644); err != nil {
		return "", fmt.Errorf("write template %s: %w", finalPath, err)
	}
	return finalPath, nil
}

var placeholderRe = regexp.MustCompile(`"\$\{([a-zA-Z0-9_]+)(?::([a-zA-Z0-9_]+))?\}"`)

// RenderJSONCell takes a JSON template string containing placeholders like
//
//	"${id:number}", "${customerid:uuid}", "${name:string}"
//
// and a map of values, and returns compact valid JSON.
func RenderJSONCell(tpl string, kv map[string]string) (string, error) {
	// 1) Substitute placeholders with JSON literals
	rendered := placeholderRe.ReplaceAllFunc([]byte(tpl), func(match []byte) []byte {
		sub := placeholderRe.FindSubmatch(match)
		if len(sub) < 2 {
			// shouldn't happen; just return original
			return match
		}

		key := string(sub[1])
		typ := ""
		if len(sub) >= 3 {
			typ = string(sub[2])
		}

		val, ok := kv[key]
		if !ok {
			return []byte(`null`)
		}

		return []byte(renderJSONLiteral(val, typ))
	})

	// 2) Validate JSON and compact it
	var anyJSON any
	if err := json.Unmarshal(rendered, &anyJSON); err != nil {
		return "", fmt.Errorf("invalid json after placeholder substitution: %w\nrendered: %s", err, rendered)
	}

	compact, err := json.Marshal(anyJSON)
	if err != nil {
		return "", fmt.Errorf("compact json: %w", err)
	}
	return string(compact), nil
}

// renderJSONLiteral converts a raw string + type into a valid JSON literal.
func renderJSONLiteral(raw, typ string) string {
	typ = strings.ToLower(strings.TrimSpace(typ))

	// string-like types: always quote
	if typ == "" ||
		typ == "string" ||
		typ == "email" ||
		typ == "uuid" ||
		typ == "timestamp" ||
		typ == "alphanumeric" ||
		typ == "range" {
		b, _ := json.Marshal(raw)
		return string(b)
	}

	switch typ {
	case "number", "int", "integer", "decimal", "float":
		s := strings.TrimSpace(raw)
		if s == "" {
			return "0"
		}
		if _, err := strconv.ParseFloat(s, 64); err != nil {
			b, _ := json.Marshal(raw)
			return string(b)
		}
		return s

	case "bool", "boolean":
		s := strings.ToLower(strings.TrimSpace(raw))
		if s == "true" || s == "false" {
			return s
		}
		// fallback to quoted string
		b, _ := json.Marshal(raw)
		return string(b)

	default:
		// unknown type: safest to treat as string
		b, _ := json.Marshal(raw)
		return string(b)
	}
}

func PerformTokenReplacement(raw []byte, vars map[string]string) ([]byte, error) {
	s := string(raw)
	pairs := make([]string, 0, len(vars)*2)

	for k, v := range vars {
		pairs = append(pairs, "{{"+k+"}}", v)
		pairs = append(pairs, "{{ "+k+" }}", v)
	}
	r := strings.NewReplacer(pairs...)
	s = r.Replace(s)

	unfilled := regexp.MustCompile(`\{\{\s*([^}]+?)\s*\}\}`)
	matches := unfilled.FindAllStringSubmatch(s, -1)
	if len(matches) > 0 {
		var missing []string
		for _, m := range matches {
			if len(m) > 1 {
				missing = append(missing, strings.TrimSpace(m[1]))
			}
		}
		return nil, fmt.Errorf("missing input for %v", missing)
	}

	return []byte(s), nil
}

// MarshalTemplate takes an inferred template AST (maps, arrays, models.Placeholder, etc.)
// and pretty-prints it as JSON. models.Placeholder.MarshalJSON ensures placeholders
// render as "${key:type}" strings.
func MarshalTemplate(v any) ([]byte, error) {
	return json.MarshalIndent(v, "", "  ")
}

func SanitizeFileStem(s string) string {
	s = strings.ToLower(regexp.MustCompile(`[^a-z0-9_-]+`).ReplaceAllString(s, "_"))
	s = strings.Trim(s, "_-")
	if s == "" {
		s = "json"
	}
	return s
}
