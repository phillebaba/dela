package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ShareIntentReference struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

// ShareRequestSpec defines the desired state of ShareRequest
type ShareRequestSpec struct {
	// Refernce to intent to copy Secret from
	IntentReference ShareIntentReference `json:"intentRef"`
	// If secret should be updated on change
	UpdateOnChange bool `json:"updateOnChange,omitempty"`
	// If hash should be appened to end of secret
	AppendHash bool `json:"appendHash,omitempty"`
}

// ShareRequestState represents the current state of a ShareRequest.
type ShareRequestState string

const (
	SRAlreadyExists ShareRequestState = "Secret Already Exists"
	SRNotFound      ShareRequestState = "ShareIntent Not Found"
	SRReady         ShareRequestState = "Ready"
)

// ShareRequestStatus defines the observed state of ShareRequest
type ShareRequestStatus struct {
	State ShareRequestState `json:"state"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.state"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// ShareRequest is the Schema for the sharerequests API
type ShareRequest struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ShareRequestSpec   `json:"spec,omitempty"`
	Status ShareRequestStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// ShareRequestList contains a list of ShareRequest
type ShareRequestList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ShareRequest `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ShareRequest{}, &ShareRequestList{})
}
