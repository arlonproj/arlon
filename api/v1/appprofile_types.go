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

// AppProfileSpec defines the desired state of AppProfile
type AppProfileSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	AppNames []string `json:"appNames,omitempty"`
}

// AppProfileStatus defines the observed state of AppProfile
type AppProfileStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Health values: healthy, degraded
	Health string `json:"health,omitempty"`
	// Names of apps that don't exist
	InvalidAppNames []string `json:"invalidAppNames,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// AppProfile is the Schema for the appprofiles API
type AppProfile struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AppProfileSpec   `json:"spec,omitempty"`
	Status AppProfileStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// AppProfileList contains a list of AppProfile
type AppProfileList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AppProfile `json:"items"`
}

func init() {
	SchemeBuilder.Register(&AppProfile{}, &AppProfileList{})
}
