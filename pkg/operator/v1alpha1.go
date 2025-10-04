package operator

import (
	"context"
	"fmt"
	"github.com/lukaspj/talos-cluster-operator/pkg/api/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"net"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
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

// "github.com/siderolabs/talos/pkg/machinery/client"

func (t *TalosMachineReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	machine := &v1alpha1.Machine{}
	if err := t.Get(ctx, req.NamespacedName, machine); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	ready := true

	conn, err := net.Dial("tcp", fmt.Sprintf("%s:6443", machine.Spec.IP))
	defer conn.Close()
	if err != nil {
		t.Recorder.Event(machine, "ConnectivityTest", "Failed", err.Error())
		ready = false
	}

	if !ready {
		t.Recorder.Event(machine, "Ready", "False", "One or more checks failed")
		return ctrl.Result{RequeueAfter: 5 * time.Second}, err
	}
	t.Recorder.Event(machine, "ConnectivityTest", "Success", "Connected")

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
		For(&v1alpha1.Machine{}).
		Complete(t)
}

// "github.com/siderolabs/talos/pkg/machinery/client"

func (t *TalosClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	cluster := &v1alpha1.Cluster{}
	if err := t.Get(ctx, req.NamespacedName, cluster); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	return ctrl.Result{}, nil
}
