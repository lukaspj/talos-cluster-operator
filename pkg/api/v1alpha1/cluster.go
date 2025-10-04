package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type MachineSet struct {
	Name     string               `json:"name"`
	Selector metav1.LabelSelector `json:"selector"`
	Config   string               `json:"config"`
}

type ClusterSpec struct {
	Nodes      MachineSet   `json:"nodes"`
	WorkerSets []MachineSet `json:"workerSets"`
}

type ClusterStatus struct {
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// Cluster describes where to locate some node running Talos
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
type Cluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClusterSpec   `json:"spec,omitempty"`
	Status ClusterStatus `json:"status,omitempty"`
}

// ClusterList contains a list of Machines
// +kubebuilder:object:root=true
type ClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Cluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Cluster{}, &ClusterList{})
}
