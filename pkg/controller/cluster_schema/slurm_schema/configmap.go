package slurm_schema

import (
	"context"
	"fmt"
	kubeclusterorgv1alpha1 "github.com/kubecluster/apis/kubecluster.org/v1alpha1"
	"github.com/kubecluster/pkg/common"
	utillabels "github.com/kubecluster/pkg/util/labels"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strconv"
	"strings"
)

const (
	placeHolderClusterName    = "${CLUSTER_NAME}"
	placeHolderControlMachine = "${CONTROL_MACHINE}"
	placeHolderControlAddr    = "${CONTROL_ADDRESS}"
	placeHolderSlurmctldPort  = "${SLURMCTLD_PORT}"
	placeHolderSlurmdPort     = "${slurmd_port}"
)

const (
	spoolDir = "/var/spool"
)

var (
	slurmConfTemplate = "#\n# slurm.conf file. Please run configurator.html\n# (in doc/html) to build a configuration file customized\n# for your environment.\n#\n#\n# slurm.conf file generated by configurator.html.\n# Put this file on all nodes of your cluster.\n# See the slurm.conf man page for more information.\n#\n################################################\n#                   CONTROL                    #\n################################################\n" +
		fmt.Sprintf("ClusterName=%s \n", placeHolderClusterName) +
		fmt.Sprintf("ControlMachine=%s \n", placeHolderControlMachine) +
		fmt.Sprintf("ControlAddr=%s \n", placeHolderControlAddr) +
		"\n################################################\n#            LOGGING & OTHER PATHS             #\n################################################\n" +
		"SlurmctldPidFile=/var/run/slurmctld.pid \n" +
		"SlurmdPidFile=/var/run/slurmd.pid \n" +
		"SlurmctldDebug=3 \n" +
		"SlurmdDebug=3 \n" +
		fmt.Sprintf("SlurmctldPort=%s \n", placeHolderSlurmctldPort) +
		fmt.Sprintf("SlurmdPort=%s \n", placeHolderSlurmdPort) +
		"SlurmdSpoolDir=" + spoolDir + "/slurmd \n" +
		"SlurmdUser=root \n" +
		"StateSaveLocation=" + spoolDir + " \n" +
		"\n################################################\n#           SCHEDULING & ALLOCATION            #\n################################################\n" +
		"SchedulerType=sched/backfill \n" +
		"SelectType=select/cons_tres \n" +
		"SelectTypeParameters=CR_Core_Memory \n" +
		"\n################################################\n#                    OTHER                     #\n################################################\n" +
		"MpiDefault=none \n" +
		"SwitchType=switch/none \n" +
		"ProctrackType=proctrack/pgid \n" +
		"ReturnToService=1 \n" +
		"TaskPlugin=task/none \n" +
		"################################################\n#                  ACCOUNTING                  #\n################################################\n" +
		"AccountingStorageType=accounting_storage/none \n" +
		"JobAcctGatherType=jobacct_gather/none \n" +
		"GresTypes=gpu \n"
	resourceTypeGpu = "gpu"
)

func (s *slurmClusterSchemaReconciler) getHostNetWorkNodeList(kcluster *kubeclusterorgv1alpha1.KubeCluster) []string {
	var hostNetWorkNodeList []string
	for replicaType, spec := range kcluster.Spec.ClusterReplicaSpec {
		if !isHostNetWork(spec) || spec.Replicas == nil {
			continue
		} else {
			for i := 0; i < int(*spec.Replicas); i++ {
				hostNetWorkNodeList = append(hostNetWorkNodeList, common.GenGeneralName(kcluster.Name, replicaType, strconv.Itoa(i)))
			}
		}
	}
	return hostNetWorkNodeList
}

