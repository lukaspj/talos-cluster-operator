package operator

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/lukaspj/talos-cluster-operator/pkg/api/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

func TestTalosMachineReconciler_Reconcile(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	require.NoError(t, v1alpha1.AddToScheme(scheme))

	pwd, err := os.Getwd()
	require.NoError(t, err)

	binPath := ""
	// TODO traverse parents
	rootDir, err := filepath.Abs(path.Join(pwd, "..", "..", ".."))
	require.NoError(t, err)
	require.NoError(t, filepath.WalkDir(path.Join(rootDir, "bin", "k8s"), func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			return nil
		}

		entries, err := os.ReadDir(path)
		if err != nil {
			return err
		}

		for _, entry := range entries {
			if strings.HasPrefix(entry.Name(), "kube-apiserver") {
				binPath = path
				return nil
			}
		}

		return nil
	}))

	require.NotEmpty(t, binPath)

	testEnv := &envtest.Environment{
		CRDDirectoryPaths:     []string{path.Join(rootDir, "crds")},
		BinaryAssetsDirectory: binPath,
		ErrorIfCRDPathMissing: true,
		Scheme:                scheme,
	}

	cfg, err := testEnv.Start()
	require.NoError(t, err)
	assert.NotNil(t, cfg)
	k8sClient, err := client.New(cfg, client.Options{Scheme: testEnv.Scheme})
	require.NoError(t, err)

	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme:         scheme,
		LeaderElection: false,
	})
	require.NoError(t, err)

	reconciler := &TalosMachineReconciler{
		Client: k8sClient,
		Scheme: scheme,
	}
	require.NoError(t, reconciler.SetupWithManager(mgr))

	t.Cleanup(func() {
		cancel()
		err := testEnv.Stop()
		if err != nil {
			slog.Warn("unable to stop test environment", "error", err)
		}
	})

	u, err := url.Parse(srv.URL)
	require.NoError(t, err)
	up, err := strconv.Atoi(u.Port())
	require.NoError(t, err)

	t.Run("reachable machine is available and ready", func(t *testing.T) {
		machine := &v1alpha1.Machine{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "m1",
				Namespace: "reachable-machine-test",
			},
			Spec: v1alpha1.MachineSpec{
				IP:   u.Hostname(),
				Port: up,
			},
		}

		require.NoError(t, k8sClient.Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: machine.Namespace}}))
		require.NoError(t, k8sClient.Create(ctx, machine))
		t.Cleanup(func() {
			require.NoError(t, k8sClient.Delete(ctx, machine))
			require.NoError(t, k8sClient.Delete(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: machine.Namespace}}))
		})

		_, err = reconciler.Reconcile(ctx, ctrl.Request{
			NamespacedName: types.NamespacedName{Namespace: machine.Namespace, Name: machine.Name},
		})
		require.NoError(t, err)

		var m v1alpha1.Machine
		require.NoError(t, k8sClient.Get(ctx, types.NamespacedName{Namespace: machine.Namespace, Name: machine.Name}, &m))

		assert.Len(t, m.Status.Conditions, 2)
		conditions := make(map[string]metav1.Condition)
		for _, condition := range m.Status.Conditions {
			conditions[condition.Type] = condition
		}
		if assert.Contains(t, conditions, "Ready") {
			assert.EqualValues(t, corev1.ConditionTrue, conditions["Ready"].Status)
		}
		if assert.Contains(t, conditions, "Available") {
			assert.EqualValues(t, corev1.ConditionTrue, conditions["Available"].Status)
		}
	})

	// Failed connectivity test
	srv.Close()
	t.Run("unreachable machine is not available", func(t *testing.T) {
		machine := &v1alpha1.Machine{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "m1",
				Namespace: "unreachable-machine-test",
			},
			Spec: v1alpha1.MachineSpec{
				IP:   u.Hostname(),
				Port: up,
			},
		}

		require.NoError(t, k8sClient.Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: machine.Namespace}}))
		require.NoError(t, k8sClient.Create(ctx, machine))
		t.Cleanup(func() {
			require.NoError(t, k8sClient.Delete(ctx, machine))
			require.NoError(t, k8sClient.Delete(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: machine.Namespace}}))
		})

		_, err = reconciler.Reconcile(ctx, ctrl.Request{
			NamespacedName: types.NamespacedName{Namespace: machine.Namespace, Name: machine.Name},
		})
		assert.Error(t, err)

		var m v1alpha1.Machine
		require.NoError(t, k8sClient.Get(ctx, types.NamespacedName{Namespace: machine.Namespace, Name: machine.Name}, &m))

		assert.Len(t, m.Status.Conditions, 2)
		conditions := make(map[string]metav1.Condition)
		for _, condition := range m.Status.Conditions {
			conditions[condition.Type] = condition
		}
		if assert.Contains(t, conditions, "Ready") {
			assert.EqualValues(t, corev1.ConditionFalse, conditions["Ready"].Status)
		}
		if assert.Contains(t, conditions, "Available") {
			assert.EqualValues(t, corev1.ConditionFalse, conditions["Available"].Status)
		}
	})
}
