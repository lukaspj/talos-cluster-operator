package cmd

import (
	"github.com/lukaspj/go-fang"
	"github.com/lukaspj/talos-cluster-operator/pkg/operator"
)

func Config(envPrefix string) (operator.Config, error) {
	config, err := fang.New[operator.Config]().
		WithAutomaticEnv("TALOS_OPERATOR").
		Load()

	return config, err
}
