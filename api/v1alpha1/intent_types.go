package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// IntentSpec defines the desired state of Intent
type IntentSpec struct {
	// Reference to Secret that is shared by Intent.
	SecretName string `json:"secretName"`
	// Namespaces that are whitelisted to access the Intent.
	// Supports either plain text or regex.
	// Empty list means allowing all namespaces.
	NamespaceWhitelist []string `json:"namespaceWhitelist,omitempty"`
}

// IntentState represents the current state of a Intent.
type IntentState string

const (
	// Error when locating referenced Secert.
	IntentStateError IntentState = "Error"
	// Secret has been located.
	IntentStateReady IntentState = "Ready"
)

// IntentStatus defines the observed state of Intent
type IntentStatus struct {
	State IntentState `json:"state"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.state"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// Intent is the Schema for the Intents API
type Intent struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   IntentSpec   `json:"spec,omitempty"`
	Status IntentStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// IntentList contains a list of Intent
type IntentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Intent `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Intent{}, &IntentList{})
}
