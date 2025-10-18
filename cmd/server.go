package cmd

import (
	"log/slog"

	"github.com/lukaspj/talos-cluster-operator/pkg/machineconfig"
	"github.com/spf13/cobra"
)

var machineconfigCmd = &cobra.Command{
	Use:   "server",
	Short: "Talos Server",
	Long:  "Talos Server",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := ServerConfig()
		if err != nil {
			slog.Error("unable to load config", slog.String("error", err.Error()))
			return err
		}
		slog.Info("config loaded", slog.String("config", cfg.String()))

		slog.SetLogLoggerLevel(slog.LevelInfo)

		srv := machineconfig.NewServer(cfg)

		return srv.Start(cmd.Context())
	},
}

func init() {
	rootCmd.AddCommand(machineconfigCmd)
}
