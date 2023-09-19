package slurm

import (
	"context"
	"fmt"
	kubeclusterorgv1alpha1 "github.com/kubecluster/apis/kubecluster.org/v1alpha1"
	"github.com/kubecluster/pkg/common"
	"github.com/kubecluster/pkg/controller/expectation"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strings"
)

const (
	ClusterSchemaKind = "slurm"

	// SchemaReplicaTypeController is the type of Master of distributed Slurm Cluster
	SchemaReplicaTypeController kubeclusterorgv1alpha1.ReplicaType = "Controller"
)

func NewSlurmClusterReconciler(ctx context.Context) (common.ClusterSchemaReconciler, error) {
	return &SlurmClusterReconciler{}, nil
}

type SlurmClusterReconciler struct {
	expectation.ControllerExpectations
}

func (s *SlurmClusterReconciler) UpdateClusterStatus(kcluster metav1.Object, replicas map[kubeclusterorgv1alpha1.ReplicaType]*kubeclusterorgv1alpha1.ReplicaSpec, clusterStatus *kubeclusterorgv1alpha1.ClusterStatus) error {
	//TODO implement me
	panic("implement me")
}

func (s *SlurmClusterReconciler) ReconcileConfigMap(kcluster *kubeclusterorgv1alpha1.KubeCluster, configMap *v1.ConfigMap) error {
	//TODO implement me
	panic("implement me")
}

func (s *SlurmClusterReconciler) IsController(
	spec map[kubeclusterorgv1alpha1.ReplicaType]*kubeclusterorgv1alpha1.ReplicaSpec,
	rType kubeclusterorgv1alpha1.ReplicaType,
	index int) bool {
	return strings.ToLower(string(SchemaReplicaTypeController)) == strings.ToLower(string(rType))
}

func (s *SlurmClusterReconciler) SetClusterSpec(kcluster *kubeclusterorgv1alpha1.KubeCluster,
	template *kubeclusterorgv1alpha1.ReplicaTemplate, rt string, str string) error {
	//TODO implement me
	panic("implement me")
}

func (s *SlurmClusterReconciler) GetDefaultContainerName() string {
	return kubeclusterorgv1alpha1.ClusterDefaultContainerName
}

func (s *SlurmClusterReconciler) ValidateV1KubeCluster(kcluster *kubeclusterorgv1alpha1.KubeCluster) error {
	for replicaType, _ := range kcluster.Spec.ClusterReplicaSpec {
		if strings.ToLower(string(SchemaReplicaTypeController)) == strings.ToLower(string(replicaType)) {
			return nil
		}
	}
	return fmt.Errorf("slurm cluster need a replica named %s", SchemaReplicaTypeController)
}
