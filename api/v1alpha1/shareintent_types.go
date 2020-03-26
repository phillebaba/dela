package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ShareIntentSpec defines the desired state of ShareIntent
type ShareIntentSpec struct {
	// Reference to Secret that is shared by Intent.
	SecretReference string `json:"secretRef"`
	// Namespaces that are allowed to access the Intent.
	// Supports either plain text or regex.
	// Empty list means allowing all namespaces.
	AllowedNamespaces []string `json:"allowedNamespaces,omitempty"`
}

// ShareIntentState represents the current state of a ShareIntent.
type ShareIntentState string

const (
	NotFound ShareIntentState = "Secret Not Found"
	Ready    ShareIntentState = "Ready"
)

// ShareIntentStatus defines the observed state of ShareIntent
type ShareIntentStatus struct {
	State ShareIntentState `json:"state"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.state"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// ShareIntent is the Schema for the shareintents API
type ShareIntent struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ShareIntentSpec   `json:"spec,omitempty"`
	Status ShareIntentStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// ShareIntentList contains a list of ShareIntent
type ShareIntentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ShareIntent `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ShareIntent{}, &ShareIntentList{})
}
