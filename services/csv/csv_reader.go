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

func ReadCSV(filepath string) ([][]string, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var firstLine string
	if scanner.Scan() {
		firstLine = scanner.Text()
	} else if err := scanner.Err(); err != nil {
		return nil, err
	}

	delimiter := DetectDelimiter(firstLine)

	_, err = file.Seek(0, 0)
	if err != nil {
		return nil, err
	}
	reader := csv.NewReader(file)
	reader.Comma = delimiter

	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}
	return records, nil
}

func MapFields(records [][]string) ([]models.Field, error) {

	var fields []models.Field

	if len(records) == 0 {
		return fields, nil
	}

	headers := records[0]
	data := records[1]
	for index, header := range headers {
		field_type, err := DetectType(data[index])
		if err != nil {
			println(err)
		}
		println("field name: ", header)
		println("detected type: ", field_type)
		fields = append(fields, models.Field{Name: header, Type: field_type})
	}

	return fields, nil
}

func DetectDelimiter(line string) rune {
	if strings.Contains(line, "|") {
		return '|'
	}
	return ',' // Default fallback
}

func DetectType(value string) (string, error) {
	v := strings.TrimSpace(value)

	switch {
	case isUUID(v):
		return "uuid", nil
	case isInteger(v):
		return "number", nil
	case isFloat(v):
		return "number", nil
	case isTimestamp(v):
		return "timestamp", nil
	case isEmail(v):
		return "email", nil
	default:
		return "unknown", nil
	}
}

// detectors
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

func isTimestamp(s string) bool {
	formats := []string{
		"2006-01-02 15:04:05",
		"02-01-06 15:04:05",
		"2006-01-02",
		"02/01/2006",
	}
	for _, layout := range formats {
		if _, err := time.Parse(layout, s); err == nil {
			return true
		}
	}
	return false
}

func isEmail(s string) bool {
	return strings.Contains(s, "@") && strings.Contains(s, ".")
}
