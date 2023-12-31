package slurm_schema

import (
	"errors"
	"fmt"
	kubeclusterorgv1alpha1 "github.com/chriskery/kubecluster/apis/kubecluster.org/v1alpha1"
	"github.com/chriskery/kubecluster/pkg/util/quota"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/klog/v2"
	"math/rand"
	"strconv"
	"strings"
	"time"
)

func GenRandomInt31n(min, max int) int {
	return rand.New(rand.NewSource(time.Now().UnixNano())).Intn(max-min+1) + (min)
}

func genSlurmRandomPort() (int, int) {
	slurmctlPort := GenRandomInt31n(10000, 20000)
	var slurmdPort int
	for {
		slurmdPort = GenRandomInt31n(10000, 20000)
		if slurmctlPort != slurmdPort {
			break
		}
	}
	return slurmctlPort, slurmdPort
}

func setSlurmDefaultPorts(spec *corev1.PodSpec, defaultContainerName string, slurmctlPort int, slurmdPort int) {
	index := kubeclusterorgv1alpha1.GetDefaultContainerIndex(spec, defaultContainerName)
	if ok := kubeclusterorgv1alpha1.HasDefaultPort(spec, index, SlurmctlPortName); !ok {
		kubeclusterorgv1alpha1.SetDefaultPort(spec, SlurmctlPortName, int32(slurmctlPort), index)
	}
	if ok := kubeclusterorgv1alpha1.HasDefaultPort(spec, index, SlurmdPortName); !ok {
		kubeclusterorgv1alpha1.SetDefaultPort(spec, SlurmdPortName, int32(slurmdPort), index)
	}
}

func getSlurmPortsFromSpec(kcluster *kubeclusterorgv1alpha1.KubeCluster, defaultContainerName string) (int, int, error) {
	for _, spec := range kcluster.Spec.ClusterReplicaSpec {
		slurmctlPort, slurmdPort := getSlurmDefaultPort(&spec.Template.Spec, defaultContainerName)
		if slurmdPort != 0 && slurmctlPort != 0 {
			return slurmctlPort, slurmdPort, nil
		}
	}
	return 0, 0, errors.New("can not find slurm ports for kclusetr")
}

func getSlurmDefaultPort(spec *corev1.PodSpec, defaultContainerName string) (int, int) {
	var slurmctlPort, slurmdPort int
	defaultContainerIndex := kubeclusterorgv1alpha1.GetDefaultContainerIndex(spec, defaultContainerName)
	for _, containerPort := range spec.Containers[defaultContainerIndex].Ports {
		if containerPort.Name == SlurmctlPortName {
			slurmctlPort = int(containerPort.ContainerPort)
		} else if containerPort.Name == SlurmdPortName {
			slurmdPort = int(containerPort.ContainerPort)
		}
		if slurmdPort != 0 && slurmctlPort != 0 {
			return slurmctlPort, slurmdPort
		}
	}
	return slurmctlPort, slurmdPort
}

func getResourceRequest(masterResourceList corev1.ResourceList) (int64, int64, int64, error) {
	var cpus, realMemory, gpus int64
	if masterResourceList == nil {
		return 0, 0, 0, fmt.Errorf("container resource request(cpu or memory or gpu) must be set")
	}
	var memory resource.Quantity
	if masterResourceList.Memory().String() != "0" {
		memory = resource.MustParse(masterResourceList.Memory().String())
	} else {
		memory = resource.MustParse("32Gi")
	}
	realMemory, ok := (&memory).AsInt64()
	if !ok {
		klog.Errorf("createSlurmClusterConfig exchange memory to realMemory failed")
		return 0, 0, 0, fmt.Errorf("exchange memory to realMemory failed")
	}
	//realMemory is megabytes, use 1000 to avoid realMemory over machine report
	realMemory = realMemory * 1000 / (1024 * 1024 * 1024)
	if masterResourceList.Cpu().MilliValue() != 0 {
		cpus = masterResourceList.Cpu().MilliValue() / 1000
	} else {
		cpus = 8
	}
	gpus = getGpuQuantity(masterResourceList).MilliValue() / 1000
	return cpus, realMemory, gpus, nil
}

func getGpuQuantity(resourceList corev1.ResourceList) *resource.Quantity {
	for resourceName := range resourceList {
		if strings.Contains(strings.ToLower(string(resourceName)), resourceTypeGpu) {
			if val, ok := (resourceList)[resourceName]; ok {
				return &val
			}
		}
	}
	return &resource.Quantity{Format: resource.DecimalSI}
}

