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
// limitations under the License.

package labels

import (
	"errors"
	"github.com/kubecluster/apis/kubecluster.org/v1alpha1"
	"strconv"
)

func ReplicaIndex(labels map[string]string) (int, error) {
	v, ok := labels[v1alpha1.ReplicaIndexLabel]
	if !ok {
		return 0, errors.New("replica index label not found")
	}
	return strconv.Atoi(v)
}

func SetReplicaIndex(labels map[string]string, idx int) {
	SetReplicaIndexStr(labels, strconv.Itoa(idx))
}

func SetReplicaIndexStr(labels map[string]string, idx string) {
	labels[v1alpha1.ReplicaIndexLabel] = idx
}

func ReplicaType(labels map[string]string) (v1alpha1.ReplicaType, error) {
	v, ok := labels[v1alpha1.ReplicaTypeLabel]
	if !ok {
		return "", errors.New("replica type label not found")
	}
	return v1alpha1.ReplicaType(v), nil
}

func SetReplicaType(labels map[string]string, rt string) {
	labels[v1alpha1.ReplicaTypeLabel] = rt
}

func HasKnownLabels(labels map[string]string, groupName string) bool {
	_, has := labels[v1alpha1.ControllerNameLabel]
	return has
}

func SetClusterRole(labels map[string]string, role string) {
	labels[v1alpha1.ClusterRoleLabel] = role
}

func SetClusterType(labels map[string]string, clusterType string) {
	labels[v1alpha1.ClusterTypeLabel] = clusterType
}
