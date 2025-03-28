package models

// Config defines global settings
type Config struct {
	FileName   string  `json:"file_name"`
	Delimiter      string `json:"delimiter"`
	RowCount       int    `json:"rowcount,string"`
	IncludeHeaders bool   `json:"include_headers"`
}

// Field defines the structure of a field
type Field struct {
	Name       string   `json:"name"`
	Type       string   `json:"type"`
	AutoInc    bool     `json:"auto_increment,omitempty"`
	ForeignKey string   `json:"foreign_key,omitempty"`
	Format     string   `json:"format,omitempty"`
	Min        float64  `json:"min,omitempty"`
	Max        float64  `json:"max,omitempty"`
	Values     []string `json:"values,omitempty"`
}

// Entity represents a database-like table
type Entity struct {
	Config   Config   `json:"config"`
	Fields []Field `json:"fields"`
}

// Root structure for JSON parsing
type FileConfig struct {
	Files []Entity `json:"files"`
}
