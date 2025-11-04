package machineconfig

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"

	"github.com/cosi-project/runtime/pkg/resource"
	"github.com/cosi-project/runtime/pkg/state"
	"github.com/go-chi/chi/v5/middleware"
	talosctl "github.com/siderolabs/talos/pkg/machinery/client"
	"github.com/siderolabs/talos/pkg/machinery/config/generate"
	"github.com/siderolabs/talos/pkg/machinery/config/machine"
	"github.com/siderolabs/talos/pkg/machinery/config/types/v1alpha1"
	"github.com/siderolabs/talos/pkg/machinery/constants"
	yaml "go.yaml.in/yaml/v4"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

type Middleware func(http.Handler) http.Handler

type Server struct {
	Config Config
}

func NewServer(conf Config) *Server {
	return &Server{
		Config: conf,
	}
}

func (s *Server) Start(ctx context.Context) error {
	srv := http.Server{
		Addr: fmt.Sprintf(":%d", s.Config.Port),
		BaseContext: func(listener net.Listener) context.Context {
			return ctx
		},
		Handler: s.Routes(),
	}

	return srv.ListenAndServe()
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /readyz", s.Readyz)
	mux.HandleFunc("GET /livez", s.Livez)

	mux.HandleFunc("GET /machineconfig/new", s.NewMachineConfig)
	mux.HandleFunc("GET /machineconfig/new/{configName}", s.NewMachineConfig)

	return WithMiddleware(mux, middleware.RealIP, middleware.StripSlashes, middleware.Recoverer, middleware.RequestID)
}

func WithMiddleware(h http.Handler, m ...Middleware) http.Handler {
	for i := len(m); i > 0; i-- {
		h = m[i-1](h)
	}

	return h
}