// getResourceRequestList returns the requested resource of the slurmCluster Master's PodSpec
func getResourceRequestList(spec *kubeclusterorgv1alpha1.ReplicaSpec) corev1.ResourceList {
	result := corev1.ResourceList{}
	for _, container := range spec.Template.Spec.Containers {
		result = quota.Add(result, container.Resources.Requests)
	}
	// take max_resource(sum_pod, any_init_container)
	for _, container := range spec.Template.Spec.InitContainers {
		result = quota.Max(result, container.Resources.Requests)
	}
	// If Overhead is being utilized, add to the total requests for the pod
	if spec.Template.Spec.Overhead != nil {
		result = quota.Add(result, spec.Template.Spec.Overhead)
	}
	return result
}

func isHostNetWork(spec *kubeclusterorgv1alpha1.ReplicaSpec) bool {
	return spec.Template.Spec.HostNetwork
}

func getSlurmPortsFromConf(slurmConf string) (int, int, error) {
	lines := strings.Split(slurmConf, "\n")
	var slurmctlPort, slurmdPort int
	var err error
	for _, line := range lines {
		if !strings.HasPrefix(line, "SlurmctldPort=") && !strings.HasPrefix(line, "SlurmdPort=") {
			continue
		}
		if strings.HasPrefix(line, "SlurmctldPort=") {
			line = strings.TrimPrefix(line, "SlurmctldPort=")
			line = strings.TrimSpace(line)
			slurmctlPort, err = strconv.Atoi(line)
			if err != nil {
				return 0, 0, err
			}
		} else {
			line = strings.TrimPrefix(line, "SlurmdPort=")
			line = strings.TrimSpace(line)
			slurmdPort, err = strconv.Atoi(line)
			if err != nil {
				return 0, 0, err
			}
		}
		if slurmctlPort != 0 && slurmdPort != 0 {
			return slurmctlPort, slurmdPort, nil
		}
	}
	return slurmctlPort, slurmdPort, errors.New("failed to parse slurm relative ports")
}

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
			Name:      configMapName,
			MountPath: fmt.Sprintf("%s/%s", SlurmConfDir, slurmConfKey),
			SubPath:   slurmConfKey,
			ReadOnly:  true,
		})
		template.Spec.InitContainers[i].VolumeMounts = append(template.Spec.InitContainers[i].VolumeMounts, corev1.VolumeMount{
			Name:      EmptyVolume,
			MountPath: EmptyVolumeMountPathInInitContainer,
			ReadOnly:  false,
		})
		template.Spec.InitContainers[i].VolumeMounts = append(template.Spec.InitContainers[i].VolumeMounts, corev1.VolumeMount{
			Name:      configMapName,
			MountPath: configMapReadyFile,
			SubPath:   configMapReadyKey,
		})

	}
	for i := range template.Spec.Containers {
		if template.Spec.Containers[i].Name != defaultContainerName {
			continue
		}
		template.Spec.Containers[i].VolumeMounts = append(template.Spec.Containers[i].VolumeMounts, corev1.VolumeMount{
			Name:      EmptyVolume,
			MountPath: EmptyVolumeMountPathInMainContainer,
			ReadOnly:  false,
		})
		template.Spec.Containers[i].VolumeMounts = append(template.Spec.Containers[i].VolumeMounts, corev1.VolumeMount{
			Name:      configMapName,
			MountPath: fmt.Sprintf("%s/%s", SlurmConfDir, slurmConfKey),
			SubPath:   slurmConfKey,
			ReadOnly:  true,
		})
		template.Spec.Containers[i].VolumeMounts = append(template.Spec.Containers[i].VolumeMounts, corev1.VolumeMount{
			Name:      configMapName,
			MountPath: fmt.Sprintf("%s/munge/munge/%s", SlurmConfDir, mungeKey),
			SubPath:   mungeKey,
		})
		if rtype == SchemaReplicaTypeController {
			template.Spec.Containers[i].VolumeMounts = append(template.Spec.Containers[i].VolumeMounts, corev1.VolumeMount{
				Name:      configMapName,
				MountPath: EntrypointFilePath,
				SubPath:   controllerEntrypoint,
			})
		} else {
			template.Spec.Containers[i].VolumeMounts = append(template.Spec.Containers[i].VolumeMounts, corev1.VolumeMount{
				Name:      configMapName,
				MountPath: EntrypointFilePath,
				SubPath:   workerEntrypoint,
			})
		}
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

func setCmd(podTemplateSpec *corev1.PodTemplateSpec, defaultContainerName string) {
	for i := range podTemplateSpec.Spec.Containers {
		if podTemplateSpec.Spec.Containers[i].Name != defaultContainerName {
			continue
		}
		if podTemplateSpec.Spec.Containers[i].Command == nil {
			podTemplateSpec.Spec.Containers[i].Command = make([]string, 0)
		}
		podTemplateSpec.Spec.Containers[i].Command = []string{"/bin/bash", "-c", fmt.Sprintf("%s <<< %s", EntrypointFilePath, EmptyVolumeMountPathInMainContainer)}
	}
}
