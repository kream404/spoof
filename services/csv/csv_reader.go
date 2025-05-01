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

func ReadCSV(filepath string) ([][]string, *os.File, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var firstLine string
	if scanner.Scan() {
		firstLine = scanner.Text()
	} else if err := scanner.Err(); err != nil {
		return nil, nil, err
	}

	delimiter := DetectDelimiter(firstLine)

	_, err = file.Seek(0, 0)
	if err != nil {
		return nil, nil, err
	}
	reader := csv.NewReader(file)
	reader.Comma = delimiter

	records, err := reader.ReadAll()
	if err != nil {
		return nil, nil, err
	}
	return records, file, nil
}

func MapFields(records [][]string) ([]models.Field, error) {

	var fields []models.Field

	if len(records) == 0 {
		return fields, nil
	}

	headers := records[0]
	data := records[1]
	for index, header := range headers {
		field_type, format, err := DetectType(data[index])
		if err != nil {
			println(err)
		}
		fields = append(fields, models.Field{Name: header, Format: format, Type: field_type})
	}

	return fields, nil
}

func DetectDelimiter(line string) rune {
	if strings.Contains(line, "|") {
		return '|'
	}
	return ','
}

func DetectType(value string) (string, string, error) {
	v := strings.TrimSpace(value)

	if isUUID(v) {
		return "uuid", "", nil
	}
	if isInteger(v) {
		return "number", "", nil
	}
	if isFloat(v) {
		return "number", "", nil
	}
	if ok, layout := isTimestamp(v); ok {
		println("layout: ", layout)
		return "timestamp", layout, nil
	}
	if isEmail(v) {
		return "email", "", nil
	}
	return "unknown", "", nil
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
