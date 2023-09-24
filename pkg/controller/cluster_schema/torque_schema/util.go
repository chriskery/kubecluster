package torque_schema

import (
	"fmt"
	kubeclusterorgv1alpha1 "github.com/kubecluster/apis/kubecluster.org/v1alpha1"
	"github.com/kubecluster/pkg/common"
	corev1 "k8s.io/api/core/v1"
	"strconv"
	"strings"
)

func setPodNetwork(template *corev1.PodTemplateSpec) {
	template.Spec.DNSPolicy = corev1.DNSClusterFirstWithHostNet
}

func setVolumes(template *corev1.PodTemplateSpec, defaultContainerName string, configMapName string) {
	defaultConfigMapMode := int32(0777)
	template.Spec.Volumes = append(template.Spec.Volumes, corev1.Volume{
		Name: configMapName,
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: configMapName,
				},
				DefaultMode: &defaultConfigMapMode,
			},
		},
	})

	for i := range template.Spec.Containers {
		if template.Spec.Containers[i].Name != defaultContainerName {
			continue
		}
		template.Spec.Containers[i].VolumeMounts = append(template.Spec.Containers[i].VolumeMounts, corev1.VolumeMount{
			Name:      configMapName,
			MountPath: PBSConf,
			SubPath:   PBSConfKey,
			ReadOnly:  false,
		})
		template.Spec.Containers[i].VolumeMounts = append(template.Spec.Containers[i].VolumeMounts, corev1.VolumeMount{
			Name:      configMapName,
			MountPath: PBCMonPrivConfig,
			SubPath:   PBCMonPrivConfigKey,
			ReadOnly:  true,
		})
	}

}

func setSecurity(podTemplateSpec *corev1.PodTemplateSpec, defaultContainerName string, _ kubeclusterorgv1alpha1.ReplicaType) {
	for i := range podTemplateSpec.Spec.Containers {
		if podTemplateSpec.Spec.Containers[i].Name != defaultContainerName {
			continue
		}
		if podTemplateSpec.Spec.Containers[i].SecurityContext == nil {
			podTemplateSpec.Spec.Containers[i].SecurityContext = &corev1.SecurityContext{}
		}
		var runAsRoot int64 = 0
		podTemplateSpec.Spec.Containers[i].SecurityContext = &corev1.SecurityContext{RunAsUser: &runAsRoot}
	}
}

func setCmd(kcluster *kubeclusterorgv1alpha1.KubeCluster, podTemplateSpec *corev1.PodTemplateSpec, defaultContainerName string, rtype kubeclusterorgv1alpha1.ReplicaType) {
	for i := range podTemplateSpec.Spec.Containers {
		if podTemplateSpec.Spec.Containers[i].Name != defaultContainerName {
			continue
		}
		if podTemplateSpec.Spec.Containers[i].Command == nil {
			podTemplateSpec.Spec.Containers[i].Command = make([]string, 0)
		}
		if rtype == SchemaReplicaTypeServer {
			podTemplateSpec.Spec.Containers[i].Command = []string{"/bin/bash", "-c", genServerCommand(kcluster)}
		} else {
			podTemplateSpec.Spec.Containers[i].Command = []string{"/bin/bash", "-c", genWorkerCommand()}
		}
	}
}

func genServerCommand(kcluster *kubeclusterorgv1alpha1.KubeCluster) string {
	cmds := getGeneralCommand()
	for replicaType, spec := range kcluster.Spec.ClusterReplicaSpec {
		//if replicaType == SchemaReplicaTypeServer {
		//	continue
		//}
		totalReplica := *spec.Replicas
		for i := 0; i < int(totalReplica); i++ {
			cmds = append(cmds, fmt.Sprintf("%s/qmgr -c \"create node %s\"", PBSBin, common.GenGeneralName(kcluster.GetName(), replicaType, strconv.Itoa(i))))
		}
	}
	return strings.Join(cmds, " && ")
}

func genWorkerCommand() string {
	cmds := getGeneralCommand()
	return strings.Join(cmds, " && ")
}

func getGeneralCommand() []string {
	chmodCmd := fmt.Sprintf("chmod +x %s ", PBSSH)
	pbsStartCmd := fmt.Sprintf("%s start ", PBSSH)
	var cmds []string
	cmds = append(cmds, chmodCmd)
	cmds = append(cmds, pbsStartCmd)
	cmds = append(cmds, "sleep 1000")
	return cmds
}
