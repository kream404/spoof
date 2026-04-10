package models

import (
	"encoding/json"
	"fmt"
)

type Config struct {
	FileName       string `json:"file_name"`
	Delimiter      string `json:"delimiter"`
	RowCount       int    `json:"row_count,string"`
	FileCount      int    `json:"file_count,omitempty,string"`
	IncludeHeaders bool   `json:"include_headers"`
	Header         string `json:"header,omitempty"`
	Footer         string `json:"footer,omitempty"`
	Seed           string `json:"seed,omitempty"`
}

type Postprocess struct {
	Enabled   bool     `json:"enabled,omitempty"`
	Operation string   `json:"operation,omitempty"`
	Location  string   `json:"location,omitempty"`
	Region    string   `json:"region,omitempty"`
	Schema    string   `json:"schema,omitempty"`
	Table     string   `json:"table,omitempty"`
	Key       string   `json:"key,omitempty"`
	Alias     string   `json:"alias,omitempty"`
	Type      string   `json:"type,omitempty"`
	HasHeader bool     `json:"headers,omitempty"`
	TrimSpace bool     `json:"trim,omitempty"`
	Columns   []string `json:"columns,omitempty"`
	BatchSize int      `json:"batch,string,omitempty"`
}

type Profiles struct {
	Profiles []Profile `json:"profiles"`
}

type Profile struct {
	Hostname string `json:"hostname"`
	Port     string `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type CacheConfig struct {
	Hostname     string        `json:"hostname"`
	Port         string        `json:"port"`
	Username     string        `json:"username"`
	Password     string        `json:"password"`
	Name         string        `json:"name"`
	Statement    string        `json:"statement"`
	Source       string        `json:"source"`
	SeedSelector *SeedSelector `json:"selector,omitempty"`
	Region       string        `json:"region,omitempty"`
	Columns      []string      `json:"columns"`
}

type SeedSelector struct {
	Column string   `json:"column"`
	Keys   []string `json:"keys"`
}

type SourceList []string

type Field struct {
	Name       string     `json:"name"`
	Alias      string     `json:"alias,omitempty"`
	Type       string     `json:"type,omitempty"`
	Modifier   string     `json:"modifier,omitempty"`
	AutoInc    bool       `json:"auto_increment,omitempty"`
	ForeignKey string     `json:"foreign_key,omitempty"`
	Format     string     `json:"format,omitempty"`
	Length     int        `json:"length,omitempty"`
	Min        float64    `json:"min,omitempty"`
	Max        float64    `json:"max,omitempty"`
	Start      *int       `json:"start,omitempty"`
	Value      string     `json:"value,omitempty"`
	Values     string     `json:"values,omitempty"`
	Interval   int64      `json:"interval,omitempty"`
	Target     string     `json:"target,omitempty"`
	Seed       bool       `json:"seed,omitempty"`
	Selector   bool       `json:"selector,omitempty"`
	Function   string     `json:"function,omitempty"`
	Source     SourceList `json:"source,omitempty"`
	Template   string     `json:"template,omitempty"`
	Rate       *float64   `json:"rate,omitempty,string"`
	Regex      string     `json:"regex,omitempty"`
	Fields     []Field    `json:"fields,omitempty"`
	Repeat     string     `json:"repeat,omitempty"`
	Skip       bool       `json:"skip,omitempty"`
}

type Entity struct {
	Config      Config              `json:"config"`
	Postprocess Postprocess         `json:"postprocess,omitempty"`
	CacheConfig *CacheConfig        `json:"cache,omitempty"`
	Fields      []Field             `json:"fields"`
	Source      string              `json:"source,omitempty"`
	Output      []map[string]string `json:"output,omitempty"`
}

type FileConfig struct {
	Metadata  *ConfigMetadata        `json:"metadata,omitempty"`
	Variables map[string]VariableDoc `json:"variables,omitempty"`
	Files     []Entity               `json:"files"`
}

type Bundle struct {
	Metadata  *BundleMetadata        `json:"metadata,omitempty"`
	Variables map[string]VariableDoc `json:"variables,omitempty"`
	Files     []BundleFile           `json:"files"`
}

type ConfigMetadata struct {
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
}

type BundleMetadata struct {
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
}

type VariableDoc struct {
	Required      bool     `json:"required,omitempty"`
	Type          string   `json:"type,omitempty"`
	Description   string   `json:"description,omitempty"`
	Example       string   `json:"example,omitempty"`
	Default       string   `json:"default,omitempty"`
	AllowedValues []string `json:"allowed_values,omitempty"`
}

type Placeholder struct {
	Key  string
	Type string
}

func (p Placeholder) MarshalJSON() ([]byte, error) {
	s := "${" + p.Key
	if p.Type != "" {
		s += ":" + p.Type
	}
	s += "}"
	return json.Marshal(s)
}

func (s *SourceList) UnmarshalJSON(data []byte) error {
	var single string
	if err := json.Unmarshal(data, &single); err == nil {
		*s = SourceList{single}
		return nil
	}

	var many []string
	if err := json.Unmarshal(data, &many); err == nil {
		*s = SourceList(many)
		return nil
	}

	return fmt.Errorf("source must be a string or array of strings")
}

type BundleFile struct {
	Source string `json:"source"`
}

// TODO: get rid of this shit
func (c CacheConfig) MergeConfig(profile CacheConfig) CacheConfig {
	merged := profile
	merged.Statement = c.Statement
	merged.Source = c.Source
	merged.Columns = c.Columns

	if merged.Hostname == "" {
		merged.Hostname = c.Hostname
	}
	if merged.Port == "" {
		merged.Port = c.Port
	}
	if merged.Username == "" {
		merged.Username = c.Username
	}
	if merged.Password == "" {
		merged.Password = c.Password
	}
	if merged.Name == "" {
		merged.Name = c.Name
	}
	if c.SeedSelector != nil {
		merged.SeedSelector = c.SeedSelector
	}

	return merged
}
