package csv

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/kream404/spoof/models"
	log "github.com/kream404/spoof/services/logger"
)

// returns records from csv, file, delim, easier to do all this on read
func ReadCSV(path string) ([][]string, string, rune, string, string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, "", 0, "", "", err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var lines []string

	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, "", 0, "", "", err
	}

	if len(lines) < 3 {
		return nil, "", 0, "", "", fmt.Errorf("file does not contain enough lines to have header/data/footer")
	}

	// Header and footer detection
	delimiter := DetectDelimiter(lines[1]) // first data line after header
	delimiterStr := string(delimiter)

	var header, footer string
	startIdx := 0
	endIdx := len(lines)

	if !strings.Contains(lines[0], delimiterStr) {
		header = lines[0]
		startIdx = 1
	}
	if !strings.Contains(lines[len(lines)-1], delimiterStr) {
		footer = lines[len(lines)-1]
		endIdx = len(lines) - 1
	}

	dataLines := lines[startIdx:endIdx]

	reader := csv.NewReader(strings.NewReader(strings.Join(dataLines, "\n")))
	reader.Comma = delimiter

	records, err := reader.ReadAll()
	if err != nil {
		return nil, "", 0, "", "", err
	}

	return records, file.Name(), delimiter, header, footer, nil
}

func ReadCSVAsMap(filepath string) ([]map[string]any, []string, rune, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, nil, 0, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var firstLine string
	if scanner.Scan() {
		firstLine = scanner.Text()
	} else if err := scanner.Err(); err != nil {
		return nil, nil, 0, err
	}

	delimiter := DetectDelimiter(firstLine)

	// rewind and read all with encoding/csv
	if _, err := file.Seek(0, 0); err != nil {
		return nil, nil, 0, err
	}
	r := csv.NewReader(file)
	r.Comma = delimiter

	records, err := r.ReadAll()
	if err != nil {
		return nil, nil, 0, err
	}
	if len(records) == 0 {
		return nil, nil, delimiter, nil
	}

	rawHeaders := records[0]
	headers := makeUniqueHeaders(rawHeaders)

	rows := make([]map[string]any, 0, len(records)-1)
	for i := 1; i < len(records); i++ {
		rec := records[i]
		row := make(map[string]any, len(headers))
		for colIdx, h := range headers {
			var v string
			if colIdx < len(rec) {
				v = rec[colIdx]
			}
			row[h] = v
		}
		rows = append(rows, row)
	}

	log.Debug("read from csv", "headers", fmt.Sprint(headers))
	return rows, headers, delimiter, nil
}

func makeUniqueHeaders(in []string) []string {
	seen := make(map[string]int)
	out := make([]string, len(in))
	for i, h := range in {
		h = strings.TrimSpace(h)
		if h == "" {
			h = fmt.Sprintf("col_%d", i)
		}
		key := h
		if c, ok := seen[key]; ok {
			seen[key] = c + 1
			key = fmt.Sprintf("%s_%d", h, c+1)
		} else {
			seen[key] = 1
		}
		out[i] = key
	}
	return out
}

func MapFields(records [][]string) ([]models.Field, []string, error) {
	log.Debug("Mapping fields")
	var fields []models.Field
	var types []string

	if len(records) == 0 {
		log.Error("No records in csv")
		return fields, nil, nil
	}

	headers := records[0]
	log.Debug("Headers in CSV", "headers", headers)
	for index, header := range headers {
		var col []string
		for rowIndex := 1; rowIndex < len(records); rowIndex++ {
			col = append(col, records[rowIndex][index])
		}

		log.Debug("Detecting type for column", "col", header)
		field, err := DetectType(col, header) //this returns a field with all the config, could rename
		if err != nil {
			log.Error("Failed to detect type ", "error", err.Error())
		}

		types = append(types, field.Type)
		fields = append(fields, field)
	}
	return fields, types, nil
}

func DetectDelimiter(line string) rune {
	if strings.Contains(line, "|") {
		return '|'
	}
	return ','
}

func DetectType(col []string, header string) (models.Field, error) {
	return InferField(header, col)
}

// TODO: this should probably be refactored to live in the fakers. Would know what optional fields can be returned and could return the field
func isUUID(s string) bool {
	uuidRegex := regexp.MustCompile(`^[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{12}$`)
	return uuidRegex.MatchString(s)
}

func isInteger(s string) bool {
	_, err := strconv.Atoi(s)
	return err == nil
}

