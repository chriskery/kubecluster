// Copyright 2019 The Kubeflow Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package v1alpha1

import (
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/pointer"
)

func addKubeClusterDefaultingFuncs(scheme *runtime.Scheme) error {
	scheme.AddTypeDefaultingFunc(&KubeCluster{}, func(obj interface{}) { SetDefaults_KubeCluster(obj.(*KubeCluster)) })
	scheme.AddTypeDefaultingFunc(&KubeClusterList{}, func(obj interface{}) { SetDefaults_KubeClusterList(obj.(*KubeClusterList)) })
	return RegisterDefaults(scheme)
}

func SetDefaults_KubeCluster(kcluster *KubeCluster) {
	if (len(kcluster.Spec.MainContainer)) == 0 {
		kcluster.Spec.MainContainer = ClusterDefaultContainerName
	}
	// Set default CleanKubeNodePolicy to None when neither fields specified.
	if kcluster.Spec.RunPolicy.CleanKubeNodePolicy == nil {
		kcluster.Spec.RunPolicy.CleanKubeNodePolicy = CleanKubeNodePolicyPointer(CleanKubeNodePolicyAll)
	}

	for _, spec := range kcluster.Spec.ClusterReplicaSpec {
		setDefaultReplicas(spec, 1)
	}
}

func setDefaultReplicas(replicaSpec *ReplicaSpec, replicas int32) {
	if replicaSpec != nil && replicaSpec.Replicas == nil {
		replicaSpec.Replicas = pointer.Int32(replicas)
	}
}
func SetDefaults_KubeClusterList(in *KubeClusterList) {
	for i := range in.Items {
		a := &in.Items[i]
		SetDefaults_KubeCluster(a)
	}
}
