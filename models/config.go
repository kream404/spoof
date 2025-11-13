package models

type Config struct {
	FileName       string `json:"file_name"`
	Delimiter      string `json:"delimiter"`
	RowCount       int    `json:"row_count,string"` // <-- allow quoted numbers
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
	Type      string   `json:"type,omitempty"`
	HasHeader bool     `json:"headers,omitempty"`
	TrimSpace bool     `json:"trim,omitempty"`
	Columns   []string `json:"columns,omitempty"`
	BatchSize int      `json:"batch,string,omitempty"` // already string-coerced
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
	Hostname  string   `json:"hostname"`
	Port      string   `json:"port"`
	Username  string   `json:"username"`
	Password  string   `json:"password"`
	Name      string   `json:"name"`
	Statement string   `json:"statement"`
	Source    string   `json:"source"`
	Region    string   `json:"region,omitempty"`
	Columns   []string `json:"columns"`
}

type Field struct {
	Name       string   `json:"name"`
	Alias      string   `json:"alias,omitempty"`
	Type       string   `json:"type"`
	Modifier   *float64 `json:"modifier,omitempty"`
	AutoInc    bool     `json:"auto_increment,omitempty"`
	ForeignKey string   `json:"foreign_key,omitempty"`
	Format     string   `json:"format,omitempty"`
	Length     int      `json:"length,omitempty"`
	Min        float64  `json:"min,omitempty"`
	Max        float64  `json:"max,omitempty"`
	Value      string   `json:"value,omitempty"`
	Values     string   `json:"values,omitempty"`
	Interval   int64    `json:"interval,omitempty"`
	Target     string   `json:"target,omitempty"`
	Seed       bool     `json:"seed,omitempty"`
	Function   string   `json:"function,omitempty"`
	Source     string   `json:"source,omitempty"`
	Template   string   `json:"template,omitempty"`
	Rate       *int     `json:"rate,omitempty,string"`
	Regex      string   `json:"regex,omitempty"`
	Fields     []Field  `json:"fields,omitempty"` // top-level (your current config)
}

type Entity struct {
	Config      Config       `json:"config"`
	Postprocess Postprocess  `json:"postprocess,omitempty"`
	CacheConfig *CacheConfig `json:"cache,omitempty"`
	Fields      []Field      `json:"fields"`
	Source      string       `json:"source,omitempty"`
}

type FileConfig struct {
	Files []Entity `json:"files"`
}

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

	return merged
}
