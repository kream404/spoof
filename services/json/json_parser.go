package json

import (
	"encoding/json"
	"os"

	"github.com/kream404/spoof/models"
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

func LoadProfiles(filepath string) (*models.Profiles, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}
	defer file.Close()


	decoder := json.NewDecoder(file)
	var profiles models.Profiles
	err = decoder.Decode(&profiles)
	if err != nil {
		return nil, err
	}

	println(ToJSONString(profiles));
	return &profiles, nil
}


func ToJSONString(data any) (string, error) {
	jsonBytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", err
	}
	return string(jsonBytes), nil
}
