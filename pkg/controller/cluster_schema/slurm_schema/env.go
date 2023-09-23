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

package slurm_schema

import (
	"fmt"
	kubeclusterorgv1alpha1 "github.com/kubecluster/apis/kubecluster.org/v1alpha1"
	"github.com/kubecluster/pkg/common"
	corev1 "k8s.io/api/core/v1"
	"strconv"
)

const (
	// EnvSlurmctlPort is the environment variable name for the number of processes per node.
	EnvSlurmctlPort = "SLURMCTL_PORT"
	// EnvSlurmdPort is the environment variable name for the number of nodes.
	EnvSlurmdPort = "SLURMD_PORT"
	// EnvControllerAddress is the environment variable name for the rank of nodes.
	EnvControllerAddress = "SLURM_CONTROLLER_ADDRESS"
	EnvTotalNodes        = "SLURM_TOTAL_NODES"
	EnvSlurmConf         = "SLURM_CONF"
)

// EnvVarGenerator is the environment variable generator interface.
type EnvVarGenerator interface {
	Generate(kcluster *kubeclusterorgv1alpha1.KubeCluster) ([]corev1.EnvVar, error)
}

func setPodEnv(
	slurmSchemaCluster *kubeclusterorgv1alpha1.KubeCluster,
	podTemplateSpec *corev1.PodTemplateSpec,
	defaultContainerName string,
	rtype kubeclusterorgv1alpha1.ReplicaType,
	index string,
	slurmctlPort, slurmdPort int,
) error {
	var env []corev1.EnvVar
	env = append(env, corev1.EnvVar{
		Name:  EnvSlurmdPort,
		Value: strconv.Itoa(slurmdPort),
	})
	env = append(env, corev1.EnvVar{
		Name:  EnvSlurmctlPort,
		Value: strconv.Itoa(slurmctlPort),
	})
	env = append(env, corev1.EnvVar{
		Name:  EnvControllerAddress,
		Value: common.GenGeneralName(slurmSchemaCluster.Name, SchemaReplicaTypeController, strconv.Itoa(0)),
	})
	env = append(env, corev1.EnvVar{
		Name: "LOCAL_NAME",
		ValueFrom: &corev1.EnvVarSource{
			FieldRef: &corev1.ObjectFieldSelector{
				FieldPath: "metadata.name",
			},
		},
	})
	env = append(env, corev1.EnvVar{
		Name: "LOCAL_IP",
		ValueFrom: &corev1.EnvVarSource{
			FieldRef: &corev1.ObjectFieldSelector{
				FieldPath: "status.podIP",
			},
		},
	})
	env = append(env, corev1.EnvVar{
		Name:  EnvSlurmConf,
		Value: fmt.Sprintf("%s/%s", SlurmConfDir, slurmConfKey),
	})
	env = append(env, corev1.EnvVar{
		Name:  "SLURM_ROLE",
		Value: string(rtype),
	})
	totalReplicas := getTotalReplicas(slurmSchemaCluster)
	env = append(env, corev1.EnvVar{
		Name:  "SLURM_WORKER_REPLICAS",
		Value: strconv.FormatInt(int64(totalReplicas-*slurmSchemaCluster.Spec.ClusterReplicaSpec[SchemaReplicaTypeController].Replicas), 10),
	})

	for i := range podTemplateSpec.Spec.InitContainers {
		if podTemplateSpec.Spec.InitContainers[i].Env == nil {
			podTemplateSpec.Spec.InitContainers[i].Env = make([]corev1.EnvVar, 0)
		}
		podTemplateSpec.Spec.InitContainers[i].Env = append(podTemplateSpec.Spec.InitContainers[i].Env, env...)
	}

	replicaSpec := slurmSchemaCluster.Spec.ClusterReplicaSpec[kubeclusterorgv1alpha1.ReplicaType(rtype)]
	resourceList := getResourceRequestList(replicaSpec)
	for i := range podTemplateSpec.Spec.Containers {
		if podTemplateSpec.Spec.Containers[i].Name != defaultContainerName {
			continue
		}
		if podTemplateSpec.Spec.Containers[i].Env == nil {
			podTemplateSpec.Spec.Containers[i].Env = make([]corev1.EnvVar, 0)
		}
		podTemplateSpec.Spec.Containers[i].Env = append(podTemplateSpec.Spec.Containers[i].Env, env...)
		podTemplateSpec.Spec.Containers[i].Env = append(
			podTemplateSpec.Spec.Containers[i].Env, corev1.EnvVar{
				Name:  "LOCAL_CPU",
				Value: resourceList.Cpu().String()})

		podTemplateSpec.Spec.Containers[i].Env = append(
			podTemplateSpec.Spec.Containers[i].Env, corev1.EnvVar{
				Name:  "LOCAL_GPU",
				Value: getGpuQuantity(resourceList).String()})

		podTemplateSpec.Spec.Containers[i].Env = append(
			podTemplateSpec.Spec.Containers[i].Env, corev1.EnvVar{
				Name:  "LOCAL_MEM",
				Value: resourceList.Memory().String()})
	}

	return nil
}

func getTotalReplicas(kcluster *kubeclusterorgv1alpha1.KubeCluster) int32 {
	kclusterReplicas := int32(0)
	for _, r := range kcluster.Spec.ClusterReplicaSpec {
		kclusterReplicas += *r.Replicas
	}
	return kclusterReplicas
}
