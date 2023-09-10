package cluster_schema

import (
	"context"
	kubeclusterorgv1alpha1 "github.com/kubecluster/apis/kubecluster.org/v1alpha1"
)

const SlurmClusterKind = "slurm"

func NewSlurmClusterReconciler(ctx context.Context) (ClusterSchemaReconciler, error) {
	return &SlurmClusterReconciler{}, nil
}

type SlurmClusterReconciler struct {
}

func (s *SlurmClusterReconciler) ReconcileCluster(cluster kubeclusterorgv1alpha1.KubeCluster) error {
	//TODO implement me
	panic("implement me")
}
