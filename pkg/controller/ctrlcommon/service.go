package ctrlcommon

import (
	"fmt"
	kubeclusterorgv1alpha1 "github.com/kubecluster/apis/kubecluster.org/v1alpha1"
	"github.com/kubecluster/pkg/common"
	"github.com/kubecluster/pkg/controller/expectation"
	"github.com/kubecluster/pkg/util"
	utillabels "github.com/kubecluster/pkg/util/labels"
	miscutil "github.com/kubecluster/pkg/util/misc"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"strconv"
)

var (
	succeededServiceCreationCount = promauto.NewCounter(prometheus.CounterOpts{
		Name: "succeeded_service_creation_total",
		Help: "The total number of succeeded service creation",
	})
	failedServiceCreationCount = promauto.NewCounter(prometheus.CounterOpts{
		Name: "failed_service_creation_total",
		Help: "The total number of failed service creation",
	})
)

// CreateNewService creates a new service for the given index and type.
func (cc *ClusterController) CreateNewService(
	kcluster *kubeclusterorgv1alpha1.KubeCluster,
	rtype kubeclusterorgv1alpha1.ReplicaType,
	spec *kubeclusterorgv1alpha1.ReplicaSpec,
	index string,
	defaultContainerName string,
	expectations expectation.ControllerExpectationsInterface) error {
	clusetrKey, err := KeyFunc(kcluster)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("couldn't get key for job object %#v: %v", kcluster, err))
		return err
	}

	labels := utillabels.GenLabels(common.ControllerName, kcluster.GetName())
	utillabels.SetReplicaType(labels, utillabels.GenReplicaTypeLabel(rtype))
	utillabels.SetReplicaIndexStr(labels, index)
	utillabels.SetClusterType(labels, string(kcluster.Spec.ClusterType))

	ports, err := cc.GetPortsFromClusterSpec(spec, defaultContainerName)
	if err != nil {
		return err
	}

	service := &corev1.Service{
		Spec: corev1.ServiceSpec{
			ClusterIP: "None",
			Selector:  labels,
			Ports:     []corev1.ServicePort{},
		},
	}

	// Add service ports to headless service
	for name, port := range ports {
		svcPort := corev1.ServicePort{Name: name, Port: port}
		service.Spec.Ports = append(service.Spec.Ports, svcPort)
	}

	service.Name = common.GenGeneralName(kcluster.GetName(), rtype, index)
	service.Labels = labels
	// Create OwnerReference.
	controllerRef := cc.GenOwnerReference(kcluster)

	// Creation is expected when there is no error returned
	expectationServicesKey := expectation.GenExpectationServicesKey(clusetrKey, string(rtype))
	expectations.RaiseExpectations(expectationServicesKey, 1, 0)

	err = cc.ServiceControl.CreateServicesWithControllerRef(kcluster.GetNamespace(), service, kcluster, controllerRef)
	if err != nil && errors.IsTimeout(err) {
		// Service is created but its initialization has timed out.
		// If the initialization is successful eventually, the
		// controller will observe the creation via the informer.
		// If the initialization fails, or if the service keeps
		// uninitialized for a long time, the informer will not
		// receive any update, and the controller will create a new
		// service when the expectation expires.
		succeededServiceCreationCount.Inc()
		return nil
	} else if err != nil {
		// Since error occurred(the informer won't observe this service),
		// we decrement the expected number of creates
		// and wait until next reconciliation
		expectations.CreationObserved(expectationServicesKey)
		failedServiceCreationCount.Inc()
		return err
	}
	succeededServiceCreationCount.Inc()
	return nil
}

