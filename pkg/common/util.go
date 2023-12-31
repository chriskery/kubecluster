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

package common

import (
	kubeclusterorgv1alpha1 "github.com/chriskery/kubecluster/apis/kubecluster.org/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ObjectFilterFunction func(obj metav1.Object) bool

// ConvertServiceList convert service list to service point list
func ConvertServiceList(list []corev1.Service) []*corev1.Service {
	if list == nil {
		return nil
	}
	ret := make([]*corev1.Service, 0, len(list))
	for i := range list {
		ret = append(ret, &list[i])
	}
	return ret
}

// KubeClusterControlledPodList filter pod list owned by the cluster.
func KubeClusterControlledPodList(list []corev1.Pod, cluster metav1.Object) []*corev1.Pod {
	if list == nil {
		return nil
	}
	ret := make([]*corev1.Pod, 0, len(list))
	for i := range list {
		if !metav1.IsControlledBy(&list[i], cluster) {
			continue
		}
		ret = append(ret, &list[i])
	}
	return ret
}

func GetReplicaTypes(specs map[kubeclusterorgv1alpha1.ReplicaType]*kubeclusterorgv1alpha1.ReplicaSpec) []kubeclusterorgv1alpha1.ReplicaType {
	keys := make([]kubeclusterorgv1alpha1.ReplicaType, 0, len(specs))
	for k := range specs {
		keys = append(keys, k)
	}
	return keys
}

// ConvertPodListWithFilter converts pod list to pod pointer list with ObjectFilterFunction
func ConvertPodListWithFilter(list []corev1.Pod, pass ObjectFilterFunction) []*corev1.Pod {
	if list == nil {
		return nil
	}
	ret := make([]*corev1.Pod, 0, len(list))
	for i := range list {
		obj := &list[i]
		if pass != nil {
			if pass(obj) {
				ret = append(ret, obj)
			}
		} else {
			ret = append(ret, obj)
		}
	}
	return ret
}
