package csv

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"os"
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
	v := strings.TrimSpace(col[0])
	if isUUID(v) {
		return models.Field{Name: header, Type: "uuid"}, nil
	}

	if ok, layout := isTimestamp(v); ok {
		return models.Field{Name: header, Type: "timestamp", Format: layout}, nil
	}
	if isEmail(v) {
		return models.Field{Name: header, Type: "email"}, nil
	}
	if isIterator(col) {
		return models.Field{Name: header, Type: "iterator"}, nil
	}

	if ok, set := isRange(col, len(col)); ok {
		return models.Field{Name: header, Type: "range", Values: strings.Join(set, ", ")}, nil
	}
	if ok, decimals, length := isNumber(v); ok {
		return models.Field{Name: header, Type: "number", Format: fmt.Sprint(decimals), Length: length, Min: 0, Max: 5000}, nil
	}
	return models.Field{Name: header, Type: "unknown"}, nil

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
	formats := []string{
		"2006-01-02 15:04:05",
		"02-01-06 15:04:05",
		"2006-01-02",
		"02/01/2006",
		"02-01-06",
		"15:04:05",
	}
	for _, layout := range formats {
		if _, err := time.Parse(layout, s); err == nil {
			return true, layout
		}
	}
	return false, ""
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

func isRange(col []string, rowCount int) (bool, []string) {
	maxDistinct := int(float64(rowCount) * 0.01)
	if maxDistinct < 10 {
		maxDistinct = 10
	}

	set := make(map[string]int)
	for _, row := range col {
		set[row]++
		if len(set) > maxDistinct {
			return false, nil
		}
	}

	unique := make([]string, 0, len(set))
	for v := range set {
		unique = append(unique, v)
	}
	sort.Strings(unique)
	return true, unique
}
