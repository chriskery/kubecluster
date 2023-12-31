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

package util

import (
	"fmt"
	kubeclusterorgv1alpha1 "github.com/chriskery/kubecluster/apis/kubecluster.org/v1alpha1"
	"github.com/chriskery/kubecluster/pkg/common"
	"github.com/chriskery/kubecluster/pkg/controller/cluster_schema"
	"github.com/chriskery/kubecluster/pkg/controller/ctrlcommon"
	"reflect"
	"strings"

	log "github.com/sirupsen/logrus"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

// GenExpectationGenericKey generates an expectation key for {Kind} of a cluster
func GenExpectationGenericKey(clusterKey string, replicaType string, pl string) string {
	return clusterKey + "/" + strings.ToLower(replicaType) + "/" + pl
}

// LoggerForGenericKind generates log entry for generic Kubernetes resource Kind
func LoggerForGenericKind(obj metav1.Object, kind string) *log.Entry {
	cluster := ""
	if controllerRef := metav1.GetControllerOf(obj); controllerRef != nil {
		if controllerRef.Kind == kind {
			cluster = obj.GetNamespace() + "." + controllerRef.Name
		}
	}
	return log.WithFields(log.Fields{
		// We use cluster to match the key used in controller.go
		// In controller.go we log the key used with the workqueue.
		"cluster": cluster,
		kind:      obj.GetNamespace() + "." + obj.GetName(),
		"uid":     obj.GetUID(),
	})
}

// OnDependentCreateFuncGeneric modify expectations when dependent (pod/service) creation observed.
func OnDependentCreateFuncGeneric(schemaReconcilers map[cluster_schema.ClusterSchema]common.ClusterSchemaReconciler) func(event.CreateEvent) bool {
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

		if controllerRef := metav1.GetControllerOf(e.Object); controllerRef != nil {
			clusterKey := fmt.Sprintf("%s/%s", e.Object.GetNamespace(), controllerRef.Name)
			var expectKey string
			pl := strings.ToLower(e.Object.GetObjectKind().GroupVersionKind().Kind) + "s"
			expectKey = GenExpectationGenericKey(clusterKey, rtype, pl)
			exp.CreationObserved(expectKey)
			return true
		}

		return true
	}
}

// OnDependentUpdateFuncGeneric modify expectations when dependent (pod/service) update observed.
func OnDependentUpdateFuncGeneric(jc *ctrlcommon.ClusterController) func(updateEvent event.UpdateEvent) bool {
	return func(e event.UpdateEvent) bool {
		newObj := e.ObjectNew
		oldObj := e.ObjectOld
		if newObj.GetResourceVersion() == oldObj.GetResourceVersion() {
			// Periodic resync will send update events for all known pods.
			// Two different versions of the same pod will always have different RVs.
			return false
		}

		kind := jc.Controller.GetAPIGroupVersionKind().Kind
		var logger = LoggerForGenericKind(newObj, kind)

		newControllerRef := metav1.GetControllerOf(newObj)
		oldControllerRef := metav1.GetControllerOf(oldObj)
		controllerRefChanged := !reflect.DeepEqual(newControllerRef, oldControllerRef)

		if controllerRefChanged && oldControllerRef != nil {
			// The ControllerRef was changed. Sync the old controller, if any.
			if cluster := resolveControllerRef(jc, oldObj.GetNamespace(), oldControllerRef); cluster != nil {
				logger.Infof("%s controller ref updated: %v, %v", kind, newObj, oldObj)
				return true
			}
		}

		// If it has a controller ref, that's all that matters.
		if newControllerRef != nil {
			cluster := resolveControllerRef(jc, newObj.GetNamespace(), newControllerRef)
			if cluster == nil {
				return false
			}
			logger.Debugf("%s has a controller ref: %v, %v", kind, newObj, oldObj)
			return true
		}
		return false
	}
}

// OnDependentDeleteFuncGeneric modify expectations when dependent (pod/service) deletion observed.
func OnDependentDeleteFuncGeneric(schemaReconcilers map[cluster_schema.ClusterSchema]common.ClusterSchemaReconciler) func(event.DeleteEvent) bool {
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

		if controllerRef := metav1.GetControllerOf(e.Object); controllerRef != nil {
			clusterKey := fmt.Sprintf("%s/%s", e.Object.GetNamespace(), controllerRef.Name)
			pl := strings.ToLower(e.Object.GetObjectKind().GroupVersionKind().Kind) + "s"
			var expectKey = GenExpectationGenericKey(clusterKey, rtype, pl)

			exp.DeletionObserved(expectKey)
			return true
		}

		return true
	}
}
