package ctrlcommon

import (
	kubeclusterorgv1alpha1 "github.com/chriskery/kubecluster/apis/kubecluster.org/v1alpha1"
	"github.com/chriskery/kubecluster/pkg/core"
	corev1 "k8s.io/api/core/v1"
)

// initializeReplicaStatuses initializes the ReplicaStatuses for replica.
func initializeReplicaStatuses(clusterStatus *kubeclusterorgv1alpha1.ClusterStatus, rtype kubeclusterorgv1alpha1.ReplicaType) {
	core.InitializeReplicaStatuses(clusterStatus, rtype)
}

// updateClusterReplicaStatuses updates the ClusterReplicaStatuses according to the pod.
func updateClusterReplicaStatuses(clusterStatus *kubeclusterorgv1alpha1.ClusterStatus, rtype kubeclusterorgv1alpha1.ReplicaType, pod *corev1.Pod) {
	core.UpdateClusterReplicaStatuses(clusterStatus, rtype, pod)
}