func (s *slurmClusterSchemaReconciler) ReconcileConfigMap(kcluster *kubeclusterorgv1alpha1.KubeCluster, configMap *corev1.ConfigMap) error {
	slurmConf, exists := configMap.Data[slurmConfKey]
	if exists {
		slurmctlPort, slurmdPort, err := getSlurmPortsFromConf(slurmConf)
		if err != nil {
			return err
		}
		s.setSlurmPorts(kcluster, slurmctlPort, slurmdPort)
	}

	needReconcile := s.needReconcileConfigMap(configMap)
	if exists && !needReconcile {
		return nil
	}

	if configMap.Data == nil {
		configMap.Data = make(map[string]string)
	}

	slurmctlPort, slurmdPort, err := getSlurmPortsFromSpec(kcluster, s.GetDefaultContainerName())
	if err != nil {
		slurmctlPort, slurmdPort = genSlurmRandomPort()
		s.Recorder.Eventf(kcluster, corev1.EventTypeNormal, "SlurmPortsNotSet", fmt.Sprintf("Will use defaul slurmctld's port:%d, slurmd's port : %d", slurmctlPort, slurmdPort))
	}
	s.setSlurmPorts(kcluster, slurmctlPort, slurmdPort)

	slurmConf, err = s.createSlurmConf(kcluster, slurmctlPort, slurmdPort)
	if err != nil {
		return fmt.Errorf("generateSlurmConfig err:%v", err)
	}
	configMap.Data[slurmConfKey] = slurmConf

	hostNetWorkNodeList := s.getHostNetWorkNodeList(kcluster)
	if len(hostNetWorkNodeList) == 0 {
		configMap.Data[configMapReadyKey] = "true"
	} else {
		configMap.Data[configMapReadyKey] = "false"
		configMap.Data[hostNetWorkNodesListKey] = strings.Join(hostNetWorkNodeList, ",")
	}

	return nil
}

func (s *slurmClusterSchemaReconciler) needReconcileConfigMap(configMap *corev1.ConfigMap) bool {
	cmReady, exists := configMap.Data[configMapReadyKey]
	if exists || cmReady == "true" {
		return false
	} else {
		return true
	}
}

func (s *slurmClusterSchemaReconciler) createSlurmConfPartitions(kcluster *kubeclusterorgv1alpha1.KubeCluster) string {
	partitions := "################################################\n#                  PARTITIONS                  #\n################################################\n"
	partitions += "PartitionName=all Nodes=ALL Default=YES MaxTime=INFINITE State=UP\n"
	return partitions
}

func (s *slurmClusterSchemaReconciler) createSlurmConfNodesByReplica(
	clusterName string,
	replicaType kubeclusterorgv1alpha1.ReplicaType,
	spec *kubeclusterorgv1alpha1.ReplicaSpec,
) (string, error) {
	var tempConfig string
	if spec.Replicas == nil {
		return "", nil
	}
	replicaResourceList := getResourceRequestList(spec)
	cpus, realMemory, gpus, err := getResourceRequest(replicaResourceList)
	if err != nil {
		return "", err
	}
	for i := 0; i < int(*spec.Replicas); i++ {
		generalName := common.GenGeneralName(clusterName, replicaType, strconv.Itoa(i))
		tempConfig += "NodeName=" + generalName + " " +
			"NodeAddr=" + generalName + " " +
			"CPUs=" + strconv.FormatInt(cpus, 10) + " " +
			"RealMemory=" + strconv.FormatInt(realMemory, 10) + " " +
			//TODO(Chris):Add Gpu Type
			"Gres=gpu:" + strconv.FormatInt(gpus, 10) + " " +
			"State=UNKNOWN \n"
	}
	return tempConfig, nil
}

func (s *slurmClusterSchemaReconciler) createSlurmConfNodes(kcluster *kubeclusterorgv1alpha1.KubeCluster) (string, error) {
	nodeConf := strings.Builder{}
	var controllerConf, replicaConf string
	var err error
	for replicaType, replicaSpec := range kcluster.Spec.ClusterReplicaSpec {
		if replicaType == SchemaReplicaTypeController {
			//TODO:shound we need controller to participate computing?
			//controllerConf, err = s.createSlurmConfNodesByReplica(kcluster.Name, replicaType, replicaSpec)
			//if err != nil {
			//	return "", err
			//}
		} else {
			replicaConf, err = s.createSlurmConfNodesByReplica(kcluster.Name, replicaType, replicaSpec)
			if err != nil {
				return "", err
			}
			nodeConf.WriteString(replicaConf)
		}
	}
	var nodePrefix = "################################################\n#                    NODES                     #\n################################################\n"
	return fmt.Sprintf("%s%s\n%s", nodePrefix, controllerConf, nodeConf.String()), nil
}

