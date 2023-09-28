/*
Copyright 2023 The Kubeflow Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package ctrlcommon

import (
	"fmt"
	"github.com/chriskery/kubecluster/apis/kubecluster.org/v1alpha1"
	"github.com/chriskery/kubecluster/pkg/common"
	"github.com/chriskery/kubecluster/pkg/controller/cluster_schema"
	"github.com/chriskery/kubecluster/pkg/controller/control"
	"github.com/chriskery/kubecluster/pkg/controller/expectation"
	"github.com/chriskery/kubecluster/pkg/core"
	"github.com/chriskery/kubecluster/pkg/util"
	"github.com/chriskery/kubecluster/pkg/util/k8sutil"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	kubeclientset "k8s.io/client-go/kubernetes"
	corelisters "k8s.io/client-go/listers/core/v1"
	schedulinglisters "k8s.io/client-go/listers/scheduling/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/client"
	schedulerpluginsv1alpha1 "sigs.k8s.io/scheduler-plugins/apis/scheduling/v1alpha1"
	"strings"
	"sync"
	volcanov1beta1 "volcano.sh/apis/pkg/apis/scheduling/v1beta1"
	volcanoclient "volcano.sh/apis/pkg/client/clientset/versioned"
)

var (
	// KeyFunc is the short name to DeletionHandlingMetaNamespaceKeyFunc.
	// IndexerInformer uses a delta queue, therefore for deletes we have to use this
	// key function but it should be just fine for non delete events.
	KeyFunc = cache.DeletionHandlingMetaNamespaceKeyFunc

	// Prometheus metrics
	createdPDBCount = promauto.NewCounter(prometheus.CounterOpts{
		Name: "created_pod_disruption_policies_total",
		Help: "The total number of created pod disruption policies",
	})
	deletedPDBCount = promauto.NewCounter(prometheus.CounterOpts{
		Name: "deleted_pod_disruption_policies_total",
		Help: "The total number of deleted pod disruption policies",
	})
	createdPodGroupsCount = promauto.NewCounter(prometheus.CounterOpts{
		Name: "created_pod_groups_total",
		Help: "The total number of created pod groups",
	})
	deletedPodGroupsCount = promauto.NewCounter(prometheus.CounterOpts{
		Name: "deleted_pod_groups_total",
		Help: "The total number of deleted pod groups",
	})
)

type GangScheduler string

const (
	GangSchedulerNone    GangScheduler = "None"
	GangSchedulerVolcano GangScheduler = "volcano"
	// GangSchedulerSchedulerPlugins Using this scheduler name or any scheduler name different than volcano uses the scheduler-plugins PodGroup
	GangSchedulerSchedulerPlugins GangScheduler = "scheduler-plugins"
)

// ClusterControllerConfiguration contains configuration of operator.
type ClusterControllerConfiguration struct {
	// GangScheduling choice: None, volcano and scheduler-plugins
	GangScheduling GangScheduler
}

func (c *ClusterControllerConfiguration) EnableGangScheduling() bool {
	return c.GangScheduling != "" && c.GangScheduling != GangSchedulerNone
}

// ClusterController abstracts other operators to manage the lifecycle of Clusters.
// User need to first implement the ControllerInterface(objectA) and then initialize a ClusterController(objectB) struct with objectA
// as the parameter.
type ClusterController struct {
	Controller common.ControllerInterface

	Config ClusterControllerConfiguration

	//TO avoid concurrent read write error
	SchemaReconcilerMutux sync.Mutex

	SchemaReconcilerManager map[cluster_schema.ClusterSchema]common.ClusterSchemaReconciler

	// PodControl is used to add or delete pods.
	PodControl control.PodControlInterface

	// ServiceControl is used to add or delete services.
	ServiceControl control.ServiceControlInterface

	// ConfigMapControl is used to add or delete services.
	ConfigMapControl control.ConfigMapControlInterface

	// KubeClientSet is a standard kubernetes clientset.
	KubeClientSet kubeclientset.Interface

	// PodGroupControl is used to add or delete PodGroup.
	PodGroupControl control.PodGroupControlInterface

	// PodLister can list/get pods from the shared informer's store.
	PodLister corelisters.PodLister

	// ServiceLister can list/get services from the shared informer's store.
	ServiceLister corelisters.ServiceLister

	// PriorityClassLister can list/get priorityClasses from the shared informer's store.
	PriorityClassLister schedulinglisters.PriorityClassLister

	// PodInformerSynced returns true if the pod store has been synced at least once.
	PodInformerSynced cache.InformerSynced

	// ServiceInformerSynced returns true if the service store has been synced at least once.
	ServiceInformerSynced cache.InformerSynced

	// PriorityClassInformerSynced returns true if the priority class store has been synced at least once.
	PriorityClassInformerSynced cache.InformerSynced

	// WorkQueue is a rate limited work queue. This is used to queue work to be
	// processed instead of performing it as soon as a change happens. This
	// means we can ensure we only process a fixed amount of resources at a
	// time, and makes it easy to ensure we are never processing the same item
	// simultaneously in two different workers.
	WorkQueue workqueue.RateLimitingInterface

	// Recorder is an event recorder for recording Event resources to the
	// Kubernetes API.
	Recorder record.EventRecorder
}

type GangSchedulingSetupFunc func(jc *ClusterController)

var GenVolcanoSetupFunc = func(vci volcanoclient.Interface) GangSchedulingSetupFunc {
	return func(jc *ClusterController) {
		jc.Config.GangScheduling = GangSchedulerVolcano
		jc.PodGroupControl = control.NewVolcanoControl(vci)
	}
}

var GenSchedulerPluginsSetupFunc = func(c client.Client, gangSchedulerName string) GangSchedulingSetupFunc {
	return func(jc *ClusterController) {
		jc.Config.GangScheduling = GangScheduler(gangSchedulerName)
		jc.PodGroupControl = control.NewSchedulerPluginsControl(c, gangSchedulerName)
	}
}

var GenNonGangSchedulerSetupFunc = func() GangSchedulingSetupFunc {
	return func(jc *ClusterController) {
		jc.Config.GangScheduling = ""
		jc.PodGroupControl = nil
	}
}

func (cc *ClusterController) GenOwnerReference(obj metav1.Object) *metav1.OwnerReference {
	boolPtr := func(b bool) *bool { return &b }
	controllerRef := &metav1.OwnerReference{
		APIVersion:         cc.Controller.GetAPIGroupVersion().String(),
		Kind:               cc.Controller.GetAPIGroupVersionKind().Kind,
		Name:               obj.GetName(),
		UID:                obj.GetUID(),
		BlockOwnerDeletion: boolPtr(true),
		Controller:         boolPtr(true),
	}

	return controllerRef
}

func (cc *ClusterController) GenLabels(clusterType string) map[string]string {
	clusterType = strings.Replace(clusterType, "/", "-", -1)
	return map[string]string{
		v1alpha1.ControllerNameLabel: cc.Controller.ControllerName(),
		v1alpha1.ClusterNameLabel:    clusterType,
	}
}

func (cc *ClusterController) RegisterSchema(schema cluster_schema.ClusterSchema, reconciler common.ClusterSchemaReconciler) error {
	cc.SchemaReconcilerMutux.Lock()
	defer cc.SchemaReconcilerMutux.Unlock()

	_, hadRegister := cc.SchemaReconcilerManager[schema]
	if hadRegister {
		return fmt.Errorf("cluster schema %s had been registered", schema)
	}
	cc.SchemaReconcilerManager[schema] = reconciler
	return nil
}

func (cc *ClusterController) ReconcileKubeCluster(kcluster *v1alpha1.KubeCluster, schemaReconciler common.ClusterSchemaReconciler) error {
	metaObject := metav1.Object(kcluster)
	runtimeObject := runtime.Object(kcluster)
	runPolicy := &kcluster.Spec.RunPolicy
	clusterKey, err := KeyFunc(kcluster)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("couldn't get key for kube cluster object %#v: %v", kcluster, err))
		return err
	}
	// Reset expectations
	// 1. Since `ReconcileClusters` is called, we expect that previous expectations are all satisfied,
	//    and it's safe to reset the expectations
	// 2. Reset expectations can avoid dirty data such as `expectedDeletion = -1`
	//    (pod or service was deleted unexpectedly)
	if err = cc.ResetExpectations(clusterKey, kcluster.Spec.ClusterReplicaSpec, schemaReconciler); err != nil {
		log.Warnf("Failed to reset expectations: %v", err)
	}

	log.Infof("Reconciling for kube cluster %s", metaObject.GetName())
	pods, err := cc.Controller.GetPodsForCluster(kcluster)
	if err != nil {
		log.Warnf("GetPodsForCluster error %v", err)
		return err
	}

	services, err := cc.Controller.GetServicesForCluster(kcluster)
	if err != nil {
		log.Warnf("GetServicesForCluster error %v", err)
		return err
	}

	if util.IsFinished(kcluster.Status) {
		// If the Cluster is failed, delete all pods relative resource
		if err = cc.CleanUpResources(runPolicy, runtimeObject, metaObject, kcluster.Status, pods); err != nil {
			return err
		}
		return nil
	}

	clusterName := metaObject.GetName()
	clusterKind := cc.Controller.GetAPIGroupVersionKind().Kind
	oldStatus := kcluster.Status.DeepCopy()
	if util.IsClusterSuspended(runPolicy) {
		if err = cc.CleanUpResources(runPolicy, runtimeObject, metaObject, kcluster.Status, pods); err != nil {
			return err
		}
		for rType := range kcluster.Status.ReplicaStatuses {
			kcluster.Status.ReplicaStatuses[rType].Active = 0
			kcluster.Status.ReplicaStatuses[rType].Activating = 0
		}
		msg := fmt.Sprintf("%s %s is suspended.", clusterKind, clusterName)
		if util.IsRunning(kcluster.Status) {
			util.UpdateClusterConditions(&kcluster.Status, v1alpha1.ClusterRunning, corev1.ConditionFalse, util.NewReason(clusterKind, util.ClusterSuspendedReason), msg)
		}
		// We add the suspended condition to the job only when the job doesn't have a suspended condition.
		if !util.IsSuspended(kcluster.Status) {
			util.UpdateClusterConditions(&kcluster.Status, v1alpha1.ClusterSuspended, corev1.ConditionTrue, util.NewReason(clusterKind, util.ClusterSuspendedReason), msg)
		}
		cc.Recorder.Event(runtimeObject, corev1.EventTypeNormal, util.NewReason(clusterKind, util.ClusterSuspendedReason), msg)
		if !reflect.DeepEqual(*oldStatus, kcluster.Status) {
			return cc.Controller.UpdateClusterStatusInApiServer(metaObject, &kcluster.Status)
		}
		return nil
	}

	if util.IsSuspended(kcluster.Status) {
		msg := fmt.Sprintf("%s %s is resumed.", clusterKind, clusterName)
		util.UpdateClusterConditions(&kcluster.Status, v1alpha1.ClusterSuspended, corev1.ConditionFalse, util.NewReason(clusterKind, util.ClusterSuspendedReason), msg)
		now := metav1.Now()
		kcluster.Status.StartTime = &now
		cc.Recorder.Eventf(runtimeObject, corev1.EventTypeNormal, util.NewReason(clusterKind, util.ClusterResumedReason), msg)
	}

	activePods := k8sutil.FilterActivePods(pods)

	cc.recordAbnormalPods(activePods, runtimeObject)

	active := int32(len(activePods))
	failed := k8sutil.FilterPodCount(pods, corev1.PodFailed)
	totalReplicas := k8sutil.GetTotalReplicas(kcluster.Spec.ClusterReplicaSpec)
	// retrieve the previous number of retry
	previousRetry := cc.WorkQueue.NumRequeues(clusterKey)

	var failureMessage string
	clusterExceedsLimit := false
	exceedsBackoffLimit := false
	pastBackoffLimit := false
	prevReplicasFailedNum := k8sutil.GetTotalFailedReplicas(kcluster.Status.ReplicaStatuses)

	if runPolicy.BackoffLimit != nil {
		jobHasNewFailure := failed > prevReplicasFailedNum
		// new failures happen when status does not reflect the failures and active
		// is different than parallelism, otherwise the previous controller loop
		// failed updating status so even if we pick up failure it is not a new one
		exceedsBackoffLimit = jobHasNewFailure && (active != totalReplicas) &&
			(int32(previousRetry)+1 > *runPolicy.BackoffLimit)

		pastBackoffLimit, err = cc.PastBackoffLimit(clusterName, runPolicy, kcluster.Spec.ClusterReplicaSpec, pods)
		if err != nil {
			return err
		}
	}

	if exceedsBackoffLimit || pastBackoffLimit {
		// check if the number of pod restart exceeds backoff (for restart OnFailure only)
		// OR if the number of failed jobs increased since the last syncCluster
		clusterExceedsLimit = true
		failureMessage = fmt.Sprintf("KubeCLuster %s has failed because it has reached the specified backoff limit", clusterName)
	} else if cc.PastActiveDeadline(runPolicy, kcluster.Status) {
		failureMessage = fmt.Sprintf("KubeCLuster %s has failed because it was active longer than specified deadline", clusterName)
		clusterExceedsLimit = true
	}

	if clusterExceedsLimit {
		// If the Cluster exceeds backoff limit or is past active deadline
		// delete all pods and services, then set the status to failed
		if err = cc.DeletePodAndServices(runtimeObject, runPolicy, kcluster.Status, pods); err != nil {
			return err
		}

		if err = cc.CleanupCluster(runtimeObject); err != nil {
			return err
		}

		if cc.Config.EnableGangScheduling() {
			cc.Recorder.Event(runtimeObject, corev1.EventTypeNormal, "ClusterTerminated", "Kube cluster has been terminated. Deleting PodGroup")
			if err := cc.DeletePodGroup(metaObject); err != nil {
				cc.Recorder.Eventf(runtimeObject, corev1.EventTypeWarning, "FailedDeletePodGroup", "Error deleting: %v", err)
				return err
			} else {
				cc.Recorder.Eventf(runtimeObject, corev1.EventTypeNormal, "SuccessfulDeletePodGroup", "Deleted PodGroup: %v", metaObject.GetName())
			}
		}

		cc.Recorder.Event(runtimeObject, corev1.EventTypeNormal, util.NewReason(clusterKind, util.ClusterFailedReason), failureMessage)

		util.UpdateClusterConditions(&kcluster.Status, v1alpha1.ClusterFailed, corev1.ConditionTrue, util.NewReason(clusterKind, util.ClusterFailedReason), failureMessage)

		return cc.Controller.UpdateClusterStatusInApiServer(metaObject, &kcluster.Status)
	}

	// General cases which need to reconcile
	if cc.Config.EnableGangScheduling() {
		minMember := totalReplicas
		queue := ""
		priorityClass := ""
		var schedulerTimeout *int32
		var minResources *corev1.ResourceList

		if runPolicy.SchedulingPolicy != nil {
			if minAvailable := runPolicy.SchedulingPolicy.MinAvailable; minAvailable != nil {
				minMember = *minAvailable
			}
			if q := runPolicy.SchedulingPolicy.Queue; len(q) != 0 {
				queue = q
			}
			if pc := runPolicy.SchedulingPolicy.PriorityClass; len(pc) != 0 {
				priorityClass = pc
			}
			if mr := runPolicy.SchedulingPolicy.MinResources; mr != nil {
				minResources = (*corev1.ResourceList)(mr)
			}
			if timeout := runPolicy.SchedulingPolicy.ScheduleTimeoutSeconds; timeout != nil {
				schedulerTimeout = timeout
			}
		}

		if minResources == nil {
			minResources = cc.calcPGMinResources(minMember, kcluster.Spec.ClusterReplicaSpec)
		}

		var pgSpecFill FillPodGroupSpecFunc
		switch cc.Config.GangScheduling {
		case GangSchedulerVolcano:
			pgSpecFill = func(pg metav1.Object) error {
				volcanoPodGroup, match := pg.(*volcanov1beta1.PodGroup)
				if !match {
					return fmt.Errorf("unable to recognize PodGroup: %v", klog.KObj(pg))
				}
				volcanoPodGroup.Spec = volcanov1beta1.PodGroupSpec{
					MinMember:         minMember,
					Queue:             queue,
					PriorityClassName: priorityClass,
					MinResources:      minResources,
				}
				return nil
			}
		default:
			pgSpecFill = func(pg metav1.Object) error {
				schedulerPluginsPodGroup, match := pg.(*schedulerpluginsv1alpha1.PodGroup)
				if !match {
					return fmt.Errorf("unable to recognize PodGroup: %v", klog.KObj(pg))
				}
				schedulerPluginsPodGroup.Spec = schedulerpluginsv1alpha1.PodGroupSpec{
					MinMember:              minMember,
					MinResources:           *minResources,
					ScheduleTimeoutSeconds: schedulerTimeout,
				}
				return nil
			}
		}

		syncReplicas := true
		pg, err := cc.SyncPodGroup(metaObject, pgSpecFill)
		if err != nil {
			log.Warnf("Sync PodGroup %v: %v", clusterKey, err)
			syncReplicas = false
		}

		// Delay pods creation until PodGroup status is Inqueue
		if cc.PodGroupControl.DelayPodCreationDueToPodGroup(pg) {
			log.Warnf("PodGroup %v unschedulable", clusterKey)
			syncReplicas = false
		}

		if !syncReplicas {
			now := metav1.Now()
			kcluster.Status.LastReconcileTime = &now

			// Update job status here to trigger a new reconciliation
			return cc.Controller.UpdateClusterStatusInApiServer(metaObject, &kcluster.Status)
		}
	}

	configMap, err := cc.ReconcileConfigMap(kcluster)
	if err != nil {
		log.Warnf("ReconcileServices error %v", err)
		return err
	}

	// Diff current active pods/services with replicas.
	for rtype, spec := range kcluster.Spec.ClusterReplicaSpec {
		err = cc.ReconcilePods(kcluster, rtype, spec, pods, configMap)
		if err != nil {
			log.Warnf("ReconcilePods error %v", err)
			return err
		}

		err = cc.ReconcileServices(kcluster, rtype, spec, services, configMap)
		if err != nil {
			log.Warnf("ReconcileServices error %v", err)
			return err
		}
	}

	err = cc.Controller.UpdateClusterStatus(kcluster, kcluster.Spec.ClusterReplicaSpec, &kcluster.Status)
	if err != nil {
		log.Warnf("UpdateClusterStatus error %v", err)
		return err
	}
	// No need to update the job status if the status hasn't changed since last time.
	if !reflect.DeepEqual(*oldStatus, &kcluster.Status) {
		return cc.Controller.UpdateClusterStatusInApiServer(metaObject, &kcluster.Status)
	}

	configMapDeepCopy := configMap.DeepCopy()
	err = schemaReconciler.UpdateConfigMap(kcluster, configMapDeepCopy)
	if err != nil {
		log.Warnf("UpdateClusterStatus error %v", err)
		return err
	}
	// No need to update the job status if the status hasn't changed since last time.
	if !reflect.DeepEqual(configMapDeepCopy.Data, configMap.Data) {
		return cc.Controller.UpdateConfigMapInApiServer(metaObject, configMapDeepCopy)
	}
	return nil
}

func (cc *ClusterController) calcPGMinResources(minMember int32, replicas map[v1alpha1.ReplicaType]*v1alpha1.ReplicaSpec) *corev1.ResourceList {
	return CalcPGMinResources(minMember, replicas, cc.PriorityClassLister.Get)
}

// PastActiveDeadline checks if job has ActiveDeadlineSeconds field set and if it is exceeded.
func (cc *ClusterController) PastActiveDeadline(runPolicy *v1alpha1.RunPolicy, clusterStatus v1alpha1.ClusterStatus) bool {
	return core.PastActiveDeadline(runPolicy, clusterStatus)
}

// PastBackoffLimit checks if container restartCounts sum exceeds BackoffLimit
// this method applies only to pods when restartPolicy is one of OnFailure, Always or ExitCode
func (cc *ClusterController) PastBackoffLimit(jobName string, runPolicy *v1alpha1.RunPolicy,
	replicas map[v1alpha1.ReplicaType]*v1alpha1.ReplicaSpec, pods []*corev1.Pod) (bool, error) {
	return core.PastBackoffLimit(jobName, runPolicy, replicas, pods, cc.FilterPodsForReplicaType)
}

// FilterPodsForReplicaType returns pods belong to a replicaType.
func (cc *ClusterController) FilterPodsForReplicaType(pods []*corev1.Pod, replicaType string) ([]*corev1.Pod, error) {
	return core.FilterPodsForReplicaType(pods, replicaType)
}

// recordAbnormalPods records the active pod whose latest condition is not in True status.
func (cc *ClusterController) recordAbnormalPods(activePods []*corev1.Pod, object runtime.Object) {
	core.RecordAbnormalPods(activePods, object, cc.Recorder)
}

func (cc *ClusterController) CleanUpResources(
	runPolicy *v1alpha1.RunPolicy,
	runtimeObject runtime.Object,
	metaObject metav1.Object,
	clusterStatus v1alpha1.ClusterStatus,
	pods []*corev1.Pod,
) error {
	if err := cc.DeletePodAndServices(runtimeObject, runPolicy, clusterStatus, pods); err != nil {
		return err
	}
	// ConfigMap have the same name with kube clusterName, thus the configmap could be deleted using metaObject's name.
	if err := cc.ConfigMapControl.DeleteConfigMap(metaObject.GetNamespace(), metaObject.GetName(), runtimeObject); err != nil {
		return err
	}
	if cc.Config.EnableGangScheduling() {
		cc.Recorder.Event(runtimeObject, corev1.EventTypeNormal, "ClusterTerminated", "Kube cluster has been terminated. Deleting PodGroup")
		if err := cc.DeletePodGroup(metaObject); err != nil {
			cc.Recorder.Eventf(runtimeObject, corev1.EventTypeWarning, "FailedDeletePodGroup", "Error deleting: %v", err)
			return err
		} else {
			cc.Recorder.Eventf(runtimeObject, corev1.EventTypeNormal, "SuccessfulDeletePodGroup", "Deleted PodGroup: %v", metaObject.GetName())
		}
	}
	if err := cc.CleanupCluster(runtimeObject); err != nil {
		return err
	}
	return nil
}

func (cc *ClusterController) CleanupCluster(runtimeObject runtime.Object) error {
	metaObject, _ := runtimeObject.(metav1.Object)
	err := cc.Controller.DeleteCluster(metaObject)
	if err != nil {
		log.Errorf("FailedDeleteKubeCluster: %s", err)
		cc.Recorder.Eventf(runtimeObject, corev1.EventTypeWarning, "FailedDeleteKubeCluster", err.Error())
		return err
	}
	return nil
}

// ResetExpectations reset the expectation for creates and deletes of pod/service to zero.
func (cc *ClusterController) ResetExpectations(clusterKey string, replicas map[v1alpha1.ReplicaType]*v1alpha1.ReplicaSpec,
	reconciler common.ClusterSchemaReconciler) error {
	var allErrs error
	for rtype := range replicas {
		expectationPodsKey := expectation.GenExpectationPodsKey(clusterKey, string(rtype))
		if err := reconciler.SetExpectations(expectationPodsKey, 0, 0); err != nil {
			allErrs = err
		}
		expectationServicesKey := expectation.GenExpectationServicesKey(clusterKey, string(rtype))
		if err := reconciler.SetExpectations(expectationServicesKey, 0, 0); err != nil {
			allErrs = fmt.Errorf("%s: %w", allErrs.Error(), err)
		}
	}
	return allErrs
}

func (cc *ClusterController) DeletePodAndServices(runtimeObject runtime.Object, runPolicy *v1alpha1.RunPolicy,
	clusterStatus v1alpha1.ClusterStatus, pods []*corev1.Pod) error {
	if len(pods) == 0 {
		return nil
	}

	// Delete nothing when the cleanPodPolicy is None and the job has Succeeded or Failed condition.
	if util.IsFinished(clusterStatus) && *runPolicy.CleanKubeNodePolicy == v1alpha1.CleanKubeNodePolicyNone {
		return nil
	}

	for _, pod := range pods {
		// Note that pending pod will turn into running once schedulable,
		// not cleaning it may leave orphan running pod in the future,
		// we should treat it equivalent to running phase here.
		if util.IsFinished(clusterStatus) &&
			*runPolicy.CleanKubeNodePolicy == v1alpha1.CleanKubeNodePolicyRunning &&
			pod.Status.Phase != corev1.PodRunning &&
			pod.Status.Phase != corev1.PodPending {
			continue
		}
		if err := cc.PodControl.DeletePod(pod.Namespace, pod.Name, runtimeObject); err != nil {
			return err
		}
		// Pod and service have the same name, thus the service could be deleted using pod's name.
		if err := cc.ServiceControl.DeleteService(pod.Namespace, pod.Name, runtimeObject); err != nil {
			return err
		}
		if err := cc.PodControl.DeletePod(pod.Namespace, pod.Name, runtimeObject); err != nil {
			return err
		}
	}
	return nil
}

func (cc *ClusterController) GetSchemaReconciler(clusterType v1alpha1.ClusterType) common.ClusterSchemaReconciler {
	cc.SchemaReconcilerMutux.Lock()
	defer cc.SchemaReconcilerMutux.Unlock()
	return cc.SchemaReconcilerManager[cluster_schema.ClusterSchema(clusterType)]
}

func (cc *ClusterController) GetPortsFromClusterSpec(spec *v1alpha1.ReplicaSpec, defaultContainerName string) (map[string]int32, error) {
	return core.GetPortsFromCluster(spec, defaultContainerName)
}
