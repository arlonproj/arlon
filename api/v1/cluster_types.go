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

// ClusterSpec defines the desired state of Cluster
type ClusterSpec struct {
	// The git location of the cluster template
	ClusterTemplate RepoSpec `json:"clusterTemplate"`
	// Optional override specification
	Override *OverrideSpec `json:"override,omitempty"`
	// Optional autoscaler specification
	Autoscaler *AutoscalerSpec `json:"autoscaler,omitempty"`
	// Optional Arlon Helm chart specification if defaults are not desired
	ArlonHelmChart *RepoSpec `json:"arlonHelmChart,omitempty"`
}

type RepoSpec struct {
	Url      string `json:"url"`
	Path     string `json:"path"`
	Revision string `json:"revision"`
}

type OverrideSpec struct {
	Patch string   `json:"patch"`
	Repo  RepoSpec `json:"repo"`
}

type AutoscalerSpec struct {
	// The external URL or host:port of the management cluster
	MgmtClusterHost string `json:"host"`
}

// ClusterStatus defines the observed state of Cluster
type ClusterStatus struct {
	// State has these possible values
	// - empty string: never processed by controller
	// - retrying: encountered a (possibly temporary) error, will retry later
	// - created: all resources created
	State string `json:"state,omitempty"`

	// The inner name of the Cluster resource in the cluster template.
	// Empty value means that the cluster template has not yet been validated.
	InnerClusterName string `json:"innerClusterName,omitempty"`

	// Indicates whether the override portion of the cluster
	// (the patch files in git) has been successfully created. Only
	// applicable to a cluster that specifies an override.
	OverrideSuccessful bool `json:"overrideSuccessful,omitempty"`

	// An optional message with details about the error for a 'retrying' state
	Message string `json:"message,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Cluster is the Schema for the clusters API
type Cluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClusterSpec   `json:"spec,omitempty"`
	Status ClusterStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ClusterList contains a list of Cluster
type ClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Cluster `json:"items"`
}

const (
	ClusterFinalizer = "cluster.core.arlon.io"
)

func init() {
	SchemeBuilder.Register(&Cluster{}, &ClusterList{})
}
