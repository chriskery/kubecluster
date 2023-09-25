package torque_schema

import (
	"fmt"
	kubeclusterorgv1alpha1 "github.com/kubecluster/apis/kubecluster.org/v1alpha1"
	"github.com/kubecluster/pkg/common"
	corev1 "k8s.io/api/core/v1"
	"strconv"
	"strings"
)

const (
	placeHolderPBS_SERVER    = "${PBS_SERVER}"
	placeHolderPBS_START_MOM = "${PBS_START_MOM}"
	placeHolderClientHost    = "${CLIENT_HOST}"
	placeHolderPBS_EXEC      = "${PBS_EXEC}"
)

const (
	TorqueConfDir    = "/etc"
	PBSConfKey       = "pbs.conf"
	PBSServerConfKey = "pbs-server.conf"
	PBSWorkerConfKey = "pbs-worker.conf"

	PBSConf             = TorqueConfDir + "/" + PBSConfKey
	PBSMonPrivConfig    = "/var/spool/pbs/mom_priv/config"
	PBSMomConfigKey     = "privConfig"
	PBSMonPrivMountPath = "/tmp/var/spool/pbs/mom_priv/config"

	ServerEntrypoint          = "entrypoint.sh"
	ServerEntryPointMountPath = "/tmp/" + ServerEntrypoint

	PBS_START_MOM_VALUE     = "1"
	NOT_PBS_START_MOM_VALUE = "0"
)

var (
	pbsServerConfTemplate = `PBS_SERVER=${PBS_SERVER}
PBS_START_MOM=${PBS_START_MOM}
PBS_EXEC=${PBS_EXEC}
PBS_START_SERVER=1
PBS_START_SCHED=1
PBS_START_COMM=1
PBS_HOME=/var/spool/pbs
PBS_CORE_LIMIT=unlimited
`
	pbsWorkerConfTemplate = `PBS_SERVER=${PBS_SERVER}
PBS_START_MOM=1
PBS_EXEC=${PBS_EXEC}
PBS_START_COMM=1
PBS_HOME=/var/spool/pbs
`
	pbsMonPrivConfigTemplate = `$clienthost ${CLIENT_HOST}
`
)

func (t *TorqueClusterSchemaReconciler) ReconcileConfigMap(
	kcluster *kubeclusterorgv1alpha1.KubeCluster,
	configMap *corev1.ConfigMap,
) error {
	isNeedReconcile := t.isNeedReconcileConfigMap(configMap)
	if !isNeedReconcile {
		return nil
	}
	if configMap.Data == nil {
		configMap.Data = make(map[string]string)
	}

	_, exists := configMap.Data[PBSServerConfKey]
	if !exists {
		configMap.Data[PBSServerConfKey] = t.genServerPBSConf(kcluster)
	}

	_, exists = configMap.Data[PBSWorkerConfKey]
	if !exists {
		configMap.Data[PBSWorkerConfKey] = t.genWorkerPBSConf(kcluster)
	}

	_, exists = configMap.Data[PBSMomConfigKey]
	if !exists {
		configMap.Data[PBSMomConfigKey] = t.genPBSMomConfig(kcluster)
	}

	_, exists = configMap.Data[ServerEntrypoint]
	if !exists {
		configMap.Data[ServerEntrypoint] = t.genPBSServerEntrypoint(kcluster)
	}
	return nil
}

func (t *TorqueClusterSchemaReconciler) isNeedReconcileConfigMap(configMap *corev1.ConfigMap) bool {
	_, exists := configMap.Data[PBSConfKey]
	if !exists {
		return true
	}

	_, exists = configMap.Data[PBSMomConfigKey]
	if !exists {
		return true
	}
	return false
}

func (t *TorqueClusterSchemaReconciler) genServerPBSConf(kcluster *kubeclusterorgv1alpha1.KubeCluster) string {
	serverName := common.GenGeneralName(kcluster.Name, SchemaReplicaTypeServer, strconv.Itoa(0))
	pbsConf := strings.Replace(pbsServerConfTemplate, placeHolderPBS_SERVER, serverName, 1)
	pbsConf = strings.Replace(pbsConf, placeHolderPBS_START_MOM, PBS_START_MOM_VALUE, 1)
	pbsConf = strings.Replace(pbsConf, placeHolderPBS_EXEC, PBSExec, 1)
	return pbsConf
}

func (s *TorqueClusterSchemaReconciler) genWorkerPBSConf(kcluster *kubeclusterorgv1alpha1.KubeCluster) string {
	serverName := common.GenGeneralName(kcluster.Name, SchemaReplicaTypeServer, strconv.Itoa(0))
	pbsConf := strings.Replace(pbsWorkerConfTemplate, placeHolderPBS_SERVER, serverName, 1)
	pbsConf = strings.Replace(pbsConf, placeHolderPBS_EXEC, PBSExec, 1)
	return pbsConf
}

func (t *TorqueClusterSchemaReconciler) genPBSMomConfig(kcluster *kubeclusterorgv1alpha1.KubeCluster) string {
	serverName := common.GenGeneralName(kcluster.Name, SchemaReplicaTypeServer, strconv.Itoa(0))
	pbsMonPrivConfig := strings.Replace(pbsMonPrivConfigTemplate, placeHolderClientHost, serverName, 1)
	return pbsMonPrivConfig
}

func (t *TorqueClusterSchemaReconciler) UpdateConfigMap(_ *kubeclusterorgv1alpha1.KubeCluster, _ *corev1.ConfigMap) error {
	return nil
}

func (t *TorqueClusterSchemaReconciler) genPBSServerEntrypoint(kcluster *kubeclusterorgv1alpha1.KubeCluster) string {
	var entrypointShell = "#!/bin/bash\n"
	for replicaType, spec := range kcluster.Spec.ClusterReplicaSpec {
		totalReplica := *spec.Replicas
		for i := 0; i < int(totalReplica); i++ {
			entrypointShell = fmt.Sprintf(
				"%s\n%s",
				entrypointShell,
				fmt.Sprintf("%s/qmgr -c \"create node %s\"", PBSBin, common.GenGeneralName(kcluster.GetName(), replicaType, strconv.Itoa(i))),
			)
		}
	}
	entrypointShell = fmt.Sprintf("%s\n%s", entrypointShell, "sleep infinity")
	return entrypointShell
}
