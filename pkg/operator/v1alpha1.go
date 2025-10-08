package operator

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"maps"
	"net"
	"slices"
	"time"

	"github.com/lukaspj/talos-cluster-operator/pkg/api/v1alpha1"
	clusterapi "github.com/siderolabs/talos/pkg/machinery/api/cluster"
	talosctl "github.com/siderolabs/talos/pkg/machinery/client"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type TalosMachineReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

func (t *TalosMachineReconciler) SetupWithManager(mgr ctrl.Manager) error {
	t.Recorder = mgr.GetEventRecorderFor("talos-machine-controller")
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.Machine{}).
		Complete(t)
}

func (t *TalosMachineReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	machine := &v1alpha1.Machine{}
	if err := t.Get(ctx, req.NamespacedName, machine); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	ready := true

	conditions := make(map[string]metav1.Condition)
	for _, condition := range machine.Status.Conditions {
		conditions[condition.Type] = condition
	}

	port := machine.Spec.Port
	if port == 0 {
		port = 6443
	}
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", machine.Spec.IP, port))
	if err != nil {
		t.Recorder.Event(machine, "Warning", "ConnectivityTestFailed", err.Error())
		ready = false
		oldAvailable, ok := conditions["Available"]
		newAvailable := metav1.Condition{
			Type:               "Available",
			Status:             metav1.ConditionFalse,
			Reason:             "ConnectivityTestFailed",
			Message:            err.Error(),
			ObservedGeneration: oldAvailable.ObservedGeneration,
			LastTransitionTime: oldAvailable.LastTransitionTime,
		}
		if oldAvailable.Status == metav1.ConditionTrue || !ok {
			newAvailable.ObservedGeneration = machine.Generation
			newAvailable.LastTransitionTime = metav1.Now()
		}
		conditions["Available"] = newAvailable
	} else {
		defer conn.Close()
		oldAvailable, ok := conditions["Available"]
		newAvailable := metav1.Condition{
			Type:               "Available",
			Status:             metav1.ConditionTrue,
			Reason:             "ConnectivityTestSucceeded",
			Message:            fmt.Sprintf("Managed to establish a connection to the machine at %s:%d"+machine.Spec.IP, port),
			ObservedGeneration: oldAvailable.ObservedGeneration,
			LastTransitionTime: oldAvailable.LastTransitionTime,
		}
		if oldAvailable.Status == metav1.ConditionFalse || !ok {
			newAvailable.ObservedGeneration = machine.Generation
			newAvailable.LastTransitionTime = metav1.Now()
		}
		conditions["Available"] = newAvailable
	}

	if !ready {
		t.Recorder.Event(machine, "Warning", "Unready", "One or more checks failed")
		oldReady, ok := conditions["Ready"]
		newReady := metav1.Condition{
			Type:               "Ready",
			Status:             metav1.ConditionFalse,
			Reason:             "ChecksFailed",
			Message:            "One or more checks failed",
			ObservedGeneration: oldReady.ObservedGeneration,
			LastTransitionTime: oldReady.LastTransitionTime,
		}
		if oldReady.Status == metav1.ConditionTrue || !ok {
			newReady.ObservedGeneration = machine.Generation
			newReady.LastTransitionTime = metav1.Now()
		}
		conditions["Ready"] = newReady

		machine.Status.Conditions = slices.Collect(maps.Values(conditions))

		statusErr := t.Status().Update(ctx, machine)
		if statusErr != nil {
			slog.Error("unable to update machine status", "error", statusErr)
		}
		return ctrl.Result{RequeueAfter: 5 * time.Second}, err
	}

	oldReady, ok := conditions["Ready"]
	newReady := metav1.Condition{
		Type:               "Ready",
		Status:             metav1.ConditionTrue,
		Reason:             "Ready",
		Message:            "All checks passed",
		ObservedGeneration: oldReady.ObservedGeneration,
		LastTransitionTime: oldReady.LastTransitionTime,
	}
	if oldReady.Status == metav1.ConditionFalse || !ok {
		newReady.ObservedGeneration = machine.Generation
		newReady.LastTransitionTime = metav1.Now()
	}
	conditions["Ready"] = newReady

	machine.Status.Conditions = slices.Collect(maps.Values(conditions))

	err = t.Status().Update(ctx, machine)
	if err != nil {
		slog.Error("unable to update machine status", "error", err)
	}

	node := &v1alpha1.Node{}
	if err = t.Get(ctx, req.NamespacedName, node); err != nil && !k8serrors.IsNotFound(err) {
		slog.Error("unable to get node", "error", err)
		return ctrl.Result{}, err
	}

	if node.Name == "" {
		node.Name = machine.Name
		node.Namespace = machine.Namespace
	}

	return ctrl.Result{}, nil
}

type TalosClusterReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

func (t *TalosClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	t.Recorder = mgr.GetEventRecorderFor("talos-cluster-controller")
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.Cluster{}).
		Complete(t)
}

// "github.com/siderolabs/talos/pkg/machinery/client"

func (t *TalosClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	cluster := &v1alpha1.Cluster{}
	if err := t.Get(ctx, req.NamespacedName, cluster); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	machines := &v1alpha1.MachineList{}

	machineSelector, err := metav1.LabelSelectorAsSelector(&cluster.Spec.Nodes.Selector)
	if err != nil {
		return ctrl.Result{}, err
	}
	if err := t.List(ctx, machines, client.MatchingLabelsSelector{Selector: machineSelector}); err != nil {
		return ctrl.Result{}, err
	}

	var endpoints []string
	for _, m := range machines.Items {
		port := m.Spec.Port
		if port == 0 {
			port = 6443
		}
		endpoints = append(endpoints, fmt.Sprintf("%s:%d", m.Spec.IP, port))
	}

	ctl, err := talosctl.New(ctx, talosctl.WithEndpoints(endpoints...), talosctl.WithConfigFromFile("/var/run/secrets/talos.dev"))
	if err != nil {
		return ctrl.Result{}, err
	}

	hcClient, err := ctl.ClusterHealthCheck(ctx, time.Minute, &clusterapi.ClusterInfo{})
	if err != nil {
		return ctrl.Result{}, err
	}

	for {
		status, err := hcClient.Recv()
		if err == nil {
			slog.Info("health check status", "message", status.Message, "metadata", status.Metadata)
		}

		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}
