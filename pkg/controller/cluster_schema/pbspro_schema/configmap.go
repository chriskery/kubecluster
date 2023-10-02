package pbspro_schema

import (
	"fmt"
	kubeclusterorgv1alpha1 "github.com/chriskery/kubecluster/apis/kubecluster.org/v1alpha1"
	"github.com/chriskery/kubecluster/pkg/common"
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
	pbsproConfDir    = "/etc"
	PBSConfKey       = "pbs.conf"
	PBSServerConfKey = "pbs-server.conf"
	PBSWorkerConfKey = "pbs-worker.conf"

	PBSConf             = pbsproConfDir + "/" + PBSConfKey
	PBSMonPrivConfig    = "/var/spool/pbs/mom_priv/config"
	PBSMomConfigKey     = "privConfig"
	PBSMonPrivMountPath = "/tmp/var/spool/pbs/mom_priv/config"

	ServerEntrypoint          = "entrypoint.sh"
	WorkerEntrypoint          = "worker-entrypoint.sh"
	ServerEntryPointMountPath = "/tmp/" + ServerEntrypoint
	WorkerEntryPointMountPath = "/tmp/" + WorkerEntrypoint

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

func (p *pbsproClusterSchemaReconciler) ReconcileConfigMap(
	kcluster *kubeclusterorgv1alpha1.KubeCluster,
	configMap *corev1.ConfigMap,
) error {
	isNeedReconcile := p.isNeedReconcileConfigMap(configMap)
	if !isNeedReconcile {
		return nil
	}
	if configMap.Data == nil {
		configMap.Data = make(map[string]string)
	}

	_, exists := configMap.Data[PBSServerConfKey]
	if !exists {
		configMap.Data[PBSServerConfKey] = p.genServerPBSConf(kcluster)
	}

	_, exists = configMap.Data[PBSWorkerConfKey]
	if !exists {
		configMap.Data[PBSWorkerConfKey] = p.genWorkerPBSConf(kcluster)
	}

	_, exists = configMap.Data[PBSMomConfigKey]
	if !exists {
		configMap.Data[PBSMomConfigKey] = p.genPBSMomConfig(kcluster)
	}

	_, exists = configMap.Data[ServerEntrypoint]
	if !exists {
		configMap.Data[ServerEntrypoint] = p.genPBSServerEntrypoint(kcluster)
	}
	_, exists = configMap.Data[WorkerEntrypoint]
	if !exists {
		configMap.Data[WorkerEntrypoint] = p.genPBSWorkerEntrypoint(kcluster)
	}
	return nil
}

func (p *pbsproClusterSchemaReconciler) isNeedReconcileConfigMap(configMap *corev1.ConfigMap) bool {
	_, exists := configMap.Data[PBSConfKey]
	if !exists {
		return true
	}

	_, exists = configMap.Data[PBSMomConfigKey]
	return !exists
}

func (p *pbsproClusterSchemaReconciler) genServerPBSConf(kcluster *kubeclusterorgv1alpha1.KubeCluster) string {
	serverName := common.GenGeneralName(kcluster.Name, SchemaReplicaTypeServer, strconv.Itoa(0))
	pbsConf := strings.Replace(pbsServerConfTemplate, placeHolderPBS_SERVER, serverName, 1)
	pbsConf = strings.Replace(pbsConf, placeHolderPBS_START_MOM, PBS_START_MOM_VALUE, 1)
	pbsConf = strings.Replace(pbsConf, placeHolderPBS_EXEC, PBSExec, 1)
	return pbsConf
}

func (p *pbsproClusterSchemaReconciler) genWorkerPBSConf(kcluster *kubeclusterorgv1alpha1.KubeCluster) string {
	serverName := common.GenGeneralName(kcluster.Name, SchemaReplicaTypeServer, strconv.Itoa(0))
	pbsConf := strings.Replace(pbsWorkerConfTemplate, placeHolderPBS_SERVER, serverName, 1)
	pbsConf = strings.Replace(pbsConf, placeHolderPBS_EXEC, PBSExec, 1)
	return pbsConf
}

func (p *pbsproClusterSchemaReconciler) genPBSMomConfig(kcluster *kubeclusterorgv1alpha1.KubeCluster) string {
	serverName := common.GenGeneralName(kcluster.Name, SchemaReplicaTypeServer, strconv.Itoa(0))
	pbsMonPrivConfig := strings.Replace(pbsMonPrivConfigTemplate, placeHolderClientHost, serverName, 1)
	return pbsMonPrivConfig
}

func (p *pbsproClusterSchemaReconciler) UpdateConfigMap(_ *kubeclusterorgv1alpha1.KubeCluster, _ *corev1.ConfigMap) error {
	return nil
}

const qmgrCreateNodeCmds = `for node_name in "${node_names[@]}"; do
  node_exists=0
  
  while [ $node_exists -eq 0 ]; do
	echo "try create node $node_name"
    {{.Pbsnodes}} "$node_name" > /dev/null 2>&1

    return_code=$?
    if [ $return_code -eq 0 ]; then
      echo "create node $node_name success"
      node_exists=1
    else
	  {{.Qmgr}} -c "create node $node_name"
      sleep 5
    fi
  done
done`

func (p *pbsproClusterSchemaReconciler) genPBSServerEntrypoint(kcluster *kubeclusterorgv1alpha1.KubeCluster) string {
	var entrypointShell = "#!/bin/bash\n"
	for _, cmd := range genServerCommand() {
		entrypointShell = fmt.Sprintf("%s\n%s\n", entrypointShell, cmd)
	}

	nodeNames := make([]string, 0)
	for replicaType, spec := range kcluster.Spec.ClusterReplicaSpec {
		totalReplica := *spec.Replicas
		for i := 0; i < int(totalReplica); i++ {
			nodeName := common.GenGeneralName(kcluster.GetName(), replicaType, strconv.Itoa(i))
			nodeNames = append(nodeNames, fmt.Sprintf("\"%s\"", nodeName))
		}
	}
	entrypointShell = fmt.Sprintf("%s\n%s\n", entrypointShell, fmt.Sprintf("node_names=(%s)", strings.Join(nodeNames, " ")))
	qmgrCreateNodeCmd := strings.Replace(qmgrCreateNodeCmds, "{{.Pbsnodes}}", PBSNodes, 1)
	qmgrCreateNodeCmd = strings.Replace(qmgrCreateNodeCmd, "{{.Qmgr}}", fmt.Sprintf("%s/qmgr", PBSBin), 1)
	entrypointShell = fmt.Sprintf("%s\n%s\n", entrypointShell, qmgrCreateNodeCmd)
	entrypointShell = fmt.Sprintf("%s\n%s\n", entrypointShell, "sleep infinity")
	return entrypointShell
}

func (p *pbsproClusterSchemaReconciler) genPBSWorkerEntrypoint(_ *kubeclusterorgv1alpha1.KubeCluster) string {
	var entrypointShell = "#!/bin/bash\n"
	for _, cmd := range genWorkerCommand() {
		entrypointShell = fmt.Sprintf("%s\n%s\n", entrypointShell, cmd)
	}
	return entrypointShell
}
