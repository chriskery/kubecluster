package common

import (
	kubeclusterorgv1alpha1 "github.com/chriskery/kubecluster/apis/kubecluster.org/v1alpha1"
	"github.com/chriskery/kubecluster/pkg/controller/expectation"
	v1 "k8s.io/api/core/v1"
)

type ClusterSchemaReconciler interface {
	expectation.ControllerExpectationsInterface

	// GetDefaultContainerName Returns the default container name in pod
	GetDefaultContainerName() string

	ReconcileConfigMap(kcluster *kubeclusterorgv1alpha1.KubeCluster, configMap *v1.ConfigMap) error

	ValidateV1KubeCluster(kcluster *kubeclusterorgv1alpha1.KubeCluster) error

	UpdateConfigMap(kcluster *kubeclusterorgv1alpha1.KubeCluster, deepCopy *v1.ConfigMap) error

	Default(kcluster *kubeclusterorgv1alpha1.KubeCluster)

	// SetClusterSpec sets the cluster spec for the pod
	SetClusterSpec(
		kcluster *kubeclusterorgv1alpha1.KubeCluster,
		template *v1.PodTemplateSpec,
		rt kubeclusterorgv1alpha1.ReplicaType,
		indexStr string,
		configMap *v1.ConfigMap,
	) error

	IsController(
		spec map[kubeclusterorgv1alpha1.ReplicaType]*kubeclusterorgv1alpha1.ReplicaSpec,
		rType kubeclusterorgv1alpha1.ReplicaType,
		index int,
	) bool

	// UpdateClusterStatus updates the cluster status and cluster conditions
	UpdateClusterStatus(
		kcluster *kubeclusterorgv1alpha1.KubeCluster,
		clusterStatus *kubeclusterorgv1alpha1.ClusterStatus,
		rtype kubeclusterorgv1alpha1.ReplicaType,
		spec *kubeclusterorgv1alpha1.ReplicaSpec,
	)
}
