package config

type Config struct {
	Providers []Provider `yaml:"providers"`
}

type Provider struct {
	Name      string     `yaml:"name"`
	Snapshots []Snapshot `yaml:"snapshots"`
}

type Snapshot struct {
	Type        string `yaml:"type"`
	ChainID     string `yaml:"chain_id"`
	URL         string `yaml:"url"`
	MetadataURL string `yaml:"metadata_url,omitempty"`
}
