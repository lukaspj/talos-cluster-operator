package operator

import (
	"context"
	"github.com/lukaspj/talos-cluster-operator/pkg/api/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"path/filepath"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"strings"
	"testing"
)

func TestTalosMachineReconciler_Reconcile(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	scheme := runtime.NewScheme()
	require.NoError(t, v1alpha1.AddToScheme(scheme))

	pwd, err := os.Getwd()
	require.NoError(t, err)

	binPath := ""
	// TODO traverse parents
	require.NoError(t, filepath.WalkDir(path.Join(pwd, "..", "..", "bin", "k8s"), func(path string, d os.DirEntry, err error) error {
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
		CRDDirectoryPaths:     []string{path.Join(pwd, "crds")},
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
		require.NoError(t, testEnv.Stop())
	})

	machine := &v1alpha1.Machine{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "m1",
			Namespace: "test",
		},
		Spec: v1alpha1.MachineSpec{
			IP: srv.URL,
		},
	}

	_, err = reconciler.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{Namespace: machine.Namespace, Name: machine.Name},
	})
	require.NoError(t, err)

	var m v1alpha1.Machine
	require.NoError(t, k8sClient.Get(ctx, types.NamespacedName{Namespace: machine.Namespace, Name: machine.Name}, &m))

	assert.Equal(t, "Ready", m.Status.Conditions[0].Type)
}
