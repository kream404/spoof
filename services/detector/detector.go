package detector

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	json_writer "github.com/kream404/spoof/services/json"

	"github.com/kream404/spoof/models"
	log "github.com/kream404/spoof/services/logger"
)

//
// ───────────────────────── BASIC DETECTORS ────────────────────────────
//

func isUUID(s string) bool {
	uuidRegex := regexp.MustCompile(`^[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{12}$`)
	return uuidRegex.MatchString(strings.TrimSpace(s))
}

func isNumber(s string) (valid bool, decimals int, length int) {
	s = strings.TrimSpace(s)
	if _, err := strconv.ParseFloat(s, 64); err != nil {
		return false, 0, 0
	}

	// strip sign for length/decimals calculation
	s = strings.TrimPrefix(s, "-")
	s = strings.TrimPrefix(s, "+")

	if strings.Contains(s, ".") {
		parts := strings.SplitN(s, ".", 2)
		decimals = len(parts[1])
	} else {
		length = len(s)
	}

	return true, decimals, length
}

func isTimestamp(s string) (bool, string) {
	s = strings.TrimSpace(s)

	formats := []string{
		// date & time (space)
		"2006-01-02 15:04:05",
		"2006-01-02 15:04:05.000",
		"2006-01-02 15:04:05.000000",
		"2006-01-02 15:04:05.000000000",
		"2006-01-02 15:04:05.000-07",
		"2006-01-02 15:04:05.000-07:00",

		// RFC3339 with/without fractional, with zone
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02T15:04:05.000Z07:00",
		"2006-01-02T15:04:05.000000Z07:00",
		"2006-01-02T15:04:05.000000000Z07:00",

		// RFC3339-like without zone
		"2006-01-02T15:04:05",
		"2006-01-02T15:04:05.000",
		"2006-01-02T15:04:05.000000",
		"2006-01-02 15:04:05.99999-07",
		"2006-01-02T15:04:05.000000000",

		// date-only / time-only
		"2006-01-02",
		"02/01/2006",
		"02-01-06",
		"02-01-06 15:04:05",
		"15:04:05",
	}

	for _, layout := range formats {
		if _, err := time.Parse(layout, s); err == nil {
			return true, layout
		}
	}
	return false, ""
}

func isNullColumn(col []string) bool {
	nonEmpty := 0
	for _, val := range col {
		if strings.TrimSpace(val) != "" {
			nonEmpty++
		}
	}
	if len(col) == 0 {
		return true
	}
	return nonEmpty == 0 || float64(nonEmpty)/float64(len(col)) < 0.05
}

func isEmail(s string) bool {
	s = strings.TrimSpace(s)
	return strings.Contains(s, "@") && strings.Contains(s, ".")
}

// isRange decides if a column is "categorical with a small set of distinct values".
// NOTE: rowCount is currently unused (kept in signature for compatibility).
func isRange(col []string, _ int) (bool, []string) {
	const maxReturn = 500 // TODO: make configurable in extract

	normalize := func(s string) string {
		s = strings.TrimSpace(s)
		if s == "" {
			return ""
		}
		return strings.Join(strings.Fields(s), " ")
	}

	freq := make(map[string]int, 1024)
	for _, raw := range col {
		v := normalize(raw)
		if v == "" {
			continue
		}
		freq[v]++
	}

	if len(freq) == 0 {
		return false, nil
	}

	// rank by frequency (desc), then lex (asc) for deterministic output
	type kv struct {
		k string
		v int
	}
	items := make([]kv, 0, len(freq))
	for k, v := range freq {
		items = append(items, kv{k: k, v: v})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].v != items[j].v {
			return items[i].v > items[j].v
		}
		return items[i].k < items[j].k
	})

	limit := maxReturn
	if len(items) < limit {
		limit = len(items)
	}
	out := make([]string, 0, limit)
	for i := 0; i < limit; i++ {
		out = append(out, items[i].k)
	}
	return true, out
}

func isAlphanumeric(col []string) (ok bool, format string, length int) {
	alphaNumRe := regexp.MustCompile(`^[A-Za-z0-9]+$`)

	type stat struct {
		upper bool
		lower bool
		mixed bool
	}

	lengthCount := make(map[int]int)
	var s stat
	sawAny := false

	for _, v := range col {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		if !alphaNumRe.MatchString(v) {
			return false, "", 0
		}
		sawAny = true

		hasLetter := false
		allUpper := true
		allLower := true
		for _, r := range v {
			switch {
			case r >= 'A' && r <= 'Z':
				hasLetter = true
				allLower = false
			case r >= 'a' && r <= 'z':
				hasLetter = true
				allUpper = false
			}
		}
		if hasLetter {
			if allUpper {
				s.upper = true
			} else if allLower {
				s.lower = true
			} else {
				s.mixed = true
			}
		}

		lengthCount[len(v)]++
	}

	if !sawAny {
		return false, "", 0
	}

	// decide format
	switch {
	case s.mixed || (s.upper && s.lower):
		format = "mixed"
	case s.upper:
		format = "upper"
	case s.lower:
		format = "lower"
	default:
		// column had digits only; still valid alphanumeric, default to mixed
		format = "mixed"
	}

	maxC := -1
	for L, c := range lengthCount {
		if c > maxC {
			maxC = c
			length = L
		}
	}

	return true, format, length
}

func isIterator(col []string) bool {
	if len(col) == 0 {
		return false
	}
	for i := 1; i < len(col); i++ {
		prev, errPrev := strconv.Atoi(strings.TrimSpace(col[i-1]))
		curr, errCurr := strconv.Atoi(strings.TrimSpace(col[i]))
		if errPrev != nil || errCurr != nil || curr-prev != 1 {
			return false
		}
	}
	return true
}

