package json

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"text/template"

	"github.com/kream404/spoof/fakers"
	"github.com/kream404/spoof/models"
	log "github.com/kream404/spoof/services/logger"
)

func LoadConfig(filepath string) (*models.FileConfig, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	var config models.FileConfig
	err = decoder.Decode(&config)
	if err != nil {
		return nil, err
	}
	return &config, nil
}

var vars = map[string]string{}

func SetVars(m map[string]string) { vars = m }

func PerformVariableInjection(config models.FileConfig) (*models.FileConfig, error) {
	var tokenRE = regexp.MustCompile(`\{\{\s*([A-Za-z0-9_.\-]+)\s*\}\}`)

	raw, err := json.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("marshal: %w", err)
	}
	jsonStr := string(raw)

	// Replace each provided var
	for k, v := range vars {
		re := regexp.MustCompile(`\{\{\s*` + regexp.QuoteMeta(k) + `\s*\}\}`)
		jsonStr = re.ReplaceAllString(jsonStr, v)
	}

	// Fail if any placeholders remain
	if unresolved := tokenRE.FindAllStringSubmatch(jsonStr, -1); len(unresolved) > 0 {
		missing := make([]string, 0, len(unresolved))
		seen := map[string]struct{}{}
		for _, m := range unresolved {
			if len(m) == 2 {
				if _, ok := seen[m[1]]; !ok {
					seen[m[1]] = struct{}{}
					missing = append(missing, m[1])
				}
			}
		}
		return nil, fmt.Errorf("missing injected variable(s): %s", strings.Join(missing, ", "))
	}

	// Back into the struct
	var out models.FileConfig
	if err := json.Unmarshal([]byte(jsonStr), &out); err != nil {
		return nil, fmt.Errorf("unmarshal after injection: %w", err)
	}
	return &out, nil
}

func PerformVariableInjectionWithMap(config models.FileConfig, vars map[string]string) (*models.FileConfig, error) {
	SetVars(vars)
	return PerformVariableInjection(config)
}

func LoadProfiles(filepath string) (*models.Profiles, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	var profiles models.Profiles
	err = decoder.Decode(&profiles)
	if err != nil {
		return nil, err
	}

	log.Debug("Profile loaded ", "profile", profiles)
	return &profiles, nil
}

func ToJSONString(data any) (string, error) {
	jsonBytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", err
	}
	return string(jsonBytes), nil
}

// pkg/json/json.go (or wherever your helpers live)
type CompiledJSON struct {
	Tpl    *template.Template
	Fields []models.Field // <-- MUST be set
	Path   string
}

func CompileJSONField(spec models.Field, templatePath string) (*CompiledJSON, error) {
	// 1) Coalesce template path (prefer arg, then json.template, then top-level template)
	tpath := strings.TrimSpace(templatePath)
	if tpath == "" && strings.TrimSpace(spec.Template) != "" {
		tpath = spec.Template
	}
	if tpath == "" && strings.TrimSpace(spec.Template) != "" {
		tpath = spec.Template
	}
	if tpath == "" {
		return nil, fmt.Errorf("json: missing template path for field '%s'", spec.Name)
	}
	abs, err := filepath.Abs(tpath)
	if err != nil {
		return nil, fmt.Errorf("resolve template path: %w", err)
	}

	// 2) Read + parse
	b, err := os.ReadFile(abs)
	if err != nil {
		return nil, fmt.Errorf("read template: %w", err)
	}
	tpl, err := template.New(spec.Name).Option("missingkey=error").Parse(string(b))
	if err != nil {
		return nil, fmt.Errorf("parse json template: %w", err)
	}

	// 3) Coalesce nested fields (prefer json.fields, else top-level fields)
	flds := spec.Fields
	if len(flds) == 0 && len(spec.Fields) > 0 {
		flds = spec.Fields
	}
	if len(flds) == 0 {
		return nil, fmt.Errorf("json: no nested fields for '%s' (expected top-level 'fields' or 'json.fields')", spec.Name)
	}

	return &CompiledJSON{
		Tpl:    tpl,
		Fields: flds,
		Path:   abs,
	}, nil
}

func GenerateNestedValues(rowIndex int, seedIndex int, rng *rand.Rand, jsonFields []models.Field, cache []map[string]any) (map[string]string, error) {
	out := make(map[string]string, len(jsonFields))

	for _, f := range jsonFields {
		var value any
		key := f.Name
		if f.Alias != "" {
			key = f.Alias
		}

		if f.Seed {
			value = cache[seedIndex][key]
			log.Debug("seeding json attribute", "field", f.Name, "value", fmt.Sprint(value))
		} else {

			factory, found := fakers.GetFakerByName(f.Type)
			if !found {
				return nil, fmt.Errorf("faker not found for type: %s", f.Type)
			}
			faker, err := factory(f, rng)
			if err != nil {
				log.Error("Error creating faker", "field_name", f.Name, "type", f.Type)
				return nil, fmt.Errorf("%w", err)
			}
			value, err = faker.Generate()
			if err != nil {
				return nil, fmt.Errorf("error generating value for field %s: %w", f.Name, err)
			}

			if err != nil {
				return nil, fmt.Errorf("json[%s]: %w", f.Name, err)
			}
		}

		valueStr := fmt.Sprint(value)
		// log.Debug("value generated", "key", key, "value", valueStr)
		out[f.Name] = valueStr
	}
	return out, nil
}

var bufPool = sync.Pool{New: func() any { return new(bytes.Buffer) }}

func RenderJSONCell(cj *CompiledJSON, kv map[string]string) (string, error) {

	buf := bufPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer bufPool.Put(buf)

	if err := cj.Tpl.Execute(buf, kv); err != nil {
		return "", fmt.Errorf("execute template: %w", err)
	}

	// Validate & compact to single line
	// (This also catches missing commas/braces in templates.)
	var anyJSON any
	if err := json.Unmarshal(buf.Bytes(), &anyJSON); err != nil {
		return "", fmt.Errorf("invalid json from template: %w\nrendered: %s", err, buf.String())
	}
	// Re-marshal compact
	compact, err := json.Marshal(anyJSON)
	if err != nil {
		return "", fmt.Errorf("compact json: %w", err)
	}
	return string(compact), nil
}
