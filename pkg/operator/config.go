package operator

type Config struct {
	ProbeAddr            string
	Namespace            string
	EnableLeaderElection bool
}

func DefaultConfig() Config {
	return Config{
		ProbeAddr:            ":8081",
		Namespace:            "talos-cluster-operator",
		EnableLeaderElection: true,
	}
}

func (c *Config) String() string {
	return "Config{}"
}