func (s *slurmClusterSchemaReconciler) createSlurmConf(
	kcluster *kubeclusterorgv1alpha1.KubeCluster,
	masterPort int,
	workerPort int,
) (string, error) {
	tempConfig := slurmConfTemplate
	tempConfig = strings.Replace(tempConfig, placeHolderClusterName, kcluster.Name, 1)
	tempConfig = strings.Replace(tempConfig, placeHolderControlMachine,
		common.GenGeneralName(kcluster.Name, SchemaReplicaTypeController, strconv.Itoa(0)), 1)
	tempConfig = strings.Replace(tempConfig, placeHolderControlAddr,
		common.GenGeneralName(kcluster.Name, SchemaReplicaTypeController, strconv.Itoa(0)), 1)
	tempConfig = strings.Replace(tempConfig, placeHolderSlurmctldPort, strconv.Itoa(masterPort), 1)
	tempConfig = strings.Replace(tempConfig, placeHolderSlurmdPort, strconv.Itoa(workerPort), 1)

	nodesConf, err := s.createSlurmConfNodes(kcluster)
	if err != nil {
		return "", err
	}
	partitionsConf := s.createSlurmConfPartitions(kcluster)

	return fmt.Sprintf("%s\n%s\n%s\n", tempConfig, nodesConf, partitionsConf), nil
}

const (
	slurmConfKey            = "slurm.conf"
	configMapReadyKey       = "ready"
	hostNetWorkNodesListKey = "hostNetWorkNodesList"
)

func (s *slurmClusterSchemaReconciler) UpdateConfigMap(kcluster *kubeclusterorgv1alpha1.KubeCluster, configMap *corev1.ConfigMap) error {
	cmReady, exist := configMap.Data[configMapReadyKey]
	if exist || cmReady == "true" {
		return nil
	}

	hostNetWorkNodesList := configMap.Data[hostNetWorkNodesListKey]
	if len(hostNetWorkNodesList) == 0 {
		configMap.Data[configMapReadyKey] = "true"
		return nil
	}

	var filter common.ObjectFilterFunction = func(obj metav1.Object) bool {
		return metav1.IsControlledBy(obj, kcluster)
	}
	hostNetWorkNodesListSet := sets.New[string](strings.Split(hostNetWorkNodesList, ",")...)
	slurmConf := configMap.Data[slurmConfKey]
	for replicaType, spec := range kcluster.Spec.ClusterReplicaSpec {
		if hostNetWorkNodesListSet.Len() == 0 {
			break
		}
		if !isHostNetWork(spec) {
			continue
		}

		labels := utillabels.GenLabels(common.ControllerName, kcluster.GetName())
		utillabels.SetReplicaType(labels, string(replicaType))
		utillabels.SetClusterType(labels, string(kcluster.Spec.ClusterType))
		// Create selector.
		selector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
			MatchLabels: labels,
		})
		if err != nil {
			return fmt.Errorf("couldn't convert Cluster selector: %v", err)
		}
		// List all pods to include those that don't match the selector anymore
		// but have a ControllerRef pointing to this controller.
		podlist := &corev1.PodList{}
		err = s.List(context.Background(), podlist,
			client.MatchingLabelsSelector{Selector: selector}, client.InNamespace(kcluster.GetNamespace()))
		if err != nil {
			return err
		}

		replicaPodList := common.ConvertPodListWithFilter(podlist.Items, filter)
		for _, pod := range replicaPodList {
			if !hostNetWorkNodesListSet.Has(pod.GetName()) {
				continue
			}
			if len(pod.Spec.NodeName) == 0 || len(pod.Spec.Hostname) == 0 {
				continue
			}
			hostName := pod.Spec.Hostname
			if len(hostName) == 0 {
				hostName = pod.Spec.NodeName
			}
			strings.Replace(slurmConf, "NodeName="+pod.GetName(), "NodeName="+hostName, 1)
			hostNetWorkNodesListSet.Delete(pod.GetName())
		}
	}

	configMap.Data[slurmConfKey] = slurmConf
	if hostNetWorkNodesListSet.Len() == 0 {
		configMap.Data[hostNetWorkNodesListKey] = ""
		configMap.Data[configMapReadyKey] = "true"
	} else {
		configMap.Data[hostNetWorkNodesListKey] = strings.Join(hostNetWorkNodesListSet.UnsortedList(), ",")
		configMap.Data[configMapReadyKey] = "false"
	}
	return nil
}