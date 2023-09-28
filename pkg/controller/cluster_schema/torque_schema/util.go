package torque_schema

import (
	"fmt"
	kubeclusterorgv1alpha1 "github.com/chriskery/kubecluster/apis/kubecluster.org/v1alpha1"
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
	template.Spec.Volumes = append(template.Spec.Volumes, corev1.Volume{
		Name: EmptyVolume,
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	})

	for i := range template.Spec.InitContainers {
		template.Spec.InitContainers[i].VolumeMounts = append(template.Spec.InitContainers[i].VolumeMounts, corev1.VolumeMount{
			Name:      EmptyVolume,
			MountPath: EmptyVolumeMountPathInInitContainer,
			ReadOnly:  false,
		})
	}

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
		template.Spec.Containers[i].VolumeMounts = append(template.Spec.Containers[i].VolumeMounts, corev1.VolumeMount{
			Name:      EmptyVolume,
			MountPath: EmptyVolumeMountPathInMainContainer,
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
	serverCmds := make([]string, 0)
	serverCmds = append(serverCmds, "rm -rf /var/spool/pbs")
	cmds := getGeneralCommand()
	serverCmds = append(serverCmds, cmds...)
	serverCmds = append(serverCmds, fmt.Sprintf("sh %s", ServerEntryPointMountPath))
	return strings.Join(serverCmds, " && ")
}

func genWorkerCommand() string {
	cmds := getGeneralCommand()
	cmds = append(cmds, "sleep infinity")
	return strings.Join(cmds, " && ")
}

func getGeneralCommand() []string {
	cpPBSProCmd := fmt.Sprintf("if [ -d %s ];then cp -r -p -n %s/* %s;fi", EmptyVolumeMountPathInMainContainer, EmptyVolumeMountPathInMainContainer, "/opt")
	//cpMomConfigCmd := fmt.Sprintf("mkdir -p /var/spool/pbs/mom_priv && if [ -e %s ];then cp -n -p %s %s;fi", PBSMonPrivMountPath, PBSMonPrivMountPath, PBSMonPrivConfig)

	initCmd := fmt.Sprintf("if [ -e %s ];then sh %s;fi", PBSInitShell, PBSInitShell)

	pbsSSHStartCmd := fmt.Sprintf(" if [ -e %s ];then chmod +x %s && %s start;fi ", PBSSH, PBSSH, PBSSH)
	pbsStatustCmd := fmt.Sprintf("%s status ", PBSCmd)

	var cmds []string
	//cmds = append(cmds, "sleep 100")
	cmds = append(cmds, cpPBSProCmd, initCmd)
	cmds = append(cmds, pbsSSHStartCmd)
	cmds = append(cmds, pbsStatustCmd)
	return cmds
}
