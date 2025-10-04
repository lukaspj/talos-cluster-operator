package cmd

import "github.com/spf13/cobra"

var operatorCmd = &cobra.Command{
	Use:   "operator",
	Short: "Talos Cluster Operator",
	Long:  "Talos Cluster Operator",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func init() {
	rootCmd.AddCommand(operatorCmd)
}
