package main

import "github.com/lukaspj/talos-cluster-operator/cmd"

//go:generate go tool setup-envtest use 1.32.0 --bin-dir bin/ -p path

func main() {
	cmd.Execute()
}
