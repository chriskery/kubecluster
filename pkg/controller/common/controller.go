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

package common

import (
	"fmt"
	"github.com/kubecluster/api/v1alpha1"
	"github.com/kubecluster/pkg/common"
	"github.com/kubecluster/pkg/controller"
	"github.com/kubecluster/pkg/controller/cluster_schema"
	"github.com/kubecluster/pkg/controller/control"
	"github.com/kubecluster/pkg/controller/expectation"
	"strings"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeclientset "k8s.io/client-go/kubernetes"
	corelisters "k8s.io/client-go/listers/core/v1"
	schedulinglisters "k8s.io/client-go/listers/scheduling/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/client"
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
// And then call objectB.ReconcileClusters as mentioned below, the ReconcileClusters method is the entrypoint to trigger the
// reconcile logic of the KubeCluster controller
//
// ReconcileClusters(
//
//	KubeCluster interface{},
//	replicas map[apiv1.ReplicaType]*apiv1.ReplicaSpec,
//	ClusterStatus apiv1.ClusterStatus,
//	runPolicy *apiv1.RunPolicy) error
type ClusterController struct {
	Controller common.ControllerInterface

	Config ClusterControllerConfiguration

	//TO avoid concurrent read write error
	SchemaReconcilerMutux sync.Mutex

	SchemaReconcilerManager map[controller.ClusterSchema]cluster_schema.ClusterSchemaReconciler

	// PodControl is used to add or delete pods.
	PodControl control.PodControlInterface

	// ServiceControl is used to add or delete services.
	ServiceControl control.ServiceControlInterface

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

	// A TTLCache of pod/services creates/deletes each KubeCluster expects to see
	// We use KubeCluster namespace/name + ReplicaType + pods/services as an expectation key,
	// For example, there is a TFCluster with namespace "tf-operator" and name "tfCluster-abc":
	// {
	//     "PS": {
	//         "Replicas": 2,
	//     },
	//     "Worker": {
	//         "Replicas": 4,
	//     }
	// }
	// We will create 4 expectations:
	// - "tf-operator/tfCluster-abc/ps/services", expects 2 adds.
	// - "tf-operator/tfCluster-abc/ps/pods", expects 2 adds.
	// - "tf-operator/tfCluster-abc/worker/services", expects 4 adds.
	// - "tf-operator/tfCluster-abc/worker/pods", expects 4 adds.
	Expectations expectation.ControllerExpectationsInterface

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

func (jc *ClusterController) GenOwnerReference(obj metav1.Object) *metav1.OwnerReference {
	boolPtr := func(b bool) *bool { return &b }
	controllerRef := &metav1.OwnerReference{
		APIVersion:         jc.Controller.GetAPIGroupVersion().String(),
		Kind:               jc.Controller.GetAPIGroupVersionKind().Kind,
		Name:               obj.GetName(),
		UID:                obj.GetUID(),
		BlockOwnerDeletion: boolPtr(true),
		Controller:         boolPtr(true),
	}

	return controllerRef
}

func (jc *ClusterController) GenLabels(clusterType string) map[string]string {
	clusterType = strings.Replace(clusterType, "/", "-", -1)
	return map[string]string{
		v1alpha1.ControllerNameLabel: jc.Controller.ControllerName(),
		v1alpha1.ClusterTypeLabel:    clusterType,
	}
}

// resolveControllerRef returns the KubeCluster referenced by a ControllerRef,
// or nil if the ControllerRef could not be resolved to a matching KubeCluster
// of the correct Kind.
func (jc *ClusterController) resolveControllerRef(namespace string, controllerRef *metav1.OwnerReference) metav1.Object {
	// We can't look up by UID, so look up by Name and then verify UID.
	// Don't even try to look up by Name if it's the wrong Kind.
	if controllerRef.Kind != jc.Controller.GetAPIGroupVersionKind().Kind {
		return nil
	}
	Cluster, err := jc.Controller.GetClusterFromInformerCache(namespace, controllerRef.Name)
	if err != nil {
		return nil
	}
	if Cluster.GetUID() != controllerRef.UID {
		// The controller we found with this Name is not the same one that the
		// ControllerRef points to.
		return nil
	}
	return Cluster
}

func (jc *ClusterController) RegisterSchema(schema controller.ClusterSchema, reconciler cluster_schema.ClusterSchemaReconciler) error {
	jc.SchemaReconcilerMutux.Lock()
	defer jc.SchemaReconcilerMutux.Unlock()

	_, hadRegister := jc.SchemaReconcilerManager[schema]
	if hadRegister {
		return fmt.Errorf("cluster schema %s had been registered", schema)
	}
	jc.SchemaReconcilerManager[schema] = reconciler
	return nil
}
