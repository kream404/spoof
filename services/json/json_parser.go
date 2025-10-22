package json

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/kream404/spoof/models"
	log "github.com/kream404/spoof/services/logger"
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

var vars = map[string]string{}

func SetVars(m map[string]string) { vars = m }

func PerformVariableInjection(config models.FileConfig) (*models.FileConfig, error) {
	var tokenRE = regexp.MustCompile(`\{\{\s*([A-Za-z0-9_.\-]+)\s*\}\}`)

	raw, err := json.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("marshal: %w", err)
	}
	jsonStr := string(raw)

	// Replace each provided var
	for k, v := range vars {
		re := regexp.MustCompile(`\{\{\s*` + regexp.QuoteMeta(k) + `\s*\}\}`)
		jsonStr = re.ReplaceAllString(jsonStr, v)
	}

	// Fail if any placeholders remain
	if unresolved := tokenRE.FindAllStringSubmatch(jsonStr, -1); len(unresolved) > 0 {
		missing := make([]string, 0, len(unresolved))
		seen := map[string]struct{}{}
		for _, m := range unresolved {
			if len(m) == 2 {
				if _, ok := seen[m[1]]; !ok {
					seen[m[1]] = struct{}{}
					missing = append(missing, m[1])
				}
			}
		}
		return nil, fmt.Errorf("missing injected variable(s): %s", strings.Join(missing, ", "))
	}

	// Back into the struct
	var out models.FileConfig
	if err := json.Unmarshal([]byte(jsonStr), &out); err != nil {
		return nil, fmt.Errorf("unmarshal after injection: %w", err)
	}
	return &out, nil
}

func PerformVariableInjectionWithMap(config models.FileConfig, vars map[string]string) (*models.FileConfig, error) {
	SetVars(vars)
	return PerformVariableInjection(config)
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

	log.Debug("Profile loaded ", "profile", profiles)
	return &profiles, nil
}

func ToJSONString(data any) (string, error) {
	jsonBytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", err
	}
	return string(jsonBytes), nil
}
