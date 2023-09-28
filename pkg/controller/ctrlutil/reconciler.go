// Copyright 2021 The Kubeflow Authors
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
// limitations under the License

package ctrlutil

import (
	"fmt"
	kubeclusterorgv1alpha1 "github.com/chriskery/kubecluster/apis/kubecluster.org/v1alpha1"
	common2 "github.com/chriskery/kubecluster/pkg/common"
	"github.com/chriskery/kubecluster/pkg/controller/cluster_schema"
	"github.com/chriskery/kubecluster/pkg/controller/ctrlcommon"
	"github.com/chriskery/kubecluster/pkg/controller/expectation"
	"github.com/chriskery/kubecluster/pkg/util"
	"reflect"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

// SatisfiedExpectations returns true if the required adds/dels for the given mxjob have been observed.
// Add/del counts are established by the controller at sync time, and updated as controllees are observed by the controller
// manager.
func SatisfiedExpectations(exp expectation.ControllerExpectationsInterface,
	clusterKey string,
	replicaTypes []kubeclusterorgv1alpha1.ReplicaType) bool {

	expectationPreSatisfiedKey := expectation.GenPreSatisfiedKey(clusterKey)
	if !exp.PreSatisfiedExpectations(expectationPreSatisfiedKey) {
		return false
	}

	for _, rtype := range replicaTypes {
		// Check the expectations of the pods.
		expectationPodsKey := expectation.GenExpectationPodsKey(clusterKey, string(rtype))
		if !exp.SatisfiedExpectations(expectationPodsKey) {
			return false
		}
		// Check the expectations of the services.
		expectationServicesKey := expectation.GenExpectationServicesKey(clusterKey, string(rtype))
		if !exp.SatisfiedExpectations(expectationServicesKey) {
			return false
		}
	}
	return true
}

// OnDependentCreateFunc modify expectations when dependent (pod/service) creation observed.
func OnDependentCreateFunc(schemaReconcilers map[cluster_schema.ClusterSchema]common2.ClusterSchemaReconciler) func(event.CreateEvent) bool {
	return func(e event.CreateEvent) bool {
		clusterType := e.Object.GetLabels()[kubeclusterorgv1alpha1.ClusterTypeLabel]
		if len(clusterType) == 0 {
			return false
		}
		exp, ok := schemaReconcilers[cluster_schema.ClusterSchema(clusterType)]
		if !ok {
			return false
		}

		rtype := e.Object.GetLabels()[kubeclusterorgv1alpha1.ReplicaTypeLabel]
		if len(rtype) == 0 {
			return false
		}

		//logrus.Info("Update on create function ", ptjr.ControllerName(), " create object ", e.Object.GetName())
		if controllerRef := metav1.GetControllerOf(e.Object); controllerRef != nil {
			clusterKey := fmt.Sprintf("%s/%s", e.Object.GetNamespace(), controllerRef.Name)
			var expectKey string
			switch e.Object.(type) {
			case *corev1.Pod:
				expectKey = expectation.GenExpectationPodsKey(clusterKey, rtype)
			case *corev1.Service:
				expectKey = expectation.GenExpectationServicesKey(clusterKey, rtype)
			default:
				return false
			}
			exp.CreationObserved(expectKey)
			return true
		}

		return true
	}
}

// OnDependentUpdateFunc modify expectations when dependent (pod/service) update observed.
func OnDependentUpdateFunc(cc *ctrlcommon.ClusterController) func(updateEvent event.UpdateEvent) bool {
	return func(e event.UpdateEvent) bool {
		newObj := e.ObjectNew
		oldObj := e.ObjectOld
		if newObj.GetResourceVersion() == oldObj.GetResourceVersion() {
			// Periodic resync will send update events for all known pods.
			// Two different versions of the same pod will always have different RVs.
			return false
		}

		kind := cc.Controller.GetAPIGroupVersionKind().Kind
		var logger = LoggerForGenericKind(newObj, kind)

		switch obj := newObj.(type) {
		case *corev1.Pod:
			logger = util.LoggerForPod(obj, cc.Controller.GetAPIGroupVersionKind().Kind)
		case *corev1.Service:
			logger = util.LoggerForService(newObj.(*corev1.Service), cc.Controller.GetAPIGroupVersionKind().Kind)
		default:
			return false
		}

		newControllerRef := metav1.GetControllerOf(newObj)
		oldControllerRef := metav1.GetControllerOf(oldObj)
		controllerRefChanged := !reflect.DeepEqual(newControllerRef, oldControllerRef)

		if controllerRefChanged && oldControllerRef != nil {
			// The ControllerRef was changed. Sync the old controller, if any.
			if job := resolveControllerRef(cc, oldObj.GetNamespace(), oldControllerRef); job != nil {
				logger.Infof("pod/service controller ref updated: %v, %v", newObj, oldObj)
				return true
			}
		}

		// If it has a controller ref, that's all that matters.
		if newControllerRef != nil {
			job := resolveControllerRef(cc, newObj.GetNamespace(), newControllerRef)
			if job == nil {
				return false
			}
			logger.Debugf("pod/service has a controller ref: %v, %v", newObj, oldObj)
			return true
		}
		return false
	}
}

// resolveControllerRef returns the job referenced by a ControllerRef,
// or nil if the ControllerRef could not be resolved to a matching job
// of the correct Kind.
func resolveControllerRef(cc *ctrlcommon.ClusterController, namespace string, controllerRef *metav1.OwnerReference) metav1.Object {
	// We can't look up by UID, so look up by Name and then verify UID.
	// Don't even try to look up by Name if it's the wrong Kind.
	if controllerRef.Kind != cc.Controller.GetAPIGroupVersionKind().Kind {
		return nil
	}
	cluster, err := cc.Controller.GetClusterFromInformerCache(namespace, controllerRef.Name)
	if err != nil {
		return nil
	}
	if cluster.GetUID() != controllerRef.UID {
		// The controller we found with this Name is not the same one that the
		// ControllerRef points to.
		return nil
	}
	return cluster
}

// OnDependentDeleteFunc modify expectations when dependent (pod/service) deletion observed.
func OnDependentDeleteFunc(schemaReconcilers map[cluster_schema.ClusterSchema]common2.ClusterSchemaReconciler) func(event.DeleteEvent) bool {
	return func(e event.DeleteEvent) bool {
		clusterType := e.Object.GetLabels()[kubeclusterorgv1alpha1.ClusterTypeLabel]
		if len(clusterType) == 0 {
			return false
		}
		exp, ok := schemaReconcilers[cluster_schema.ClusterSchema(clusterType)]
		if !ok {
			return false
		}

		rtype := e.Object.GetLabels()[kubeclusterorgv1alpha1.ReplicaTypeLabel]
		if len(rtype) == 0 {
			return false
		}

		// logrus.Info("Update on deleting function ", xgbr.ControllerName(), " delete object ", e.Object.GetName())
		if controllerRef := metav1.GetControllerOf(e.Object); controllerRef != nil {
			clusterKey := fmt.Sprintf("%s/%s", e.Object.GetNamespace(), controllerRef.Name)
			var expectKey string
			switch e.Object.(type) {
			case *corev1.Pod:
				expectKey = expectation.GenExpectationPodsKey(clusterKey, rtype)
			case *corev1.Service:
				expectKey = expectation.GenExpectationServicesKey(clusterKey, rtype)
			default:
				return false
			}
			exp.DeletionObserved(expectKey)
			return true
		}

		return true
	}
}
