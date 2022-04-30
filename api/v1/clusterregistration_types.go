/*
Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ClusterRegistrationSpec defines the desired state of ClusterRegistration
type ClusterRegistrationSpec struct {
	ClusterName             string `json:"clusterName,omitempty"`
	KubeconfigSecretName    string `json:"kubeconfigSecretName"`
	KubeconfigSecretKeyName string `json:"kubeconfigSecretKeyName"`
}

// ClusterRegistrationStatus defines the observed state of ClusterRegistration
type ClusterRegistrationStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	State   string `json:"state"`   // "retrying", "error", or "complete"
	Message string `json:"message"` // for "retrying" status
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// ClusterRegistration is the Schema for the clusterregistrations API
type ClusterRegistration struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClusterRegistrationSpec   `json:"spec,omitempty"`
	Status ClusterRegistrationStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ClusterRegistrationList contains a list of ClusterRegistration
type ClusterRegistrationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ClusterRegistration `json:"items"`
}

const (
	ClusterRegistrationFinalizer = "clusterregistration.core.arlon.io"
)

func init() {
	SchemeBuilder.Register(&ClusterRegistration{}, &ClusterRegistrationList{})
}
