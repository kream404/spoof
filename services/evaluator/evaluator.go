package evaluator

import (
	"fmt"
	"math/rand"
	"strings"

	jsonstd "encoding/json" // if you already alias this elsewhere, keep consistent

	"github.com/kream404/spoof/fakers"
	"github.com/kream404/spoof/models"
	"github.com/kream404/spoof/services/json" // keep your existing json pkg alias as needed
	"github.com/kream404/spoof/services/logger"
	"github.com/shopspring/decimal"
)

type evalCtx struct {
	// row state
	rowIndex  int
	seedIndex int
	rng       *rand.Rand

	// data
	cache        []map[string]any
	fieldSources map[string][]map[string]any

	// generated scopes (keyed by output key: alias if present else name)
	generated       map[string]string
	parentGenerated map[string]string

	// selector (optional)
	seedSelector *models.SeedSelector

	// injection gating
	shouldInject func(models.Field, *rand.Rand) bool
}

func lookupKey(field models.Field) string {
	if field.Alias != "" {
		return field.Alias
	}
	return field.Name
}

// Output/render key must be name (this is what templates refer to)
func outKey(field models.Field) string {
	return field.Name
}

func cleanCSVList(s string) []string {
	raw := strings.TrimSpace(s)
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		v := strings.TrimSpace(p)
		if v != "" {
			out = append(out, v)
		}
	}
	return out
}

func getSeededRow(cache []map[string]any, seedIndex int) map[string]any {
	if len(cache) == 0 {
		return nil
	}
	idx := seedIndex % len(cache)
	if idx < 0 {
		idx = -idx
	}
	return cache[idx]
}

func (c *evalCtx) tryInjectFromSource(field models.Field, key string) (any, bool) {
	if field.Source == "" || !strings.Contains(field.Source, ".csv") {
		return nil, false
	}
	if c.shouldInject != nil && !c.shouldInject(field, c.rng) {
		return nil, false
	}

	rows, ok := c.fieldSources[field.Source]
	if !ok || len(rows) == 0 {
		return nil, false
	}

	idx := c.seedIndex % len(rows)
	if idx < 0 {
		idx = -idx
	}

	row := rows[idx]
	if row == nil {
		return nil, false
	}

	if val, ok := row[key]; ok && val != nil {
		return val, true
	}

	return nil, false
}

func (c *evalCtx) trySeed(field models.Field, key string) (any, bool, error) {
	if !field.Seed || len(c.cache) == 0 {
		return nil, false, nil
	}

	// Selector mode: pick lookup key for this row, find cache row by selector column, read output column key.
	if c.seedSelector != nil {
		sel := c.seedSelector
		if sel.Column == "" {
			return nil, false, fmt.Errorf("seedSelector.column is required for field %s", field.Name)
		}
		if len(sel.Keys) == 0 {
			return nil, false, fmt.Errorf("seedSelector.keys is required for field %s", field.Name)
		}

		ki := (c.rowIndex - 1) % len(sel.Keys)
		if ki < 0 {
			ki = -ki
		}
		lookupKey := sel.Keys[ki]
		logger.Debug("lookupKey", "lookupKey", lookupKey)

		matches := 0
		var picked any

		for _, row := range c.cache {
			if row == nil {
				continue
			}
			v := row[sel.Column]
			logger.Debug("seed selection", "row", row, "v", v, "sel.Column", sel.Column)
			if v == nil {
				continue
			}
			if fmt.Sprint(v) == lookupKey {
				out := row[key]
				if out == nil || fmt.Sprint(out) == "" {
					return nil, false, fmt.Errorf(
						"seedSelector matched row where %s=%q but output column %q was missing/empty",
						sel.Column, lookupKey, key,
					)
				}
				picked = out
				logger.Debug("value picked", "picked", picked)
				matches++
			}
		}

		if matches == 0 {
			return nil, false, fmt.Errorf(
				"seedSelector lookup failed for field=%s: no cache row where %s == %q",
				field.Name, sel.Column, lookupKey,
			)
		}
		if matches > 1 {
			return nil, false, fmt.Errorf(
				"seedSelector lookup ambiguous for field=%s: %d cache rows where %s == %q",
				field.Name, matches, sel.Column, lookupKey,
			)
		}

		return picked, true, nil
	}

	row := getSeededRow(c.cache, c.seedIndex)
	if row == nil {
		return nil, false, nil
	}
	if val, ok := row[key]; ok && val != nil {
		return val, true, nil
	}
	return nil, false, nil
}

