package models

type Config struct {
	FileName   string  `json:"file_name"`
	Delimiter      string `json:"delimiter"`
	RowCount       int    `json:"rowcount,string"`
	IncludeHeaders bool   `json:"include_headers"`
}

type Field struct {
	Name       string   `json:"name"`
	Type       string   `json:"type"`
	AutoInc    bool     `json:"auto_increment,omitempty"`
	ForeignKey string   `json:"foreign_key,omitempty"`
	Format     string   `json:"format,omitempty"`
	Min        float64    `json:"min,omitempty"`
	Max        float64    `json:"max,omitempty"`
	Values     string 	`json:"values,omitempty"`
}

type Entity struct {
	Config   Config   `json:"config"`
	Fields []Field 		`json:"fields"`
}

type FileConfig struct {
	Files []Entity `json:"files"`
}
