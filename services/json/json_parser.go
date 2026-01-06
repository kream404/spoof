package json

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/kream404/spoof/fakers"
	"github.com/kream404/spoof/models"
	log "github.com/kream404/spoof/services/logger"
	"github.com/shopspring/decimal"
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

	for k, v := range vars {
		re := regexp.MustCompile(`\{\{\s*` + regexp.QuoteMeta(k) + `\s*\}\}`)
		jsonStr = re.ReplaceAllString(jsonStr, v)
	}

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

type CompiledJSON struct {
	Raw    string // raw JSON template text
	Fields []models.Field
	Path   string
}

func CompileJSONField(spec models.Field, templatePath string) (*CompiledJSON, error) {
	tpath := strings.TrimSpace(templatePath)
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

	b, err := os.ReadFile(abs)
	if err != nil {
		return nil, fmt.Errorf("read template: %w", err)
	}

	flds := spec.Fields
	if len(flds) == 0 {
		return nil, fmt.Errorf("json: no nested fields for '%s' (expected top-level 'fields')", spec.Name)
	}

	return &CompiledJSON{
		Raw:    string(b),
		Fields: flds,
		Path:   abs,
	}, nil
}

func GenerateNestedValues(rowIndex, seedIndex int, rng *rand.Rand, fields []models.Field, cache []map[string]any, fieldSources map[string][]map[string]any, parentGenerated map[string]string) (map[string]string, error) {

	values := make(map[string]string)
	generated := make(map[string]string)

	for _, field := range fields {
		var value any
		var key string

		if field.Alias != "" {
			key = field.Alias
		} else {
			key = field.Name
		}

		injected := false

		if field.Source != "" && strings.Contains(field.Source, ".csv") {
			if rows, ok := fieldSources[field.Source]; ok && len(rows) > 0 {
				idx := seedIndex % len(rows)
				if row := rows[idx]; row != nil {
					if val, ok := row[key]; ok && val != nil {
						value = val
						injected = true
					}
				}
			}
		}

		if !injected && field.Seed && len(cache) > 0 {
			idx := seedIndex % len(cache)
			if row := cache[idx]; row != nil {
				if val, ok := row[key]; ok && val != nil {
					value = val
					injected = true
				}
			}
		}

		if !injected {
			switch {
			case field.Type == "reflection":
				if field.Target == "" {
					return nil, fmt.Errorf("You must provide a 'target' to use reflection")
				}

				// Prefer nested fields first, then fall back to parent scope
				targetValue, ok := generated[field.Target]
				if !ok && parentGenerated != nil {
					targetValue, ok = parentGenerated[field.Target]
				}
				if !ok {
					return nil, fmt.Errorf("reflection target '%s' not found in previous fields", field.Target)
				}

				out := targetValue
				if field.Modifier != nil {
					modified, err := modifier(targetValue, *field.Modifier)
					if err != nil {
						return nil, err
					}
					out = modified
				}

				values[field.Name] = out
				generated[field.Name] = out
				continue
			case field.Type == "":
				value = field.Value
			case field.Type == "iterator":
				value = rowIndex
			case field.Type == "foreach":
				vals := field.Values
				raw := strings.TrimSpace(vals)
				parts := strings.Split(raw, ",")
				cleaned := make([]string, 0, len(parts))
				for _, p := range parts {
					s := strings.TrimSpace(p)
					if s != "" {
						cleaned = append(cleaned, s)
					}
				}

				if len(cleaned) == 0 {
					log.Warn("foreach field has only empty/whitespace values; emitting empty string", "field", field.Name)
					value = ""
					break
				}

				idx := rowIndex % len(cleaned)
				value = cleaned[idx]
			default:
				factory, found := fakers.GetFakerByName(field.Type)
				if !found {
					return nil, fmt.Errorf("faker not found for nested field type: %s", field.Type)
				}
				faker, err := factory(field, rng)
				if err != nil {
					return nil, fmt.Errorf("create faker for nested field %s: %w", field.Name, err)
				}
				v, err := faker.Generate()
				if err != nil {
					return nil, fmt.Errorf("generate value for nested field %s: %w", field.Name, err)
				}
				value = v
			}
		}

		var s string
		if value == nil {
			s = ""
		} else {
			s = fmt.Sprint(value)
		}

		generated[field.Name] = s
		values[field.Name] = s
	}

	return values, nil
}

func modifier(raw string, modifier float64) (string, error) {
	decimalValue, err := decimal.NewFromString(raw)
	if err != nil {
		return "", fmt.Errorf("invalid number: %v", err)
	}
	modifiedValue := decimalValue.Mul(decimal.NewFromFloat(modifier))

	decimals := 0
	if dot := strings.Index(raw, "."); dot != -1 {
		decimals = len(raw) - dot - 1
	}

	return modifiedValue.StringFixed(int32(decimals)), nil
}