func (c *evalCtx) resolveReflection(field models.Field) (string, error) {
	if field.Target == "" {
		return "", fmt.Errorf("you must provide a 'target' to use reflection")
	}

	if v, ok := c.generated[field.Target]; ok {
		return v, nil
	}
	if c.parentGenerated != nil {
		if v, ok := c.parentGenerated[field.Target]; ok {
			return v, nil
		}
	}

	for k, v := range c.generated {
		if k == field.Target {
			return v, nil
		}
		_ = k
	}
	if c.parentGenerated != nil {
		for k, v := range c.parentGenerated {
			if k == field.Target {
				return v, nil
			}
			_ = k
		}
	}

	return "", fmt.Errorf("reflection target '%s' not found in previous fields", field.Target)
}

func (c *evalCtx) evaluateField(field models.Field) (string, error) {
	lk := lookupKey(field) // for cache/source lookup
	okey := outKey(field)  // for generated/output maps

	// 1) injection
	if val, ok := c.tryInjectFromSource(field, lk); ok {
		s := fmt.Sprint(val)
		out, err := applyModifier(s, field)
		if err != nil {
			return "", fmt.Errorf("modifier failed for field %s: %w", field.Name, err)
		}
		c.generated[okey] = out
		return out, nil
	}

	// 2) seed
	if val, ok, err := c.trySeed(field, lk); err != nil {
		return "", err
	} else if ok {
		s := fmt.Sprint(val)
		out, err := applyModifier(s, field)
		if err != nil {
			return "", fmt.Errorf("modifier failed for field %s: %w", field.Name, err)
		}
		c.generated[okey] = out
		return out, nil
	}

	// 3) compute fallback
	var value any

	switch {
	case field.Type == "reflection":
		targetValue, err := c.resolveReflection(field)
		if err != nil {
			return "", err
		}
		value = targetValue
		if field.Modifier != "" {
			modified, err := modifier(targetValue, field.Modifier)
			if err != nil {
				return "", err
			}
			value = modified
		}

	case field.Type == "iterator":
		start := 1
		if field.Start != nil {
			start = *field.Start
		}
		value = start + c.rowIndex

	case field.Type == "":
		value = field.Value

	case field.Type == "foreach":
		cleaned := cleanCSVList(field.Values)
		if len(cleaned) == 0 {
			value = ""
			break
		}
		idx := (c.rowIndex - 1) % len(cleaned)
		if idx < 0 {
			idx = -idx
		}
		value = cleaned[idx]

	case field.Type == "json":
		cj, err := json.CompileJSONField(field, field.Template)
		if err != nil {
			return "", err
		}

		raw := strings.TrimSpace(cj.Raw)
		rootIsArray := strings.HasPrefix(raw, "[")

		if !rootIsArray {
			kv, err := GenerateNestedValuesUnified(
				c.rowIndex,
				c.seedIndex,
				c.rng,
				cj.Fields,
				c.cache,
				c.fieldSources,
				c.generated, // parent for nested
				c.shouldInject,
				c.seedSelector,
			)
			if err != nil {
				return "", err
			}

			s, err := json.RenderJSONCell(cj.Raw, kv)
			if err != nil {
				return "", err
			}
			value = s
			break
		}

		repeat := field.Repeat
		if repeat <= 0 {
			repeat = 1
		}

		items := make([]any, 0, repeat)

		for j := 0; j < repeat; j++ {
			iterSeed := c.seedIndex + j

			kv, err := GenerateNestedValuesUnified(
				c.rowIndex,
				iterSeed,
				c.rng,
				cj.Fields,
				c.cache,
				c.fieldSources,
				c.generated, // parent for nested
				c.shouldInject,
				c.seedSelector,
			)
			if err != nil {
				return "", err
			}

			rendered, err := json.RenderJSONCell(cj.Raw, kv)
			if err != nil {
				return "", err
			}

			var arr []any
			if err := jsonstd.Unmarshal([]byte(rendered), &arr); err != nil {
				return "", fmt.Errorf(
					"invalid rendered JSON for %s (expected array root): %w\nrendered: %s",
					field.Name, err, rendered,
				)
			}

			if len(arr) > 0 {
				items = append(items, arr[0])
			}
		}

		out, err := jsonstd.Marshal(items)
		if err != nil {
			return "", err
		}
		value = string(out)

	default:
		factory, found := fakers.GetFakerByName(field.Type)
		if !found {
			return "", fmt.Errorf("faker not found for type: %s", field.Type)
		}
		faker, err := factory(field, c.rng)
		if err != nil {
			return "", fmt.Errorf("error creating faker for field %s: %w", field.Name, err)
		}
		v, err := faker.Generate()
		if err != nil {
			return "", fmt.Errorf("error generating value for field %s: %w", field.Name, err)
		}
		value = v
	}

	s := ""
	if value != nil {
		s = fmt.Sprint(value)
	}

	out, err := applyModifier(s, field)
	if err != nil {
		return "", fmt.Errorf("modifier failed for field %s: %w", field.Name, err)
	}

	c.generated[okey] = out
	return out, nil
}

