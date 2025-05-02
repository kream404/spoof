package csv

import (
	"bufio"
	"encoding/csv"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/kream404/spoof/models"
)

func ReadCSV(filepath string) ([][]string, *os.File, rune, error) {
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

	_, err = file.Seek(0, 0)
	if err != nil {
		return nil, nil, 0, err
	}
	reader := csv.NewReader(file)
	reader.Comma = delimiter

	records, err := reader.ReadAll()
	if err != nil {
		return nil, nil, 0, err
	}
	return records, file, delimiter, nil
}

func MapFields(records [][]string) ([]models.Field, error) {

	var fields []models.Field

	if len(records) == 0 {
		return fields, nil
	}

	headers := records[0]
	for index, header := range headers {
		var col []string
		for rowIndex := 1; rowIndex < len(records); rowIndex++ {
			col = append(col, records[rowIndex][index])
		}

		field_type, format, values, err := DetectType(col)
		if err != nil {
			println(err)
		}
		fields = append(fields, models.Field{Name: header, Format: format, Type: field_type, Values: strings.Join(values, ", ")})
	}
	return fields, nil
}

func DetectDelimiter(line string) rune {
	if strings.Contains(line, "|") {
		return '|'
	}
	return ','
}

func DetectType(col []string) (string, string, []string, error) {
	v := strings.TrimSpace(col[0])
	if isUUID(v) {
		return "uuid", "", nil, nil
	}

	if ok, layout := isTimestamp(v); ok {
		return "timestamp", layout, nil, nil
	}
	if isEmail(v) {
		return "email", "", nil, nil
	}
	if isIterator(col) { //check for iterator first - need to do something similar for range, return the array of values too
		return "iterator", "", nil, nil
	}
	if ok, set := isRange(col); ok {
		return "range", "", set, nil
	}
	if isFloat(v) {
		return "number", "", nil, nil
	}
	return "unknown", "", nil, nil
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

func isFloat(s string) bool {
	_, err := strconv.ParseFloat(s, 64)
	return err == nil
}

func isTimestamp(s string) (bool, string) {
	formats := []string{
		"2006-01-02 15:04:05",
		"02-01-06 15:04:05",
		"2006-01-02",
		"02/01/2006",
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

// check if header contains id ??
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

func isRange(col []string) (bool, []string) {
	var unique []string
	set := make(map[string]struct{})
	for _, v := range col {
		v = strings.TrimSpace(v)

		if strings.Contains(v, ".") {
			return false, []string{}
		}

		if isInteger(v) || len(v) > 0 {
			set[v] = struct{}{}
		} else {
			return false, []string{}
		}

		unique = make([]string, 0, len(set))
		for v := range set {
			unique = append(unique, v)
		}
	}
	return true, unique
}
