package machineconfig

import "fmt"

type Config struct {
	Port            int
	TalosConfigPath string
	Namespace       string
}

func DefaultConfig() Config {
	return Config{
		Port:            4242,
		TalosConfigPath: "/var/run/secrets/talos.dev/config",
		Namespace:       "default",
	}
}

func (c *Config) String() string {
	return fmt.Sprintf("Config{Port: %d, Namespace: %s, TalosConfigPath: %s}", c.Port, c.Namespace, c.TalosConfigPath)
}
