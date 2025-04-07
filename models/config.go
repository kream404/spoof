package models

type Config struct {
	FileName   		 string  `json:"file_name"`
	Delimiter      string `json:"delimiter"`
	RowCount       int    `json:"rowcount,string"`
	IncludeHeaders bool   `json:"include_headers"`
}

type CacheConfig struct {
	Hostname    string  `json:"db_hostname"`
	Port   		 		 string  `json:"db_port"`
	Username   		 string  `json:"db_username"`
	Password   string  `json:"db_password"`
	Name			 string  `json:"db_name"`
	Statement  string	 `json:"statement"`
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
	SeedType   string   `json:"seed_type,omitempty"`
}

type Entity struct {
	Config   Config   `json:"config"`
	CacheConfig			 CacheConfig 	`json:"cache"`
	Fields []Field 		`json:"fields"`
}

type FileConfig struct {
	Files []Entity `json:"files"`
}

func (c CacheConfig) HasCache() bool {
	return c != CacheConfig{}
}
