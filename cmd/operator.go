package cmd

import (
	"log/slog"

	"github.com/go-logr/logr"
	"github.com/lukaspj/talos-cluster-operator/pkg/api/v1alpha1"
	"github.com/lukaspj/talos-cluster-operator/pkg/operator"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
)

var operatorCmd = &cobra.Command{
	Use:   "operator",
	Short: "Talos Cluster Operator",
	Long:  "Talos Cluster Operator",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := OperatorConfig()
		if err != nil {
			slog.Error("unable to load config", slog.String("error", err.Error()))
			return err
		}
		slog.Info("config loaded", slog.String("config", cfg.String()))

		slog.SetLogLoggerLevel(slog.LevelInfo)
		ctrl.SetLogger(logr.FromSlogHandler(slog.Default().Handler()))

		scheme := runtime.NewScheme()
		if err := v1alpha1.AddToScheme(scheme); err != nil {
			slog.Error("unable to add to scheme", "error", err)
			return err
		}

		mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
			Scheme:                  scheme,
			HealthProbeBindAddress:  cfg.ProbeAddr,
			LeaderElectionNamespace: cfg.Namespace,
			LeaderElection:          cfg.EnableLeaderElection,
			LeaderElectionID:        "election42.talos-cluster-operator.lukaspj.com",
			LivenessEndpointName:    "/livez",
			ReadinessEndpointName:   "/readyz",
		})
		if err != nil {
			slog.Error("unable to start manager", "error", err)
			return err
		}

		machineReconciler := &operator.TalosMachineReconciler{
			Client:   mgr.GetClient(),
			Scheme:   mgr.GetScheme(),
			Recorder: mgr.GetEventRecorderFor("talos-machine-controller"),
		}

		if err = machineReconciler.SetupWithManager(mgr); err != nil {
			slog.Error("unable to create controller", "error", err)
			return err
		}

		clusterReconciler := &operator.TalosClusterReconciler{
			Client:   mgr.GetClient(),
			Scheme:   mgr.GetScheme(),
			Recorder: mgr.GetEventRecorderFor("talos-cluster-controller"),
		}

		if err = clusterReconciler.SetupWithManager(mgr); err != nil {
			slog.Error("unable to create controller", "error", err)
			return err
		}

		if err = mgr.AddHealthzCheck("livez", healthz.Ping); err != nil {
			slog.Error("unable to set up health check", "error", err)
			return err
		}
		if err = mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
			slog.Error("unable to set up ready check", "error", err)
			return err
		}

		slog.Info("starting manager")
		if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
			slog.Error("problem running manager", "error", err)
			return err
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(operatorCmd)
}
