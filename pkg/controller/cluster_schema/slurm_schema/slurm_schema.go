package slurm_schema

import (
	"context"
	"fmt"
	kubeclusterorgv1alpha1 "github.com/kubecluster/apis/kubecluster.org/v1alpha1"
	"github.com/kubecluster/pkg/common"
	"github.com/kubecluster/pkg/controller/expectation"
	"github.com/kubecluster/pkg/util"
	corev1 "k8s.io/api/core/v1"
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

func (s *slurmClusterSchemaReconciler) UpdateClusterStatus(
	kcluster *kubeclusterorgv1alpha1.KubeCluster,
	clusterStatus *kubeclusterorgv1alpha1.ClusterStatus,
	rtype kubeclusterorgv1alpha1.ReplicaType,
	spec *kubeclusterorgv1alpha1.ReplicaSpec,
) {

	status := clusterStatus.ReplicaStatuses[rtype]
	// Generate the label selector.
	//status.Selector = metav1.FormatLabelSelector(r.GenLabelSelector(pytorchjob.Name, rtype))

	specReplicas := *spec.Replicas
	running := status.Active
	failed := status.Failed
	expected := specReplicas - running

	if rtype == SchemaReplicaTypeController {
		var msg string
		if expected == 0 {
			msg = fmt.Sprintf("KubeCLuster %s is running.", kcluster.GetName())
		} else if running > 0 {
			msg = fmt.Sprintf("KubeCLuster %s is avtivating.", kcluster.GetName())
		}
		if len(msg) != 0 {
			util.UpdateClusterConditions(
				clusterStatus,
				kubeclusterorgv1alpha1.ClusterRunning,
				corev1.ConditionTrue,
				util.NewReason(kubeclusterorgv1alpha1.KubeClusterKind, util.ClusterRunningReason),
				msg,
			)
		}
	}

	if failed > 0 {
		// For the situation that jobStatus has a restarting condition, and append a running condition,
		// the restarting condition will be removed from jobStatus by kubeflowv1.filterOutCondition(),
		// so we need to record the existing restarting condition for later use.
		var existingRestartingCondition *kubeclusterorgv1alpha1.ClusterCondition
		for _, condition := range clusterStatus.Conditions {
			if condition.Type == kubeclusterorgv1alpha1.ClusterRestarting {
				existingRestartingCondition = &kubeclusterorgv1alpha1.ClusterCondition{
					Reason:  condition.Reason,
					Message: condition.Message,
				}
			}
		}

		// For the situation that jobStatus has a restarting condition, and appends a new running condition,
		// the restarting condition will be removed from jobStatus by kubeflowv1.filterOutCondition(),
		// so we need to append the restarting condition back to jobStatus.
		if existingRestartingCondition != nil {
			util.UpdateClusterConditions(clusterStatus, kubeclusterorgv1alpha1.ClusterRestarting, corev1.ConditionTrue, existingRestartingCondition.Reason, existingRestartingCondition.Message)
			// job is restarting, no need to set it failed
			// we know it because we update the status condition when reconciling the replicas
			common.RestartedClustersCounterInc(kcluster.GetNamespace(), kcluster.Spec.ClusterType)
		} else {
			if rtype != SchemaReplicaTypeController || running != 0 {
				util.LoggerForCluster(kcluster).Infof("KubeCLuster %s/%s continues regardless %d  %s replica(s) failed .",
					kcluster.Namespace, kcluster.Name, failed, rtype)
				return
			}
			msg := fmt.Sprintf("KubeCLuster %s/%s has failed because %d %s replica(s) failed.",
				kcluster.Namespace, kcluster.Name, failed, rtype)
			s.Recorder.Event(kcluster, corev1.EventTypeNormal, util.NewReason(kubeclusterorgv1alpha1.KubeClusterKind, util.ClusterFailedReason), msg)
			if clusterStatus.CompletionTime == nil {
				now := metav1.Now()
				clusterStatus.CompletionTime = &now
			}
			util.UpdateClusterConditions(clusterStatus, kubeclusterorgv1alpha1.ClusterFailed, corev1.ConditionTrue, util.NewReason(kubeclusterorgv1alpha1.KubeClusterKind, util.ClusterFailedReason), msg)
			common.FailedClustersCounterInc(kcluster.Namespace, kcluster.Spec.ClusterType)
		}
	}
	return
}

func (s *slurmClusterSchemaReconciler) IsController(
	spec map[kubeclusterorgv1alpha1.ReplicaType]*kubeclusterorgv1alpha1.ReplicaSpec,
	rType kubeclusterorgv1alpha1.ReplicaType,
	index int) bool {
	return (SchemaReplicaTypeController) == (rType)
}

func (s *slurmClusterSchemaReconciler) SetClusterSpec(
	kcluster *kubeclusterorgv1alpha1.KubeCluster,
	podTemplate *corev1.PodTemplateSpec,
	rtype kubeclusterorgv1alpha1.ReplicaType,
	index string,
	configMap *corev1.ConfigMap,
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
