package machineconfig

import "fmt"

type Config struct {
	Port int
}

func DefaultConfig() Config {
	return Config{
		Port: 4242,
	}
}

func (c *Config) String() string {
	return fmt.Sprintf("Config{Port: %d}", c.Port)
}
