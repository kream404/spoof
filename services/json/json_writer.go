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

func RenderJSONCell(tpl string, kv map[string]string) (string, error) {
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

func renderJSONLiteral(raw, typ string) string {
	t := strings.ToLower(strings.TrimSpace(typ))

	if t == "number" {
		s := strings.TrimSpace(raw)
		if s == "" {
			return "0"
		}
		if _, err := strconv.ParseFloat(s, 64); err != nil {
			b, _ := json.Marshal(raw)
			return string(b)
		}
		return s
	}

	// default: treat as string
	b, _ := json.Marshal(raw)
	return string(b)
}

func PerformTokenReplacement(raw []byte, vars map[string]string) ([]byte, error) {
	s := string(raw)

	missing := make([]string, 0)
	tokenRE := regexp.MustCompile(`\{\{\s*([^}]+?)\s*\}\}`)
	out := tokenRE.ReplaceAllStringFunc(s, func(full string) string {
		inner := tokenRE.FindStringSubmatch(full)[1]
		inner = strings.TrimSpace(inner)

		// Split on first pipe: "KEY | default value"
		key, def, hasDef := splitKeyDefault(inner)

		if v, ok := vars[key]; ok {
			return v
		}

		if hasDef {
			return def
		}

		missing = append(missing, key)
		return full // leave it unfilled for now; we'll error after.
	})

	if len(missing) > 0 {
		missing = dedupePreserveOrder(missing)
		return nil, fmt.Errorf("missing input for %v", missing)
	}

	if leftovers := tokenRE.FindAllStringSubmatch(out, -1); len(leftovers) > 0 {
		var still []string
		for _, m := range leftovers {
			if len(m) > 1 {
				still = append(still, strings.TrimSpace(m[1]))
			}
		}
		still = dedupePreserveOrder(still)
		return nil, fmt.Errorf("missing input for %v", still)
	}

	return []byte(out), nil
}

func splitKeyDefault(inner string) (key string, def string, hasDef bool) {
	parts := strings.SplitN(inner, "|", 2)
	key = strings.TrimSpace(parts[0])

	if len(parts) == 2 {
		hasDef = true
		def = strings.TrimSpace(parts[1])

		// Optional: support "default:..." sugar
		if strings.HasPrefix(strings.ToLower(def), "default:") {
			def = strings.TrimSpace(def[len("default:"):])
		}
	}
	return
}

func dedupePreserveOrder(in []string) []string {
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, x := range in {
		if _, ok := seen[x]; ok {
			continue
		}
		seen[x] = struct{}{}
		out = append(out, x)
	}
	return out
}

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
