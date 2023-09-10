package cluster_schema

import (
	kubeclusterorgv1alpha1 "github.com/kubecluster/apis/kubecluster.org/v1alpha1"
)

type ClusterSchemaReconciler interface {
	ReconcileCluster(cluster kubeclusterorgv1alpha1.KubeCluster) error
}
