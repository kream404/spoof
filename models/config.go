package models

type Config struct {
	FileName       string `json:"file_name"`
	Delimiter      string `json:"delimiter"`
	RowCount       int    `json:"rowcount,string"`
	IncludeHeaders bool   `json:"include_headers"`
	Seed           string `json:"seed,omitempty"`
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
	Hostname  string `json:"hostname"`
	Port      string `json:"port"`
	Username  string `json:"username"`
	Password  string `json:"password"`
	Name      string `json:"name"`
	Statement string `json:"statement"`
}

type Field struct {
	Name       string   `json:"name"`
	Alias      string   `json:"alias,omitempty"`
	Type       string   `json:"type"`
	Modifier   *float64 `json:"modifier,omitempty"`
	AutoInc    bool     `json:"auto_increment,omitempty"`
	ForeignKey string   `json:"foreign_key,omitempty"`
	Format     string   `json:"format,omitempty"`
	Min        float64  `json:"min,omitempty"`
	Max        float64  `json:"max,omitempty"`
	Value      string   `json:"value,omitempty"`
	Values     string   `json:"values,omitempty"`
	Target     string   `json:"target,omitempty"`
	SeedType   string   `json:"seed_type,omitempty"`
}

type Entity struct {
	Config      Config       `json:"config"`
	CacheConfig *CacheConfig `json:"cache,omitempty"`
	Fields      []Field      `json:"fields"`
}

type FileConfig struct {
	Files []Entity `json:"files"`
}

func (c CacheConfig) MergeConfig(profile CacheConfig) CacheConfig {
	merged := profile
	merged.Statement = c.Statement

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
