package cmd

import (
	"github.com/lukaspj/go-fang"
	"github.com/lukaspj/talos-cluster-operator/pkg/operator"
)

func Config() (operator.Config, error) {
	config, err := fang.New[operator.Config]().
		WithDefault(operator.DefaultConfig()).
		WithAutomaticEnv("TALOS_OPERATOR").
		Load()

	return config, err
}
