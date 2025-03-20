package json

import (
	"encoding/json"
	"os"

	"github.com/kream404/scratch/models"
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

func ToJSONString(config *models.FileConfig) (string, error) {
	jsonBytes, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return "", err
	}
	return string(jsonBytes), nil
}
