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

// ProfileSpec defines the desired state of Profile.
// The RepoXXX fields are set for a dynamic profile, and empty otherwise.
type ProfileSpec struct {
	Description string   `json:"description,omitempty"`
	Tags        []string `json:"tags,omitempty"`

	// Names of bundles in this profile. Order is not significant.
	Bundles []string `json:"bundles,omitempty"`
	// Optional parameter overrides for specific bundles
	Overrides []Override `json:"overrides,omitempty"`
	// URL of git repository where dynamic profile shall be stored
	RepoUrl string `json:"repoUrl,omitempty"`
	// Path within git repository
	RepoPath string `json:"repoPath,omitempty"`
	// Git revision (tag/branch/commit)
	RepoRevision string `json:"repoRevision,omitempty"`
}

type Override struct {
	Bundle string `json:"bundle"`
	Key    string `json:"key"`
	Value  string `json:"value"`
}

// ProfileStatus defines the observed state of Profile
type ProfileStatus struct {
	// State reaches 'synced' value when git repo is synchronized with dynamic profile
	State string `json:"state"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Profile is the Schema for the profiles API
type Profile struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ProfileSpec   `json:"spec,omitempty"`
	Status ProfileStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ProfileList contains a list of Profile
type ProfileList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Profile `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Profile{}, &ProfileList{})
}