func (s *Server) Readyz(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (s *Server) Livez(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (s *Server) NewMachineConfig(w http.ResponseWriter, req *http.Request) {
	uuid := req.URL.Query().Get("uuid")
	serial := req.URL.Query().Get("serial")
	mac := req.URL.Query().Get("mac")
	hostname := req.URL.Query().Get("hostname")
	configName := req.PathValue("configName")

	if configName == "" {
		configName = "default-machine-config"
	}

	slog.Info("new machine config request", "uuid", uuid, "serial", serial, "mac", mac, "hostname", hostname, "configName", configName)

	ns, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
	if err != nil {
		slog.Error("could not read current namespace, using default", "error", err)
		ns = []byte(s.Config.Namespace)
	}

	ctx := req.Context()

	clusterConfig, err := rest.InClusterConfig()
	if err != nil {
		slog.Error("failed to get in-cluster configuration", "error", err)

		var kubeconfig string
		if home := homedir.HomeDir(); home != "" {
			kubeconfig = filepath.Join(home, ".kube", "config")
		}

		// use the current context in kubeconfig
		clusterConfig, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			errorResponse(w, err, "failed to get kubernetes client configuration", http.StatusInternalServerError)
			return
		}
	}
	clientset, err := kubernetes.NewForConfig(clusterConfig)
	if err != nil {
		errorResponse(w, err, "failed to initialise Kubernetes client", http.StatusInternalServerError)
		return
	}

	configMap, err := clientset.CoreV1().ConfigMaps(string(ns)).Get(ctx, configName, metav1.GetOptions{})
	if err != nil {
		errorResponse(w, err, "could not get machine patch", http.StatusInternalServerError)
		return
	}

	var configPatch v1alpha1.Config
	err = yaml.Unmarshal([]byte(configMap.Data["machineconfig"]), &configPatch)
	if err != nil {
		errorResponse(w, err, "could not unmarshal machine patch", http.StatusInternalServerError)
		return
	}

	ctl, err := talosctl.New(ctx, talosctl.WithConfigFromFile(s.Config.TalosConfigPath))
	if err != nil {
		errorResponse(w, err, "could not initialise talosctl", http.StatusInternalServerError)
		return
	}

	talosNamespace := "config"
	resourceKind, err := ctl.ResolveResourceKind(ctx, &talosNamespace, "machineconfig")
	if err != nil {
		errorResponse(w, err, "could not get talos machine config kind", http.StatusInternalServerError)
		return
	}

	r, err := ctl.COSI.Get(ctx, resource.NewMetadata(talosNamespace, resourceKind.TypedSpec().Type, "v1alpha1", resource.VersionUndefined),
		state.WithGetUnmarshalOptions(state.WithSkipProtobufUnmarshal()))
	if err != nil {
		errorResponse(w, err, "could not get talos machine config spec", http.StatusInternalServerError)
		return
	}

	conf, err := yaml.Marshal(r.Spec())
	if err != nil {
		errorResponse(w, err, "could not marshal talos machine config spec", http.StatusInternalServerError)
		return
	}
	var machineConfig v1alpha1.Config
	slog.Info("machine config spec", "conf", string(conf))
	err = yaml.Unmarshal(conf, &machineConfig)
	if err != nil {
		errorResponse(w, err, "could not unmarshal talos machine config spec", http.StatusInternalServerError)
		return
	}

	input, err := generate.NewInput("_placeholder", "1.2.3.4", constants.DefaultKubernetesVersion)
	if err != nil {
		errorResponse(w, err, "failed to set new input", http.StatusInternalServerError)
		return
	}

	config, err := input.Config(machine.TypeWorker)
	if err != nil {
		errorResponse(w, err, "failed to generate config", http.StatusInternalServerError)
		return
	}

	config, err = config.PatchV1Alpha1(func(config *v1alpha1.Config) error {
		err = yaml.Unmarshal([]byte(configMap.Data["machineconfig"]), &config)
		if err != nil {
			return err
		}

		config.MachineConfig.MachineNetwork.NetworkHostname = "nucas-node-x"

		config.MachineConfig.MachineToken = machineConfig.MachineConfig.MachineToken
		config.MachineConfig.MachineCA.Crt = machineConfig.MachineConfig.MachineCA.Crt
		config.ClusterConfig.ClusterID = machineConfig.ClusterConfig.ClusterID
		config.ClusterConfig.ClusterSecret = machineConfig.ClusterConfig.ClusterSecret
		config.ClusterConfig.ControlPlane.Endpoint = machineConfig.ClusterConfig.ControlPlane.Endpoint
		config.ClusterConfig.ClusterName = machineConfig.ClusterConfig.ClusterName
		config.ClusterConfig.ClusterNetwork = machineConfig.ClusterConfig.ClusterNetwork
		config.ClusterConfig.BootstrapToken = machineConfig.ClusterConfig.BootstrapToken
		config.ClusterConfig.ClusterCA.Crt = machineConfig.ClusterConfig.ClusterCA.Crt

		return nil
	})
	if err != nil {
		errorResponse(w, err, "failed to patch config", http.StatusInternalServerError)
		return
	}

	bs, err := config.Bytes()
	if err != nil {
		errorResponse(w, err, "failed to serialize config", http.StatusInternalServerError)
		return
	}

	_, err = w.Write(bs)
	if err != nil {
		errorResponse(w, err, "failed to write config", http.StatusInternalServerError)
		return
	}
}

func errorResponse(w http.ResponseWriter, err error, msg string, code int) {
	w.WriteHeader(code)
	slog.Error(msg, "error", err)
	_, _ = w.Write([]byte(msg))
}

type mcYamlRepr struct{ resource.Resource }

func (m *mcYamlRepr) Spec() any { return &mcYamlSpec{res: m.Resource} }

type mcYamlSpec struct{ res resource.Resource }

func (m *mcYamlSpec) MarshalYAML() (any, error) {
	out, err := yaml.Marshal(m.res.Spec())
	if err != nil {
		return nil, err
	}

	return string(out), err
}
