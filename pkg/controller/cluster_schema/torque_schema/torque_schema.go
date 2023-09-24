package torque_schema

import (
	"context"
	"fmt"
	kubeclusterorgv1alpha1 "github.com/kubecluster/apis/kubecluster.org/v1alpha1"
	"github.com/kubecluster/pkg/common"
	"github.com/kubecluster/pkg/controller/expectation"
	"github.com/kubecluster/pkg/util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	ClusterSchemaKind = "torque"

	PBSSH  = "/etc/profile.d/pbs.sh"
	PBCCmd = "/etc/init.d/pbs"

	PBSBin                                                     = "/opt/pbs/bin"
	SchemaReplicaTypeServer kubeclusterorgv1alpha1.ReplicaType = "Server"
)

func NewTorqueClusterReconciler(_ context.Context, mgr ctrl.Manager) (common.ClusterSchemaReconciler, error) {
	return &TorqueClusterSchemaReconciler{
		ControllerExpectations: *expectation.NewControllerExpectations(),
		Recorder:               mgr.GetEventRecorderFor(common.ControllerName),
	}, nil
}

type TorqueClusterSchemaReconciler struct {
	expectation.ControllerExpectations
	// Recorder is an event recorder for recording Event resources to the
	// Kubernetes API.
	Recorder record.EventRecorder
}

func (s *TorqueClusterSchemaReconciler) Default(kcluster *kubeclusterorgv1alpha1.KubeCluster) {
	// Update the key of Controller replica to camel case.
	kubeclusterorgv1alpha1.SetTypeNameToCamelCase(kcluster.Spec.ClusterReplicaSpec, SchemaReplicaTypeServer)
	for _, spec := range kcluster.Spec.ClusterReplicaSpec {
		index := kubeclusterorgv1alpha1.GetDefaultContainerIndex(&spec.Template.Spec, s.GetDefaultContainerName())
		if ok := kubeclusterorgv1alpha1.HasDefaultPort(&spec.Template.Spec, index, "ssh"); !ok {
			kubeclusterorgv1alpha1.SetDefaultPort(&spec.Template.Spec, "ssh", int32(22), index)
		}
	}
}

func (s *TorqueClusterSchemaReconciler) UpdateClusterStatus(
	kcluster *kubeclusterorgv1alpha1.KubeCluster,
	clusterStatus *kubeclusterorgv1alpha1.ClusterStatus,
	rtype kubeclusterorgv1alpha1.ReplicaType,
	spec *kubeclusterorgv1alpha1.ReplicaSpec,
) {

	status := clusterStatus.ReplicaStatuses[rtype]
	// Generate the label selector.
	//status.Selector = metav1.FormatLabelSelector(r.GenLabelSelector(pytorchjob.Name, rtype))

	running := status.Active
	failed := status.Failed

	if rtype == SchemaReplicaTypeServer {
		var msg string
		if running == 0 && failed == 0 {
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
			if rtype != SchemaReplicaTypeServer {
				util.LoggerForCluster(kcluster).Infof("KubeCLuster %s/%s continues regardless %d  %s replica(s) failed .",
					kcluster.Namespace, kcluster.Name, failed, rtype)

			} else {
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
	}
	return
}

func (s *TorqueClusterSchemaReconciler) IsController(
	spec map[kubeclusterorgv1alpha1.ReplicaType]*kubeclusterorgv1alpha1.ReplicaSpec,
	rType kubeclusterorgv1alpha1.ReplicaType,
	index int) bool {
	return (SchemaReplicaTypeServer) == (rType)
}

func (s *TorqueClusterSchemaReconciler) SetClusterSpec(
	kcluster *kubeclusterorgv1alpha1.KubeCluster,
	podTemplate *corev1.PodTemplateSpec,
	rtype kubeclusterorgv1alpha1.ReplicaType,
	index string,
	configMap *corev1.ConfigMap,
) error {

	if err := setPodEnv(kcluster, podTemplate, s.GetDefaultContainerName(), rtype, index); err != nil {
		return err
	}

	setVolumes(podTemplate, s.GetDefaultContainerName(), configMap.Name)
	setPodNetwork(podTemplate)
	setCmd(kcluster, podTemplate, s.GetDefaultContainerName(), rtype)
	setSecurity(podTemplate, s.GetDefaultContainerName(), rtype)
	return nil
}

func (s *TorqueClusterSchemaReconciler) GetDefaultContainerName() string {
	return kubeclusterorgv1alpha1.ClusterDefaultContainerName
}

func (s *TorqueClusterSchemaReconciler) ValidateV1KubeCluster(kcluster *kubeclusterorgv1alpha1.KubeCluster) error {
	for replicaType, _ := range kcluster.Spec.ClusterReplicaSpec {
		if SchemaReplicaTypeServer == replicaType {
			return nil
		}
	}
	return fmt.Errorf("torque cluster need a replica named %s", SchemaReplicaTypeServer)
}