func modifier(raw string, modifier string) (string, error) {
	decimalValue, err := decimal.NewFromString(raw)
	if err != nil {
		return "", fmt.Errorf("invalid number: %v", err)
	}
	modifierValue, err := decimal.NewFromString(modifier)
	if err != nil {
		return "", fmt.Errorf("invalid number: %v", err)
	}

	modifiedValue := decimalValue.Mul(modifierValue)

	decimals := 0
	if dot := strings.Index(raw, "."); dot != -1 {
		decimals = len(raw) - dot - 1
	}

	return modifiedValue.StringFixed(int32(decimals)), nil
}

func applyModifier(val string, field models.Field) (string, error) {
	if field.Modifier == "" {
		return val, nil
	}
	return modifier(val, field.Modifier)
}

func shouldInjectFromSource(field models.Field, rng *rand.Rand) bool {
	if field.Source == "" {
		return false
	}
	// default to 100% if rate is omitted
	if field.Rate == nil {
		return true
	}
	r := *field.Rate
	if r <= 0 {
		return false
	}
	if r >= 100 {
		return true
	}
	return rng.Intn(100) < r
}

// JSON Evaluation
func GenerateNestedValues(
	rowIndex, seedIndex int,
	rng *rand.Rand,
	fields []models.Field,
	cache []map[string]any,
	fieldSources map[string][]map[string]any,
	parentGenerated map[string]string,
) (map[string]string, error) {

	return GenerateNestedValuesUnified(
		rowIndex,
		seedIndex,
		rng,
		fields,
		cache,
		fieldSources,
		parentGenerated,
		shouldInjectFromSource,
		nil,
	)
}

func GenerateNestedValuesUnified(
	rowIndex, seedIndex int,
	rng *rand.Rand,
	fields []models.Field,
	cache []map[string]any,
	fieldSources map[string][]map[string]any,
	parentGenerated map[string]string,
	shouldInject func(models.Field, *rand.Rand) bool,
	seedSelector *models.SeedSelector,
) (map[string]string, error) {

	values := make(map[string]string, len(fields))
	localGenerated := make(map[string]string, len(fields))

	ctx := evalCtx{
		rowIndex:        rowIndex,
		seedIndex:       seedIndex,
		rng:             rng,
		cache:           cache,
		fieldSources:    fieldSources,
		generated:       localGenerated,
		parentGenerated: parentGenerated,
		shouldInject:    shouldInject,
		seedSelector:    seedSelector,
	}

	for _, field := range fields {
		val, err := ctx.evaluateField(field)
		if err != nil {
			return nil, err
		}

		values[outKey(field)] = val
	}

	return values, nil
}

func GenerateValues(
	file models.Entity,
	cache []map[string]any,
	fieldSources map[string][]map[string]any,
	rowIndex int,
	seedIndex int,
	rng *rand.Rand,
) ([]string, map[string]string, error) {

	record := make([]string, 0, len(file.Fields))
	generatedFields := make(map[string]string, len(file.Fields))

	ctx := evalCtx{
		rowIndex:        rowIndex,
		seedIndex:       seedIndex,
		rng:             rng,
		cache:           cache,
		fieldSources:    fieldSources,
		generated:       generatedFields,
		parentGenerated: nil,
		shouldInject:    shouldInjectFromSource,
	}

	if file.CacheConfig != nil {
		ctx.seedSelector = file.CacheConfig.SeedSelector
	}

	for _, field := range file.Fields {
		val, err := ctx.evaluateField(field)
		if err != nil {
			return nil, nil, err
		}
		if !field.Skip {
			record = append(record, val)
		}
	}

	return record, generatedFields, nil
}
