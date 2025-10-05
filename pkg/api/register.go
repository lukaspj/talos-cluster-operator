package api

// Generate CRDs
//go:generate go tool controller-gen rbac:roleName=manager-role crd:maxDescLen=2048 object paths="./v1alpha1" output:crd:dir=./../../crds

const (
	GroupName = "talos-cluster-operator.lukaspj.com"
)
