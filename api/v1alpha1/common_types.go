// Copyright 2018 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package v1alpha1

import (
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// ReplicaIndexLabel represents the label key for the replica-index, e.g. 0, 1, 2.. etc
	ReplicaIndexLabel = "training.kubeflow.org/replica-index"

	// ReplicaTypeLabel represents the label key for the replica-type, e.g. ps, worker etc.
	ReplicaTypeLabel = "kubeclusetr.org/replica-type"
)

// ReplicaType represents the type of the replica. Each operator needs to define its
// own set of ReplicaTypes.
type ReplicaType string

// ReplicaStatus represents the current observed state of the replica.
type ReplicaStatus string

// KubeNode We use pod as the replica, so the replica spec is the pod spec
type KubeNode struct {
	v1.PodSpec
}

// ReplicaTemplate describes the data a replica(or a node) should have when created from a template
type ReplicaTemplate KubeNode

// ReplicaSpec is a description of the replica
type ReplicaSpec struct {
	// Replicas is the desired number of replicas of the given template.
	// If unspecified, defaults to 1.
	Replicas *int32 `json:"replicas,omitempty"`

	// Template is the object that describes the pod that
	// will be created for this replica. RestartPolicy in PodTemplateSpec
	// will be overide by RestartPolicy in ReplicaSpec
	Template ReplicaTemplate `json:"template"`
}

// ClusterCondition describes the state of the KubeCluster at a certain point.
type ClusterCondition struct {
	// Type of KubeCluster condition.
	Type ClusterConditionType `json:"type"`
	// Status of the condition, one of True, False, Unknown.
	Status v1.ConditionStatus `json:"status"`
	// The reason for the condition's last transition.
	Reason string `json:"reason,omitempty"`
	// A human readable message indicating details about the transition.
	Message string `json:"message,omitempty"`
	// The last time this condition was updated.
	LastUpdateTime metav1.Time `json:"lastUpdateTime,omitempty"`
	// Last time the condition transitioned from one status to another.
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
}

// ClusterConditionType defines all kinds of types of ClusterStatus.
type ClusterConditionType string

const (
	// ClusterCreated means the KubeCluster has been accepted by the system,
	// but one or more of the pods/services has not been started.
	// This includes time before pods being scheduled and launched.
	ClusterCreated ClusterConditionType = "Created"

	// ClusterRunning means all sub-resources (e.g. services/pods) of this KubeCluster
	// have been successfully scheduled and launched.
	// The training is running without error.
	ClusterRunning ClusterConditionType = "Running"

	// ClusterRestarting means one or more sub-resources (e.g. services/pods) of this KubeCluster
	// reached phase failed but maybe restarted according to it's restart policy
	// which specified by user in v1.PodTemplateSpec.
	// The training is freezing/pending.
	ClusterRestarting ClusterConditionType = "Restarting"

	// ClusterSuspended means the KubeCluster has been suspended.
	ClusterSuspended ClusterConditionType = "Suspended"

	// ClusterFailed means one or more sub-resources (e.g. services/pods) of this KubeCluster
	// reached phase failed with no restarting.
	// The training has failed its execution.
	ClusterFailed ClusterConditionType = "Failed"
)

// CleanKubeNodePolicy describes how to deal with pods when the KubeCluster is finished.
type CleanKubeNodePolicy string

const (
	CleanKubeNodePolicyUndefined CleanKubeNodePolicy = ""
	CleanKubeNodePolicyAll       CleanKubeNodePolicy = "All"
	CleanKubeNodePolicyRunning   CleanKubeNodePolicy = "Running"
	CleanKubeNodePolicyNone      CleanKubeNodePolicy = "None"
)

// RestartPolicy describes how the replicas should be restarted.
// Only one of the following restart policies may be specified.
// If none of the following policies is specified, the default one
// is RestartPolicyAlways.
type RestartPolicy string

const (
	RestartPolicyAlways    RestartPolicy = "Always"
	RestartPolicyOnFailure RestartPolicy = "OnFailure"
	RestartPolicyNever     RestartPolicy = "Never"

	// RestartPolicyExitCode policy means that user should add exit code by themselves,
	// The KubeCluster operator will check these exit codes to
	// determine the behavior when an error occurs:
	// - 1-127: permanent error, do not restart.
	// - 128-255: retryable error, will restart the pod.
	RestartPolicyExitCode RestartPolicy = "ExitCode"
)

// RunPolicy encapsulates various runtime policies of the distributed training
// KubeCluster, for example how to clean up resources and how long the KubeCluster can stay
// active.
type RunPolicy struct {
	// CleanKubeNodePolicy defines the policy to kill pods after the KubeCluster completes.
	// Default to None.
	CleanKubeNodePolicy *CleanKubeNodePolicy `json:"CleanKubeNodePolicy,omitempty"`

	// TTLSecondsAfterFinished is the TTL to clean up Clusters.
	// It may take extra ReconcilePeriod seconds for the cleanup, since
	// reconcile gets called periodically.
	// Default to infinite.
	TTLSecondsAfterFinished *int32 `json:"ttlSecondsAfterFinished,omitempty"`

	// Specifies the duration in seconds relative to the startTime that the KubeCluster may be active
	// before the system tries to terminate it; value must be positive integer.
	// +optional
	ActiveDeadlineSeconds *int64 `json:"activeDeadlineSeconds,omitempty"`

	// Optional number of retries before marking this KubeCluster failed.
	// +optional
	BackoffLimit *int32 `json:"backoffLimit,omitempty"`

	// SchedulingPolicy defines the policy related to scheduling, e.g. gang-scheduling
	// +optional
	SchedulingPolicy *SchedulingPolicy `json:"schedulingPolicy,omitempty"`

	// suspend specifies whether the KubeCluster controller should create Pods or not.
	// If a KubeCluster is created with suspend set to true, no Pods are created by
	// the KubeCluster controller. If a KubeCluster is suspended after creation (i.e. the
	// flag goes from false to true), the KubeCluster controller will delete all
	// active Pods and PodGroups associated with this KubeCluster.
	// Users must design their workload to gracefully handle this.
	// Suspending a KubeCluster will reset the StartTime field of the KubeCluster.
	//
	// Defaults to false.
	// +kubebuilder:default:=false
	// +optional
	Suspend *bool `json:"suspend,omitempty"`
}

// SchedulingPolicy encapsulates various scheduling policies of the distributed training
// KubeCluster, for example `minAvailable` for gang-scheduling.
type SchedulingPolicy struct {
	MinAvailable           *int32                                 `json:"minAvailable,omitempty"`
	Queue                  string                                 `json:"queue,omitempty"`
	MinResources           *map[v1.ResourceName]resource.Quantity `json:"minResources,omitempty"`
	PriorityClass          string                                 `json:"priorityClass,omitempty"`
	ScheduleTimeoutSeconds *int32                                 `json:"scheduleTimeoutSeconds,omitempty"`
}