func isJSON(s string) (bool, any) {
	s = strings.TrimSpace(s)
	if s == "" {
		return false, nil
	}

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

//
// ───────────────────────── JSON INFERENCE ─────────────────────────────
//

type infCtx struct {
	used map[string]int
}

func inferJSONTemplateAndFields(v any) (string, []models.Field, error) {
	ctx := &infCtx{used: make(map[string]int)}
	node, fields := jsonInference(ctx, v, nil)

	tplBytes, err := json_writer.MarshalTemplate(node)
	if err != nil {
		return "", nil, err
	}
	return string(tplBytes), fields, nil
}

func jsonInference(ctx *infCtx, v any, path []string) (any, []models.Field) {
	switch t := v.(type) {
	case map[string]any:
		return inferJSONObject(ctx, t, path)
	case []any:
		return inferJSONArray(ctx, t, path)
	default:
		return inferJSONLeaf(ctx, t, path)
	}
}

func inferJSONObject(ctx *infCtx, obj map[string]any, path []string) (any, []models.Field) {
	out := make(map[string]any, len(obj))
	var fields []models.Field

	for k, val := range obj {
		n, f := jsonInference(ctx, val, append(path, k))
		out[k] = n
		fields = append(fields, f...)
	}

	return out, fields
}

func inferJSONArray(ctx *infCtx, arr []any, path []string) (any, []models.Field) {
	if len(arr) == 0 {
		// empty array: keep as-is, no fields
		return []any{}, nil
	}

	out := make([]any, len(arr))
	var fields []models.Field

	for i, val := range arr {
		n, f := jsonInference(ctx, val, append(path, strconv.Itoa(i)))
		out[i] = n
		fields = append(fields, f...)
	}

	return out, fields
}

func inferJSONLeaf(ctx *infCtx, v any, path []string) (any, []models.Field) {
	key := makeKey(ctx, path)
	mf := inferFieldFromValue(key, v)
	phType := ""
	if mf.Type == "number" {
		phType = "number"
	} else {
		phType = "string"
	}

	ph := models.Placeholder{
		Key:  key,
		Type: phType,
	}

	return ph, []models.Field{mf}
}

func makeKey(ctx *infCtx, path []string) string {
	// join path with underscores and make it identifier-safe
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

//
// ───────────────────────── FIELD INFERENCE ────────────────────────────
//

func InferField(name string, col []string) (models.Field, error) {
	// 1) All / almost-all null
	if isNullColumn(col) {
		return models.Field{Name: name, Value: ""}, nil
	}

	// 2) Pick a non-empty sample value
	var sample string
	for _, v := range col {
		v = strings.TrimSpace(v)
		if v != "" {
			sample = v
			break
		}
	}

	// 3) JSON
	if ok, jsonVal := isJSON(sample); ok {
		tplName := fmt.Sprintf("%s_template.json", json_writer.SanitizeFileStem(name))
		tplText, fields, err := inferJSONTemplateAndFields(jsonVal)
		if err != nil {
			return models.Field{Name: name, Type: "json"}, fmt.Errorf("infer json: %w", err)
		}

		written, err := json_writer.WriteTemplate(tplName, tplText, true)
		if err != nil {
			return models.Field{Name: name, Type: "json"}, err
		}
		log.Debug("wrote json template", "path", written)

		return models.Field{
			Name:     name,
			Type:     "json",
			Template: written,
			Fields:   fields,
		}, nil
	}

	// 4) UUID
	if isUUID(sample) {
		return models.Field{Name: name, Type: "uuid"}, nil
	}

	// 5) Timestamp
	if ok, layout := isTimestamp(sample); ok {
		return models.Field{
			Name:     name,
			Type:     "timestamp",
			Format:   layout,
			Function: "sin:period=0.0001,dir=past,interval=1d,amplitude=3,jitter=0.005,jitter_type=scale",
		}, nil
	}

	// 6) Email
	if isEmail(sample) {
		return models.Field{Name: name, Type: "email"}, nil
	}

	// 7) Number
	if ok, decimals, length := isNumber(sample); ok {
		if length == 2 {
			return models.Field{Name: name, Type: "number", Length: length, Min: 1, Max: 99}, nil
		}
		if length > 0 {
			return models.Field{Name: name, Type: "number", Length: length}, nil
		}
		if decimals == 2 {
			return models.Field{
				Name:   name,
				Type:   "number",
				Format: fmt.Sprint(decimals),
				Length: length,
				Min:    1,
				Max:    99,
				Function: "sin:period=0.01,amplitude=1.5,center=50," +
					"jitter=0.005,jitter_type=scale,jitter_amp=3",
			}, nil
		}
		return models.Field{Name: name, Type: "number"}, nil
	}

	// 8) Iterator
	if isIterator(col) {
		return models.Field{Name: name, Type: "iterator"}, nil
	}

	// 9) Alphanumeric
	if ok, fmtCase, L := isAlphanumeric(col); ok {
		return models.Field{
			Name:   name,
			Type:   "alphanumeric",
			Format: fmtCase,
			Length: L,
		}, nil
	}

	// 10) Small range (categorical)
	if ok, set := isRange(col, len(col)); ok {
		return models.Field{
			Name:   name,
			Type:   "range",
			Values: strings.Join(set, ", "),
		}, nil
	}

	// 11) Fallback
	return models.Field{Name: name, Type: "unknown"}, nil
}

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
		// fallback to generic alphanumeric
		return models.Field{Name: name, Type: "alphanumeric", Length: 16}
	}
}
