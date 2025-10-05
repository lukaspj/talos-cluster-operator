package api

// Generate CRDs
//go:generate go run sigs.k8s.io/controller-tools/cmd/controller-gen@v0.19.0 rbac:roleName=manager-role crd:maxDescLen=2048 object paths="./v1alpha1" output:crd:dir=./../../crds

const (
	GroupName = "talos-cluster-operator.lukaspj.com"
)
