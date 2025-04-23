package json_test

import (
	"path/filepath"
	"testing"

	"github.com/kream404/spoof/services/json"
)

func TestLoadConfig(t *testing.T) {
	// Use existing schema file
	schemaPath := filepath.Join("../../test", "test_config.json")
	config, err := json.LoadConfig(schemaPath)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if len(config.Files) != 1 {
		t.Fatalf("Expected 1 file entry, got %d", len(config.Files))
	}

	file := config.Files[0]

	if file.Config.FileName != "testfile.csv" {
		t.Errorf("Expected FileName 'testfile.csv', got '%s'", file.Config.FileName)
	}

	if file.Config.Delimiter != "|" {
		t.Errorf("Expected Delimiter '|', got '%s'", file.Config.Delimiter)
	}

	if file.Config.IncludeHeaders != true {
		t.Errorf("Expected IncludeHeaders true, got false")
	}

	if len(file.Fields) != 5 {
		t.Errorf("Expected 5 fields, got %d", len(file.Fields))
	}
}

func TestToJSONString(t *testing.T) {
	data := map[string]interface{}{
		"example": true,
		"count":   10,
	}
	str, err := json.ToJSONString(data)
	if err != nil {
		t.Fatalf("ToJSONString returned error: %v", err)
	}
	if str == "" {
		t.Error("Expected non-empty JSON string")
	}
}
