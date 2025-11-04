package machineconfig

import "fmt"

type Config struct {
	Port            int
	TalosConfigPath string
	Namespace       string
	MachineCIDR     string
}

func DefaultConfig() Config {
	return Config{
		Port:            4242,
		TalosConfigPath: "/var/run/secrets/talos.dev/config",
		Namespace:       "default",
		MachineCIDR:     "",
	}
}

func (c *Config) String() string {
	return fmt.Sprintf("Config{Port: %d, Namespace: %s, TalosConfigPath: %s, MachineCIDR: %s}", c.Port, c.Namespace, c.TalosConfigPath, c.MachineCIDR)
}
