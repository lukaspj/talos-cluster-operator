package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type NodeSpec struct {
}

type NodeStatus struct {
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// Node describes where to locate some node running Talos
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
type Node struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NodeSpec   `json:"spec,omitempty"`
	Status NodeStatus `json:"status,omitempty"`
}

// NodeList contains a list of Machines
// +kubebuilder:object:root=true
type NodeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Cluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Node{}, &NodeList{})
}
