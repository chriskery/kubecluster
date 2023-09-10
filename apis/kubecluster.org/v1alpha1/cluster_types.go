/*
Copyright 2023.

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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// KubeClusterKind is the kind name.
	KubeClusterKind = "KubeCluster"
	// KubeClusterPlural is the TensorflowPlural for KubeCluster.
	KubeClusterPlural = "KubeClusters"
	// KubeClusterSingular is the singular for KubeCluster.
	KubeClusterSingular = "KubeCluster"

	// ControllerNameLabel represents the label key for the operator name, e.g. tf-operator, mpi-operator, etc.
	ControllerNameLabel = "kubeclusetr.org/controller-name"

	// ClusterNameLabel represents the label key for the cluster name, the value is the cluster name.
	ClusterNameLabel = "kubeclusetr.org/clusetr-name"

	ClusterDefaultContainerName = "kubenode"
)

type ClusterTemplate struct {
}

type ClusterType string

// ClusterSpec defines the desired state of KubeCluster
type ClusterSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of KubeCluster. Edit cluster_types.go to remove/update
	//Foo string `json:"foo,omitempty"`

	//ClusterType define the type of the cluster to be created
	ClusterType ClusterType `json:"clusterType"`

	//ClusterType define the template of the cluster to be created
	ClusterReplicaSpec map[ReplicaType]*ReplicaSpec `json:"ClusterReplicaSpec"`

	// MainContainer specifies name of the main container which
	// run as kubenode.
	MainContainer string `json:"mainContainer,omitempty"`

	// `RunPolicy` encapsulates various runtime policies of the distributed training
	// job, for example how to clean up resources and how long the job can stay
	// active.
	RunPolicy RunPolicy `json:"runPolicy,omitempty"`
}

// ClusterStatus defines the observed state of KubeCluster
type ClusterStatus struct {
	// Conditions is an array of current observed KubeCluster conditions.
	Conditions []ClusterCondition `json:"conditions,omitempty"`

	// ReplicaStatuses is map of ReplicaType and ReplicaStatus,
	// specifies the status of each replica.
	ReplicaStatuses map[ReplicaType]*ReplicaStatus `json:"replicaStatuses,omitempty"`

	// Represents time when the KubeCluster was acknowledged by the KubeCluster controller.
	// It is not guaranteed to be set in happens-before order across separate operations.
	// It is represented in RFC3339 form and is in UTC.
	StartTime *metav1.Time `json:"startTime,omitempty"`

	// Represents last time when the KubeCluster was reconciled. It is not guaranteed to
	// be set in happens-before order across separate operations.
	// It is represented in RFC3339 form and is in UTC.
	LastReconcileTime *metav1.Time `json:"lastReconcileTime,omitempty"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +resource:path=kubecluster
//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// KubeCluster is the Schema for the clusters API
type KubeCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClusterSpec   `json:"spec,omitempty"`
	Status ClusterStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimzxachinery/pkg/runtime.Object
// +resource:path=kubeclusters
//+kubebuilder:object:root=true

// KubeClusterList contains a list of KubeCluster
type KubeClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KubeCluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&KubeCluster{}, &KubeClusterList{})
	SchemeBuilder.SchemeBuilder.Register(addKubeClusterDefaultingFuncs)
}
