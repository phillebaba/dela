package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ShareIntentSpec defines the desired state of ShareIntent
type ShareIntentSpec struct {
	AllowedNamespaces []string `json:"allowedNamespaces"`
	SecretReference   string   `json:"secretRef"`
}

// ShareIntentStatus defines the observed state of ShareIntent
type ShareIntentStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true

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
