package cmd

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "talos-cluster-operator",
	Short: "Talos Cluster Operator",
	Long:  "Talos Cluster Operator",
}

func Execute() {
	cobra.CheckErr(rootCmd.Execute())
}
