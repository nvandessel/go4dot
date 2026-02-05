package config

// ConfigLoader defines the interface for loading configuration files.
// This interface allows for easier testing by providing a mockable contract.
type ConfigLoader interface {
	// Load reads and parses a config file from the given path.
	Load(path string) (*Config, error)

	// Discover searches for a config file in standard locations and returns its path.
	Discover() (string, error)
}

// DefaultConfigLoader is the production implementation of ConfigLoader.
type DefaultConfigLoader struct{}

// Load reads and parses a config file from the given path.
func (l *DefaultConfigLoader) Load(path string) (*Config, error) {
	return LoadFromPath(path)
}

// Discover searches for a config file in standard locations and returns its path.
func (l *DefaultConfigLoader) Discover() (string, error) {
	return FindConfig()
}

// NewConfigLoader creates a new DefaultConfigLoader.
func NewConfigLoader() ConfigLoader {
	return &DefaultConfigLoader{}
}
