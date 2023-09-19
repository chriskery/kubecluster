// Copyright 2019 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package common

import (
	"fmt"
	kubeclusterorgv1alpha1 "github.com/kubecluster/apis/kubecluster.org/v1alpha1"
	"github.com/kubecluster/pkg/controller/expectation"
	"github.com/kubecluster/pkg/core"
	"github.com/kubecluster/pkg/util"
	utillabels "github.com/kubecluster/pkg/util/labels"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"strconv"
)

const (
	// podTemplateRestartPolicyReason is the warning reason when the restart
	// policy is set in pod template.
	podTemplateRestartPolicyReason = "SettedPodTemplateRestartPolicy"
	// exitedWithCodeReason is the normal reason when the pod is exited because of the exit code.
	exitedWithCodeReason = "ExitedWithCode"
	// podTemplateSchedulerNameReason is the warning reason when other scheduler name is set
	// in pod templates with gang-scheduling enabled
	podTemplateSchedulerNameReason = "SettedPodTemplateSchedulerName"
)

var (
	// Prometheus metrics
	createdPodsCount = promauto.NewCounter(prometheus.CounterOpts{
		Name: "created_pods_total",
		Help: "The total number of created pods",
	})
	deletedPodsCount = promauto.NewCounter(prometheus.CounterOpts{
		Name: "deleted_pods_total",
		Help: "The total number of deleted pods",
	})
	failedPodsCount = promauto.NewCounter(prometheus.CounterOpts{
		Name: "failed_pods_total",
		Help: "The total number of failed pods",
	})
)

// GetPodSlices returns a slice, which element is the slice of pod.
// It gives enough information to caller to make decision to up/down scale resources.
func (cc *ClusterController) GetPodSlices(pods []*v1.Pod, replicas int, logger *log.Entry) [][]*v1.Pod {
	return core.GetPodSlices(pods, replicas, logger)
}

// CreateNewPod creates a new pod for the given index and type.
func (cc *ClusterController) CreateNewPod(kcluster *kubeclusterorgv1alpha1.KubeCluster, rt string, index int, spec *kubeclusterorgv1alpha1.ReplicaSpec, controller bool, replicas map[kubeclusterorgv1alpha1.ReplicaType]*kubeclusterorgv1alpha1.ReplicaSpec, configMap *v1.ConfigMap) error {
	jobKey, err := KeyFunc(kcluster)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("couldn't get key for kcluster object %#v: %v", kcluster, err))
		return err
	}
	logger := util.LoggerForReplica(kcluster, rt)

	// Set type and index for the worker.
	labels := cc.GenLabels(kcluster.GetName())
	utillabels.SetReplicaType(labels, rt)
	utillabels.SetReplicaIndex(labels, index)

	if controller {
		utillabels.SetClusterRole(labels, "controller")
	}

	podTemplate := spec.Template.DeepCopy()

	idxStr := strconv.Itoa(index)
	// Set name for the template.
	podTemplate.Name = GenGeneralName(kcluster.GetName(), rt, idxStr)

	if podTemplate.Labels == nil {
		podTemplate.Labels = make(map[string]string)
	}

	for key, value := range labels {
		podTemplate.Labels[key] = value
	}

	schemaReconciler := cc.GetSchemaReconciler(kcluster.Spec.ClusterType)
	if err = schemaReconciler.SetClusterSpec(kcluster, podTemplate, rt, idxStr); err != nil {
		return err
	}

	// Submit a warning event if the user specifies restart policy for
	// the pod template. We recommend to set it from the replica level.
	if podTemplate.Spec.RestartPolicy != "" {
		errMsg := "Restart policy in pod template will be overwritten by restart policy in replica spec"
		logger.Warning(errMsg)
		cc.Recorder.Event(kcluster, v1.EventTypeWarning, podTemplateRestartPolicyReason, errMsg)
	}
	core.SetRestartPolicy((*v1.PodTemplateSpec)(podTemplate), spec)

	// if gang-scheduling is enabled:
	// 1. if user has specified other scheduler, we report a warning without overriding any fields.
	// 2. if no SchedulerName is set for pods, we set the SchedulerName to gang-scheduler-name.
	if cc.Config.EnableGangScheduling() {
		if isCustomSchedulerSet(replicas, cc.PodGroupControl.GetSchedulerName()) {
			errMsg := "Another scheduler is specified when gang-scheduling is enabled and it will not be overwritten"
			logger.Warning(errMsg)
			cc.Recorder.Event(kcluster, v1.EventTypeWarning, podTemplateSchedulerNameReason, errMsg)
		}
		cc.PodGroupControl.DecoratePodTemplateSpec((*v1.PodTemplateSpec)(podTemplate), kcluster, rt)
	}

	// Creation is expected when there is no error returned
	// We use `RaiseExpectations` here to accumulate expectations since `SetExpectations` has no such kind of ability
	expectationPodsKey := expectation.GenExpectationPodsKey(jobKey, rt)
	schemaReconciler.RaiseExpectations(expectationPodsKey, 1, 0)

	controllerRef := cc.GenOwnerReference(kcluster)
	err = cc.PodControl.CreatePodsWithControllerRef(
		kcluster.GetNamespace(),
		(*v1.PodTemplateSpec)(podTemplate),
		kcluster,
		controllerRef)
	if err != nil && errors.IsTimeout(err) {
		// Pod is created but its initialization has timed out.
		// If the initialization is successful eventually, the
		// controller will observe the creation via the informer.
		// If the initialization fails, or if the pod keeps
		// uninitialized for a long time, the informer will not
		// receive any update, and the controller will create a new
		// pod when the expectation expires.
		return nil
	} else if err != nil {
		// Since error occurred(the informer won't observe this pod),
		// we decrement the expected number of creates
		// and wait until next reconciliation
		schemaReconciler.CreationObserved(expectationPodsKey)
		return err
	}
	createdPodsCount.Inc()
	return nil
}

func isCustomSchedulerSet(replicas map[kubeclusterorgv1alpha1.ReplicaType]*kubeclusterorgv1alpha1.ReplicaSpec, gangSchedulerName string) bool {
	for _, spec := range replicas {
		if spec.Template.Spec.SchedulerName != "" && spec.Template.Spec.SchedulerName != gangSchedulerName {
			return true
		}
	}
	return false
}
