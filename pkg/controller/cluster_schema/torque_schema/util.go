package torque_schema

import (
	"fmt"
	kubeclusterorgv1alpha1 "github.com/kubecluster/apis/kubecluster.org/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"strings"
)

func setPodNetwork(template *corev1.PodTemplateSpec) {
	template.Spec.DNSPolicy = corev1.DNSClusterFirstWithHostNet
}

func setVolumes(template *corev1.PodTemplateSpec, defaultContainerName string, rtype kubeclusterorgv1alpha1.ReplicaType, configMapName string) {
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
		if rtype == SchemaReplicaTypeServer {
			template.Spec.Containers[i].VolumeMounts = append(template.Spec.Containers[i].VolumeMounts, corev1.VolumeMount{
				Name:      configMapName,
				MountPath: PBSConf,
				SubPath:   PBSServerConfKey,
				ReadOnly:  false,
			})
			template.Spec.Containers[i].VolumeMounts = append(template.Spec.Containers[i].VolumeMounts, corev1.VolumeMount{
				Name:      configMapName,
				MountPath: ServerEntryPointMountPath,
				SubPath:   ServerEntrypoint,
				ReadOnly:  false,
			})
		} else {
			template.Spec.Containers[i].VolumeMounts = append(template.Spec.Containers[i].VolumeMounts, corev1.VolumeMount{
				Name:      configMapName,
				MountPath: PBSConf,
				SubPath:   PBSWorkerConfKey,
				ReadOnly:  false,
			})
		}

		template.Spec.Containers[i].VolumeMounts = append(template.Spec.Containers[i].VolumeMounts, corev1.VolumeMount{
			Name:      configMapName,
			MountPath: PBSMonPrivMountPath,
			SubPath:   PBSMomConfigKey,
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

func setCmd(_ *kubeclusterorgv1alpha1.KubeCluster, podTemplateSpec *corev1.PodTemplateSpec, defaultContainerName string, rtype kubeclusterorgv1alpha1.ReplicaType) {
	for i := range podTemplateSpec.Spec.Containers {
		if podTemplateSpec.Spec.Containers[i].Name != defaultContainerName {
			continue
		}
		if podTemplateSpec.Spec.Containers[i].Command == nil {
			podTemplateSpec.Spec.Containers[i].Command = make([]string, 0)
		}
		if rtype == SchemaReplicaTypeServer {
			podTemplateSpec.Spec.Containers[i].Command = []string{"/bin/bash", "-c", genServerCommand()}
		} else {
			podTemplateSpec.Spec.Containers[i].Command = []string{"/bin/bash", "-c", genWorkerCommand()}
		}
	}
}

func genServerCommand() string {
	cmds := getGeneralCommand()
	cmds = append(cmds, "sleep 20")
	cmds = append(cmds, fmt.Sprintf("sh %s", ServerEntryPointMountPath))
	return strings.Join(cmds, " && ")
}

func genWorkerCommand() string {
	cmds := getGeneralCommand()
	cmds = append(cmds, "sleep infinity")
	return strings.Join(cmds, " && ")
}

func getGeneralCommand() []string {
	cpMomConfigCmd := fmt.Sprintf("cp %s %s", PBSMonPrivMountPath, PBSMonPrivConfig)
	pbsSSHStartCmd := fmt.Sprintf("chmod +x %s && %s start ", PBSSH, PBSSH)
	pbsStartCmd := fmt.Sprintf("%s start ", PBSCmd)

	var cmds []string
	cmds = append(cmds, cpMomConfigCmd)
	cmds = append(cmds, pbsSSHStartCmd)
	cmds = append(cmds, pbsStartCmd)
	return cmds
}
