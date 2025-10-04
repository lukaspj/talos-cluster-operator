// Package v1alpha1 contains API Schema definitions for the talos-cluster-operator v1alpha1 API group.
// +kubebuilder:object:generate=true
// +groupName=talos-cluster-operator.lukaspj.com
package v1alpha1

import (
	"github.com/lukaspj/talos-cluster-operator/pkg/api"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
)

var (
	// GroupVersion is group version used to register these objects.
	GroupVersion = schema.GroupVersion{Group: api.GroupName, Version: "v1alpha1"}

	// SchemeBuilder is used to add go types to the GroupVersionKind scheme.
	SchemeBuilder = &scheme.Builder{GroupVersion: GroupVersion}

	// AddToScheme adds the types in this group-version to the given scheme.
	AddToScheme = SchemeBuilder.AddToScheme
)
