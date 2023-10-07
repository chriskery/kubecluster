// Copyright 2018 The Kubeflow Authors
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

package util

import (
	kubeclusterorgv1alpha1 "github.com/chriskery/kubecluster/apis/kubecluster.org/v1alpha1"
	"strings"

	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	metav1unstructured "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func LoggerForReplica(cluster metav1.Object, rtype kubeclusterorgv1alpha1.ReplicaType) *log.Entry {
	return log.WithFields(log.Fields{
		// We use cluster to match the key used in controller.go
		// Its more common in K8s to use a period to indicate namespace.name. So that's what we use.
		"cluster":      cluster.GetNamespace() + "." + cluster.GetName(),
		"uid":          cluster.GetUID(),
		"replica-type": rtype,
	})
}

func LoggerForCluster(cluster metav1.Object) *log.Entry {
	return log.WithFields(log.Fields{
		// We use cluster to match the key used in controller.go
		// Its more common in K8s to use a period to indicate namespace.name. So that's what we use.
		"cluster": cluster.GetNamespace() + "." + cluster.GetName(),
		"uid":     cluster.GetUID(),
	})
}

func LoggerForPod(pod *v1.Pod, kind string) *log.Entry {
	cluster := ""
	if controllerRef := metav1.GetControllerOf(pod); controllerRef != nil {
		if controllerRef.Kind == kind {
			cluster = pod.Namespace + "." + controllerRef.Name
		}
	}
	return log.WithFields(log.Fields{
		// We use cluster to match the key used in controller.go
		// In controller.go we log the key used with the workqueue.
		"cluster": cluster,
		"pod":     pod.Namespace + "." + pod.Name,
		"uid":     pod.ObjectMeta.UID,
	})
}

func LoggerForService(svc *v1.Service, kind string) *log.Entry {
	cluster := ""
	if controllerRef := metav1.GetControllerOf(svc); controllerRef != nil {
		if controllerRef.Kind == kind {
			cluster = svc.Namespace + "." + controllerRef.Name
		}
	}
	return log.WithFields(log.Fields{
		// We use cluster to match the key used in controller.go
		// In controller.go we log the key used with the workqueue.
		"cluster": cluster,
		"service": svc.Namespace + "." + svc.Name,
		"uid":     svc.ObjectMeta.UID,
	})
}

func LoggerForConfigMap(cm *v1.ConfigMap, kind string) *log.Entry {
	kcluster := ""
	if controllerRef := metav1.GetControllerOf(cm); controllerRef != nil {
		if controllerRef.Kind == kind {
			kcluster = cm.Namespace + "." + controllerRef.Name
		}
	}
	return log.WithFields(log.Fields{
		// We use cluster to match the key used in controller.go
		// In controller.go we log the key used with the workqueue.
		"kcluster":  kcluster,
		"configMap": cm.Namespace + "." + cm.Name,
		"uid":       cm.ObjectMeta.UID,
	})
}

func LoggerForKey(key string) *log.Entry {
	return log.WithFields(log.Fields{
		// The key used by the workQueue should be namespace + "/" + name.
		// Its more common in K8s to use a period to indicate namespace.name. So that's what we use.
		"cluster": strings.Replace(key, "/", ".", -1),
	})
}

func LoggerForUnstructured(obj *metav1unstructured.Unstructured, kind string) *log.Entry {
	cluster := ""
	if obj.GetKind() == kind {
		cluster = obj.GetNamespace() + "." + obj.GetName()
	}
	return log.WithFields(log.Fields{
		// We use cluster to match the key used in controller.go
		// In controller.go we log the key used with the workqueue.
		"cluster": cluster,
		"uid":     obj.GetUID(),
	})
}
