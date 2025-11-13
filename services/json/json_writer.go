package json

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/kream404/spoof/models"
	log "github.com/kream404/spoof/services/logger"
)

type infCtx struct {
	used map[string]int
}

type placeholder struct {
	Key   string
	Quote bool
} // Quote=true -> rendered as "{{ .Key }}", otherwise {{ .Key }}

// isJSON stays here because it's only used for JSON inference.
func isJSON(s string) (bool, any) {
	var v any
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		return false, nil
	}
	switch v.(type) {
	case map[string]any, []any:
		return true, v
	default:
		return false, nil
	}
}

func inferJSONTemplateAndFields(v any, tplName string) (string, []models.Field, error) {
	ctx := &infCtx{used: make(map[string]int)}
	node, fields := jsonInference(ctx, v, nil)

	tplBytes, err := marshalTemplate(node)
	if err != nil {
		return "", nil, err
	}
	return string(tplBytes), fields, nil
}

func jsonInference(ctx *infCtx, v any, path []string) (any, []models.Field) {
	switch t := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(t))
		var fields []models.Field
		for k, val := range t {
			n, f := jsonInference(ctx, val, append(path, k))
			out[k] = n
			fields = append(fields, f...)
		}
		return out, fields

	case []any:
		arr := make([]any, len(t))
		var fields []models.Field
		for i, val := range t {
			n, f := jsonInference(ctx, val, append(path, strconv.Itoa(i)))
			arr[i] = n
			fields = append(fields, f...)
		}
		return arr, fields

	default:
		key := makeKey(ctx, path)

		mf := inferFieldFromValue(key, t)
		ph := placeholder{
			Key: key,
			Quote: mf.Type == "email" ||
				mf.Type == "uuid" ||
				mf.Type == "timestamp" ||
				mf.Type == "alphanumeric" ||
				mf.Type == "",
		}

		return ph, []models.Field{mf}
	}
}

func makeKey(ctx *infCtx, path []string) string {
	base := strings.ToLower(strings.Join(path, "_"))
	base = regexp.MustCompile(`[^a-z0-9_]+`).ReplaceAllString(base, "_")
	base = strings.Trim(base, "_")
	if base == "" {
		base = "value"
	}

	if n, ok := ctx.used[base]; ok {
		ctx.used[base] = n + 1
		return fmt.Sprintf("%s_%d", base, n+1)
	}
	ctx.used[base] = 1
	return base
}

// NOTE: this reuses your unified InferField so JSON leaves behave like CSV columns.
func inferFieldFromValue(name string, v any) models.Field {
	switch x := v.(type) {
	case string:
		f, err := InferField(name, []string{x})
		if err != nil {
			log.Error("inferFieldFromValue string failed, falling back", "err", err)
			return models.Field{Name: name, Type: "unknown"}
		}
		return f

	case float64:
		s := strconv.FormatFloat(x, 'f', -1, 64)
		f, err := InferField(name, []string{s})
		if err != nil {
			log.Error("inferFieldFromValue float failed, falling back", "err", err)
			return models.Field{Name: name, Type: "number"}
		}
		return f

	case bool:
		return models.Field{Name: name, Type: "range", Values: "true, false"}

	case nil:
		return models.Field{Name: name, Value: ""}

	default:
		return models.Field{Name: name, Type: "alphanumeric", Length: 16}
	}
}

func marshalTemplate(v any) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := writeTemplate(&buf, v); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func writeTemplate(w *bytes.Buffer, v any) error {
	switch t := v.(type) {
	case map[string]any:
		w.WriteByte('{')
		i := 0
		keys := make([]string, 0, len(t))
		for k := range t {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			if i > 0 {
				w.WriteByte(',')
			}
			kb, _ := json.Marshal(k)
			w.Write(kb)
			w.WriteByte(':')
			if err := writeTemplate(w, t[k]); err != nil {
				return err
			}
			i++
		}
		w.WriteByte('}')
		return nil

	case []any:
		w.WriteByte('[')
		for i, el := range t {
			if i > 0 {
				w.WriteByte(',')
			}
			if err := writeTemplate(w, el); err != nil {
				return err
			}
		}
		w.WriteByte(']')
		return nil

	case placeholder:
		if t.Quote {
			w.WriteString(`"{{ .`)
			w.WriteString(t.Key)
			w.WriteString(` }}"`)
		} else {
			w.WriteString(`{{ .`)
			w.WriteString(t.Key)
			w.WriteString(` }}`)
		}
		return nil

	case string, float64, bool, nil:
		b, _ := json.Marshal(t)
		w.Write(b)
		return nil

	default:
		b, _ := json.Marshal(t)
		w.Write(b)
		return nil
	}
}

func sanitizeFileStem(s string) string {
	s = strings.ToLower(regexp.MustCompile(`[^a-z0-9_-]+`).ReplaceAllString(s, "_"))
	s = strings.Trim(s, "_-")
	if s == "" {
		s = "json"
	}
	return s
}

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
