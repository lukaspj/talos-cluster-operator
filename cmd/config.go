package cmd

import (
	"github.com/lukaspj/go-fang"
	"github.com/lukaspj/talos-cluster-operator/pkg/machineconfig"
	"github.com/lukaspj/talos-cluster-operator/pkg/operator"
)

func OperatorConfig() (operator.Config, error) {
	config, err := fang.New[operator.Config]().
		WithDefault(operator.DefaultConfig()).
		WithAutomaticEnv("TALOS_OPERATOR").
		Load()

	return config, err
}

func ServerConfig() (machineconfig.Config, error) {
	config, err := fang.New[machineconfig.Config]().
		WithDefault(machineconfig.DefaultConfig()).
		WithAutomaticEnv("TALOS_OPERATOR").
		Load()

	return config, err
}
