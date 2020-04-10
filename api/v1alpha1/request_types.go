package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type IntentReference struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

// RequestSpec defines the desired state of Request
type RequestSpec struct {
	// Refernce to intent to copy Secret from
	IntentReference IntentReference `json:"intentRef"`
	// If secret should be updated on change
	UpdateOnChange bool `json:"updateOnChange,omitempty"`
	// If hash should be appened to end of secret
	AppendHash bool `json:"appendHash,omitempty"`
}

// RequestState represents the current state of a Request.
type RequestState string

const (
	// Secret destination already exists.
	RAlreadyExists RequestState = "Secret Already Exists"
	// Could not find Intent referenced by Request.
	RNotFound RequestState = "Intent Not Found"
	// Referenced Intent is not ready
	RIntentError RequestState = "Intent not ready"
	// Request is not allowed from the Namespace
	RNotAllowed RequestState = "Request not allowed from Namespace"
	// Request fulfilled, meaning Secret is present.
	RReady RequestState = "Ready"
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
