package common

import (
	kubeclusterorgv1alpha1 "github.com/kubecluster/apis/kubecluster.org/v1alpha1"
	"github.com/kubecluster/pkg/core"
	corev1 "k8s.io/api/core/v1"
)

// initializeReplicaStatuses initializes the ReplicaStatuses for replica.
func initializeReplicaStatuses(clusterStatus *kubeclusterorgv1alpha1.ClusterStatus, rtype kubeclusterorgv1alpha1.ReplicaType) {
	core.InitializeReplicaStatuses(clusterStatus, rtype)
}

// updateJobReplicaStatuses updates the JobReplicaStatuses according to the pod.
func updateJobReplicaStatuses(clusterStatus *kubeclusterorgv1alpha1.ClusterStatus, rtype kubeclusterorgv1alpha1.ReplicaType, pod *corev1.Pod) {
	core.UpdateJobReplicaStatuses(clusterStatus, rtype, pod)
}
