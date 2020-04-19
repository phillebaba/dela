package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// IntentReference contains the name and namespace of an Intent.
type IntentReference struct {
	// Name of Intent.
	Name string `json:"name"`
	// Namespace of Intent.
	Namespace string `json:"namespace"`
}

// RequestSpec defines the desired state of Request
type RequestSpec struct {
	// Identifier of Intent to make Request for.
	IntentRef IntentReference `json:"intentRef"`
	// Overrides ObjectMeta of the Secret copy.
	SecretObjectMeta metav1.ObjectMeta `json:"secretMetadata"`
}

// RequestState represents the current state of a Request.
type RequestState string

const (
	// Error has occured when copying the Secret.
	RequestStateError RequestState = "Error"
	// Request fulfilled and the Secret has been copied.
	RequestStateReady RequestState = "Ready"
)

// RequestStatus defines the observed state of Request
type RequestStatus struct {
	State RequestState `json:"state"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.state"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// Request is the Schema for the Requests API
type Request struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RequestSpec   `json:"spec,omitempty"`
	Status RequestStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// RequestList contains a list of Request
type RequestList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Request `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Request{}, &RequestList{})
}
