package common

import (
	kubeclusterorgv1alpha1 "github.com/kubecluster/apis/kubecluster.org/v1alpha1"
	"github.com/kubecluster/pkg/controller/expectation"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ClusterSchemaReconciler interface {
	expectation.ControllerExpectationsInterface

	// GetDefaultContainerName Returns the default container name in pod
	GetDefaultContainerName() string

	// SetClusterSpec sets the cluster spec for the pod
	SetClusterSpec(kcluster *kubeclusterorgv1alpha1.KubeCluster, template *v1.PodTemplateSpec, rt kubeclusterorgv1alpha1.ReplicaType, indexStr string, configMap *v1.ConfigMap) error

	ReconcileConfigMap(kcluster *kubeclusterorgv1alpha1.KubeCluster, configMap *v1.ConfigMap) error

	IsController(spec map[kubeclusterorgv1alpha1.ReplicaType]*kubeclusterorgv1alpha1.ReplicaSpec, rType kubeclusterorgv1alpha1.ReplicaType, index int) bool

	ValidateV1KubeCluster(kcluster *kubeclusterorgv1alpha1.KubeCluster) error

	// UpdateClusterStatus updates the job status and job conditions
	UpdateClusterStatus(kcluster metav1.Object, replicas map[kubeclusterorgv1alpha1.ReplicaType]*kubeclusterorgv1alpha1.ReplicaSpec, clusterStatus *kubeclusterorgv1alpha1.ClusterStatus) error

	UpdateConfigMap(kcluster *kubeclusterorgv1alpha1.KubeCluster, deepCopy *v1.ConfigMap) error

	Default(kcluster *kubeclusterorgv1alpha1.KubeCluster)
}
