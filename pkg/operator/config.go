package operator

type Config struct {
	ProbeAddr            string
	Namespace            string
	EnableLeaderElection bool
	ConfigSecretName     string
	ConfigSecretKey      string
}

func DefaultConfig() Config {
	return Config{
		ProbeAddr:            ":8081",
		Namespace:            "talos-cluster-operator",
		EnableLeaderElection: true,
		ConfigSecretName:     "talos-config",
		ConfigSecretKey:      "config",
	}
}

func (c *Config) String() string {
	return "Config{}"
}
