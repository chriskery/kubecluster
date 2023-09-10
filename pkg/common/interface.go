package common

import (
	kubeclusterorgv1alpha1 "github.com/kubecluster/apis/kubecluster.org/v1alpha1"
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

	// GetServicesForCluster returns the services managed by the cluster. This can be achieved by selecting services using label key "kubecluster-name"
	// i.e. all services created by the job will come with label "kubecluster-name" = <this_cluster_name>
	GetServicesForCluster(kcluster *kubeclusterorgv1alpha1.KubeCluster) ([]*v1.Service, error)

	// GetConfigMapsForCluster returns the cofigMaps managed by the cluster. This can be achieved by selecting configmap using label key "kubecluster-name"
	// i.e. all services created by the job will come with label "kubecluster-name" = <this_cluster_name>
	GetConfigMapsForCluster(kcluster *kubeclusterorgv1alpha1.KubeCluster) ([]*v1.ConfigMap, error)

	// DeleteCluster deletes the cluster
	DeleteCluster(kcluster metav1.Object) error

	// UpdateClusterStatusInApiServer updates the cluster status in API server
	UpdateClusterStatusInApiServer(kcluster metav1.Object, clusterStatus *kubeclusterorgv1alpha1.ClusterStatus) error

	// UpdateJobStatus updates the job status and job conditions
	UpdateClusterStatus(kcluster metav1.Object, replicas map[kubeclusterorgv1alpha1.ReplicaType]*kubeclusterorgv1alpha1.ReplicaSpec, clusterStatus *kubeclusterorgv1alpha1.ClusterStatus) error

	// ReconcilePods checks and updates pods for each given ReplicaSpec.
	// It will requeue the job in case of an error while creating/deleting pods.
	// Common implementation will be provided and User can still override this to implement their own reconcile logic
	ReconcilePods(kcluster metav1.Object, clusterStatus *kubeclusterorgv1alpha1.ClusterStatus, pods []*v1.Pod, rtype kubeclusterorgv1alpha1.ReplicaType, spec *kubeclusterorgv1alpha1.ReplicaSpec) error

	// ReconcileServices checks and updates services for each given ReplicaSpec.
	// It will requeue the job in case of an error while creating/deleting services.
	// Common implementation will be provided and User can still override this to implement their own reconcile logic
	ReconcileServices(kcluster metav1.Object, clusterStatus *kubeclusterorgv1alpha1.ClusterStatus, services []*v1.Service, rtype kubeclusterorgv1alpha1.ReplicaType, spec *kubeclusterorgv1alpha1.ReplicaSpec) error
}
