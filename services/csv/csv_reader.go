package csv

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"os"
	"strings"

	"github.com/kream404/spoof/models"
	"github.com/kream404/spoof/services/detector"
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
		field, err := detector.InferField(header, col) //this returns a field with all the config, could rename
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
