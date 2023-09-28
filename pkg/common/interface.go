package common

import (
	kubeclusterorgv1alpha1 "github.com/chriskery/kubecluster/apis/kubecluster.org/v1alpha1"
	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// ControllerInterface defines the Interface to be implemented by custom operators. e.g. tf-operator needs to implement this interface
type ControllerInterface interface {
	// ControllerName Returns the Controller name
	ControllerName() string

	// GetAPIGroupVersionKind Returns the GroupVersionKind of the API
	GetAPIGroupVersionKind() schema.GroupVersionKind

	// GetAPIGroupVersion Returns the GroupVersion of the API
	GetAPIGroupVersion() schema.GroupVersion

	// GetClusterFromInformerCache Returns the Cluster from Informer Cache
	GetClusterFromInformerCache(namespace, name string) (metav1.Object, error)

	// GetPodsForCluster returns the pods managed by the cluster. This can be achieved by selecting pods using label key "kubecluster-name"
	// i.e. all pods created by the job will come with label "kubecluster-name" = <this_cluster_name>
	GetPodsForCluster(kcluster *kubeclusterorgv1alpha1.KubeCluster) ([]*v1.Pod, error)

	// GetConfigMapForCluster returns the pods managed by the cluster. This can be achieved by selecting pods using label key "kubecluster-name"
	// i.e. all pods created by the job will come with label "kubecluster-name" = <this_cluster_name>
	GetConfigMapForCluster(kcluster *kubeclusterorgv1alpha1.KubeCluster) (*v1.ConfigMap, error)

	// GetServicesForCluster returns the services managed by the cluster. This can be achieved by selecting services using label key "kubecluster-name"
	// i.e. all services created by the job will come with label "kubecluster-name" = <this_cluster_name>
	GetServicesForCluster(kcluster *kubeclusterorgv1alpha1.KubeCluster) ([]*v1.Service, error)

	// DeleteCluster deletes the cluster
	DeleteCluster(kcluster metav1.Object) error

	// UpdateClusterStatusInApiServer updates the cluster status in API server
	UpdateClusterStatusInApiServer(kcluster metav1.Object, clusterStatus *kubeclusterorgv1alpha1.ClusterStatus) error

	// UpdateClusterStatus updates the job status and job conditions
	UpdateClusterStatus(
		kcluster *kubeclusterorgv1alpha1.KubeCluster,
		replicas map[kubeclusterorgv1alpha1.ReplicaType]*kubeclusterorgv1alpha1.ReplicaSpec,
		clusterStatus *kubeclusterorgv1alpha1.ClusterStatus,
	) error

	FilterServicesForReplicaType(services []*v1.Service, rt string) ([]*v1.Service, error)

	GetServiceSlices(services []*v1.Service, replicas int, replica *log.Entry) [][]*v1.Service

	// GetGroupNameLabelValue Returns the Group Name(value) in the labels of the job
	GetGroupNameLabelValue() string

	UpdateConfigMapInApiServer(metav1.Object, *v1.ConfigMap) error
}