func isNumber(s string) (valid bool, decimals int, length int) {
	s = strings.TrimSpace(s)
	if _, err := strconv.ParseFloat(s, 64); err != nil {
		return false, 0, 0
	}

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

		// RFC3339-like WITHOUT zone (this is what your JSON uses)
		"2006-01-02T15:04:05",
		"2006-01-02T15:04:05.000",
		"2006-01-02T15:04:05.000000",
		"2006-01-02T15:04:05.000000000",

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
	return nonEmpty == 0 || float64(nonEmpty)/float64(len(col)) < 0.05
}

func isEmail(s string) bool {
	return strings.Contains(s, "@") && strings.Contains(s, ".")
}

func isIterator(col []string) bool {
	for i := 1; i < len(col); i++ {
		prev, errPrev := strconv.Atoi(col[i-1])
		curr, errCurr := strconv.Atoi(col[i])
		if errPrev != nil || errCurr != nil || curr-prev != 1 {
			return false
		}
	}
	return true
}

// isRange decides if a column is "categorical with a small set of distinct values".
// It adapts to file size by allowing up to min(max(ceil(5% of non-empty), 8), 200) distincts.
func isRange(col []string, rowCount int) (bool, []string) {
	const maxReturn = 500 //TODO: make configurable in extract

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
			if r >= 'A' && r <= 'Z' {
				hasLetter = true
				allLower = false
			} else if r >= 'a' && r <= 'z' {
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

type infCtx struct {
	used map[string]int
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

type placeholder struct {
	Key   string
	Quote bool
} // Quote=true -> rendered as "{{ .Key }}", otherwise {{ .Key }}

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
		ph := placeholder{Key: key, Quote: mf.Type == "email" || mf.Type == "uuid" || mf.Type == "timestamp" || mf.Type == "alphanumeric" || mf.Type == ""}

		return ph, []models.Field{mf}
	}
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

func InferField(name string, col []string) (models.Field, error) {
	// 1) All / almost-all null
	if isNullColumn(col) {
		return models.Field{Name: name, Value: ""}, nil
	}

	var sample string
	for _, v := range col {
		v = strings.TrimSpace(v)
		if v != "" {
			sample = v
			break
		}
	}

	if ok, jsonVal := isJSON(sample); ok {
		tplName := fmt.Sprintf("%s_template.json", sanitizeFileStem(name))

		tplText, fields, err := inferJSONTemplateAndFields(jsonVal, tplName)
		if err != nil {
			return models.Field{Name: name, Type: "json"}, fmt.Errorf("infer json: %w", err)
		}

		written, err := WriteTemplate(tplName, tplText, true)
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
	if isUUID(sample) {
		return models.Field{Name: name, Type: "uuid"}, nil
	}
	if ok, layout := isTimestamp(sample); ok {
		return models.Field{
			Name:     name,
			Type:     "timestamp",
			Format:   layout,
			Function: "sin:period=0.0001,dir=past,interval=1d,amplitude=3,jitter=0.005,jitter_type=scale",
		}, nil
	}
	if isEmail(sample) {
		return models.Field{Name: name, Type: "email"}, nil
	}
	if isIterator(col) {
		return models.Field{Name: name, Type: "iterator"}, nil
	}
	if ok, decimals, length := isNumber(sample); ok {
		if length > 0 {
			return models.Field{Name: name, Type: "number", Length: length}, nil
		}
		if decimals == 2 {
			return models.Field{
				Name:   name,
				Type:   "number",
				Format: fmt.Sprint(decimals),
				Length: length,
				Min:    0,
				Max:    500,
				Function: "sin:period=0.01,amplitude=1.5,center=50," +
					"jitter=0.005,jitter_type=scale,jitter_amp=3",
			}, nil
		}
		return models.Field{Name: name, Type: "number"}, nil
	}

	if ok, fmtCase, L := isAlphanumeric(col); ok {
		return models.Field{
			Name:   name,
			Type:   "alphanumeric",
			Format: fmtCase,
			Length: L,
		}, nil
	}

	if ok, set := isRange(col, len(col)); ok {
		return models.Field{
			Name:   name,
			Type:   "range",
			Values: strings.Join(set, ", "),
		}, nil
	}

	// 6) Fallback
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
		// bool stays a tiny range
		return models.Field{Name: name, Type: "range", Values: "true, false"}

	case nil:
		return models.Field{Name: name, Value: ""}

	default:
		// fallback to alphanumeric-ish
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
			// key
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
			// string-like
			w.WriteString(`"{{ .`)
			w.WriteString(t.Key)
			w.WriteString(` }}"`)
		} else {
			// number/bool
			w.WriteString(`{{ .`)
			w.WriteString(t.Key)
			w.WriteString(` }}`)
		}
		return nil
	case string:
		b, _ := json.Marshal(t)
		w.Write(b)
		return nil
	case float64, bool, nil:
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