func (cc *ClusterController) ReconcilePods(
	kcluster *kubeclusterorgv1alpha1.KubeCluster,
	rType kubeclusterorgv1alpha1.ReplicaType,
	spec *kubeclusterorgv1alpha1.ReplicaSpec,
	pods []*corev1.Pod,
	configMap *corev1.ConfigMap) error {
	clusterKey, err := KeyFunc(kcluster)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("couldn't get key for kcluster object %#v: %v", kcluster, err))
		return err
	}
	clusterKind := cc.Controller.GetAPIGroupVersionKind().Kind
	expectationPodsKey := expectation.GenExpectationPodsKey(clusterKey, string(rType))

	// Convert ReplicaType to lower string.
	logger := util.LoggerForReplica(kcluster, (rType))
	// Get all pods for the type rt.
	pods, err = cc.FilterPodsForReplicaType(pods, utillabels.GenReplicaTypeLabel(rType))
	if err != nil {
		return err
	}
	numReplicas := int(*spec.Replicas)
	var controllerRole bool

	initializeReplicaStatuses(&kcluster.Status, rType)

	schemaReconciler := cc.GetSchemaReconciler(kcluster.Spec.ClusterType)
	// GetPodSlices will return enough information here to make decision to add/remove/update resources.
	//
	// For example, let's assume we have pods with replica-index 0, 1, 2
	// If replica is 4, return a slice with size 4. [[0],[1],[2],[]], a pod with replica-index 3 will be created.
	//
	// If replica is 1, return a slice with size 3. [[0],[1],[2]], pod with replica-index 1 and 2 are out of range and will be deleted.
	podSlices := cc.GetPodSlices(pods, numReplicas, logger)
	for index, podSlice := range podSlices {
		if len(podSlice) > 1 {
			logger.Warningf("We have too many pods for %s %d", rType, index)
		} else if len(podSlice) == 0 {
			logger.Infof("Need to create new pod: %s-%d", rType, index)

			// check if this replica is the master role
			controllerRole = schemaReconciler.IsController(kcluster.Spec.ClusterReplicaSpec, rType, index)
			err = cc.CreateNewPod(kcluster, rType, index, spec, controllerRole, kcluster.Spec.ClusterReplicaSpec, configMap)
			if err != nil {
				return err
			}
		} else {
			// Check the status of the current pod.
			pod := podSlice[0]

			// check if the index is in the valid range, if not, we should kill the pod
			if index < 0 || index >= numReplicas {
				err = cc.PodControl.DeletePod(pod.Namespace, pod.Name, kcluster)
				if err != nil {
					return err
				}
				// Deletion is expected
				schemaReconciler.RaiseExpectations(expectationPodsKey, 0, 1)
			}

			// Get the exit code of the container.
			var exitCode int32 = 0xbeef // magic number
			for _, status := range pod.Status.ContainerStatuses {
				state := status.State
				if status.Name == kubeclusterorgv1alpha1.ClusterDefaultContainerName && state.Terminated != nil {
					exitCode = state.Terminated.ExitCode
					logger.Infof("Pod: %v.%v exited with code %v", pod.Namespace, pod.Name, exitCode)
					cc.Recorder.Eventf(kcluster, corev1.EventTypeNormal, exitedWithCodeReason, "Pod: %v.%v exited with code %v", pod.Namespace, pod.Name, exitCode)
					break
				}
			}
			// Check if the pod is retryable.
			if pod.Status.Phase == corev1.PodFailed &&
				(spec.RestartPolicy == kubeclusterorgv1alpha1.RestartPolicyExitCode && miscutil.IsRetryableExitCode(exitCode) ||
					spec.RestartPolicy == kubeclusterorgv1alpha1.RestartPolicyOnFailure ||
					spec.RestartPolicy == kubeclusterorgv1alpha1.RestartPolicyAlways) {
				failedPodsCount.Inc()
				logger.Infof("Need to restart the pod: %v.%v", pod.Namespace, pod.Name)
				if err := cc.PodControl.DeletePod(pod.Namespace, pod.Name, kcluster); err != nil {
					return err
				}
				// Deletion is expected
				schemaReconciler.RaiseExpectations(expectationPodsKey, 0, 1)

				msg := fmt.Sprintf("job %s is restarting because %s replica(s) failed.",
					kcluster.GetName(), rType)
				cc.Recorder.Event(kcluster, corev1.EventTypeWarning, util.NewReason(clusterKind, util.ClusterRestartingReason), msg)
				util.UpdateClusterConditions(&kcluster.Status, kubeclusterorgv1alpha1.ClusterRestarting,
					corev1.ConditionTrue, util.NewReason(clusterKind, util.ClusterRestartingReason), msg)
				common.RestartedClustersCounterInc(kcluster.GetNamespace(), kcluster.Spec.ClusterType)
			}

			updateClusterReplicaStatuses(&kcluster.Status, rType, pod)
		}
	}
	return nil
}

func (cc *ClusterController) ReconcileServices(
	kcluster *kubeclusterorgv1alpha1.KubeCluster,
	rtype kubeclusterorgv1alpha1.ReplicaType,
	spec *kubeclusterorgv1alpha1.ReplicaSpec,
	services []*corev1.Service,
	_ *corev1.ConfigMap,
) error {
	// Convert ReplicaType to lower string.
	replicas := int(*spec.Replicas)
	// Get all services for the type rt.
	services, err := cc.Controller.FilterServicesForReplicaType(services, utillabels.GenReplicaTypeLabel(rtype))
	if err != nil {
		return err
	}

	schemaReconciler := cc.GetSchemaReconciler(kcluster.Spec.ClusterType)
	// GetServiceSlices will return enough information here to make decision to add/remove/update resources.
	//
	// For example, let's assume we have services with replica-index 0, 1, 2
	// If replica is 4, return a slice with size 4. [[0],[1],[2],[]], a svc with replica-index 3 will be created.
	//
	// If replica is 1, return a slice with size 3. [[0],[1],[2]], svc with replica-index 1 and 2 are out of range and will be deleted.
	serviceSlices := cc.Controller.GetServiceSlices(services, replicas, util.LoggerForReplica(kcluster, (rtype)))

	for index, serviceSlice := range serviceSlices {
		if len(serviceSlice) > 1 {
			util.LoggerForReplica(kcluster, rtype).Warningf("We have too many services for %s %d", rtype, index)
		} else if len(serviceSlice) == 0 {
			util.LoggerForReplica(kcluster, rtype).Infof("need to create new service: %s-%d", rtype, index)
			err = cc.CreateNewService(
				kcluster,
				rtype,
				spec,
				strconv.Itoa(index),
				schemaReconciler.GetDefaultContainerName(),
				schemaReconciler)
			if err != nil {
				return err
			}
		} else {
			// Check the status of the current svc.
			svc := serviceSlice[0]

			// check if the index is in the valid range, if not, we should kill the svc
			if index < 0 || index >= replicas {
				err = cc.ServiceControl.DeleteService(svc.Namespace, svc.Name, kcluster)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}
