package slurm_schema

import (
	"context"
	"fmt"
	kubeclusterorgv1alpha1 "github.com/kubecluster/apis/kubecluster.org/v1alpha1"
	"github.com/kubecluster/pkg/common"
	"github.com/kubecluster/pkg/controller/expectation"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	ClusterSchemaKind = "slurm"

	// SchemaReplicaTypeController is the type of Master of distributed Slurm Cluster
	SchemaReplicaTypeController kubeclusterorgv1alpha1.ReplicaType = "Controller"

	SlurmConfDir = "/etc/slurm"

	SlurmctlPortName = "slurmctl-port"
	SlurmdPortName   = "slurmd-port"

	SlurmCmdDir = SlurmConfDir
	MungeDir    = SlurmCmdDir + "/munge"

	EmptyVolume                         = "slurm-config-dir"
	EmptyVolumeMountPathInInitContainer = "/mnt/share"
	EmptyVolumeMountPathInMainContainer = SlurmConfDir

	ConfigureShellFile = "configure.sh"
	ConfigureShellPath = EmptyVolumeMountPathInMainContainer
)

func NewSlurmClusterReconciler(_ context.Context, mgr ctrl.Manager) (common.ClusterSchemaReconciler, error) {
	return &slurmClusterSchemaReconciler{
		ControllerExpectations: *expectation.NewControllerExpectations(),
		Client:                 mgr.GetClient(),
		Scheme:                 mgr.GetScheme(),
		Recorder:               mgr.GetEventRecorderFor(common.ControllerName),
	}, nil
}

type slurmClusterSchemaReconciler struct {
	expectation.ControllerExpectations
	client.Client
	Scheme *runtime.Scheme
	// Recorder is an event recorder for recording Event resources to the
	// Kubernetes API.
	Recorder record.EventRecorder
}

func (s *slurmClusterSchemaReconciler) Default(kcluster *kubeclusterorgv1alpha1.KubeCluster) {
	// Update the key of Controller replica to camel case.
	kubeclusterorgv1alpha1.SetTypeNameToCamelCase(kcluster.Spec.ClusterReplicaSpec, SchemaReplicaTypeController)
}

func (s *slurmClusterSchemaReconciler) setSlurmPorts(kcluster *kubeclusterorgv1alpha1.KubeCluster, slurmctlPort int, slurmdPort int) {
	for _, spec := range kcluster.Spec.ClusterReplicaSpec {
		setSlurmDefaultPorts(&spec.Template.Spec, s.GetDefaultContainerName(), slurmctlPort, slurmdPort)
	}
}

func (s *slurmClusterSchemaReconciler) UpdateClusterStatus(kcluster metav1.Object, replicas map[kubeclusterorgv1alpha1.ReplicaType]*kubeclusterorgv1alpha1.ReplicaSpec, clusterStatus *kubeclusterorgv1alpha1.ClusterStatus) error {
	return nil
}

func (s *slurmClusterSchemaReconciler) IsController(
	spec map[kubeclusterorgv1alpha1.ReplicaType]*kubeclusterorgv1alpha1.ReplicaSpec,
	rType kubeclusterorgv1alpha1.ReplicaType,
	index int) bool {
	return (SchemaReplicaTypeController) == (rType)
}

func (s *slurmClusterSchemaReconciler) SetClusterSpec(
	kcluster *kubeclusterorgv1alpha1.KubeCluster,
	podTemplate *v1.PodTemplateSpec,
	rtype kubeclusterorgv1alpha1.ReplicaType,
	index string,
	configMap *v1.ConfigMap,
) error {
	slurmConf := configMap.Data[slurmConfKey]
	slurmctlPort, slurmdPort, err := getSlurmPortsFromConf(slurmConf)
	if err != nil {
		return err
	}

	if err = setPodEnv(kcluster, podTemplate, s.GetDefaultContainerName(), rtype, index, slurmctlPort, slurmdPort); err != nil {
		return err
	}
	if err = setInitContainer(kcluster, podTemplate, rtype, index, slurmctlPort, slurmdPort); err != nil {
		return err
	}
	setVolumes(podTemplate, s.GetDefaultContainerName(), configMap.Name)
	setPodNetwork(podTemplate)
	setCmd(podTemplate, s.GetDefaultContainerName(), rtype)
	setSecurity(podTemplate, s.GetDefaultContainerName(), rtype)
	return nil
}

func (s *slurmClusterSchemaReconciler) GetDefaultContainerName() string {
	return kubeclusterorgv1alpha1.ClusterDefaultContainerName
}

func (s *slurmClusterSchemaReconciler) ValidateV1KubeCluster(kcluster *kubeclusterorgv1alpha1.KubeCluster) error {
	for replicaType, _ := range kcluster.Spec.ClusterReplicaSpec {
		if SchemaReplicaTypeController == replicaType {
			return nil
		}
	}
	return fmt.Errorf("slurm cluster need a replica named %s", SchemaReplicaTypeController)
}
