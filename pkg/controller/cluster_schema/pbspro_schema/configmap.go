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

func (t *pbsproClusterSchemaReconciler) ReconcileConfigMap(
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

func (t *pbsproClusterSchemaReconciler) isNeedReconcileConfigMap(configMap *corev1.ConfigMap) bool {
	_, exists := configMap.Data[PBSConfKey]
	if !exists {
		return true
	}

	_, exists = configMap.Data[PBSMomConfigKey]
	return !exists
}

func (t *pbsproClusterSchemaReconciler) genServerPBSConf(kcluster *kubeclusterorgv1alpha1.KubeCluster) string {
	serverName := common.GenGeneralName(kcluster.Name, SchemaReplicaTypeServer, strconv.Itoa(0))
	pbsConf := strings.Replace(pbsServerConfTemplate, placeHolderPBS_SERVER, serverName, 1)
	pbsConf = strings.Replace(pbsConf, placeHolderPBS_START_MOM, PBS_START_MOM_VALUE, 1)
	pbsConf = strings.Replace(pbsConf, placeHolderPBS_EXEC, PBSExec, 1)
	return pbsConf
}

func (s *pbsproClusterSchemaReconciler) genWorkerPBSConf(kcluster *kubeclusterorgv1alpha1.KubeCluster) string {
	serverName := common.GenGeneralName(kcluster.Name, SchemaReplicaTypeServer, strconv.Itoa(0))
	pbsConf := strings.Replace(pbsWorkerConfTemplate, placeHolderPBS_SERVER, serverName, 1)
	pbsConf = strings.Replace(pbsConf, placeHolderPBS_EXEC, PBSExec, 1)
	return pbsConf
}

func (t *pbsproClusterSchemaReconciler) genPBSMomConfig(kcluster *kubeclusterorgv1alpha1.KubeCluster) string {
	serverName := common.GenGeneralName(kcluster.Name, SchemaReplicaTypeServer, strconv.Itoa(0))
	pbsMonPrivConfig := strings.Replace(pbsMonPrivConfigTemplate, placeHolderClientHost, serverName, 1)
	return pbsMonPrivConfig
}

func (t *pbsproClusterSchemaReconciler) UpdateConfigMap(_ *kubeclusterorgv1alpha1.KubeCluster, _ *corev1.ConfigMap) error {
	return nil
}

/*
# 定义要检查的节点名称列表
node_names=("node1" "node2" "node3" "node4" "node5" "node6" "node7" "node8" "node9" "node10")

# 循环遍历节点名称列表
for node_name in "${node_names[@]}"; do

	node_exists=0

	# 循环检查节点是否存在，直到存在为止
	while [ $node_exists -eq 0 ]; do
	  # 使用catnodes+nodename命令检查节点是否存在
	  catnodes+nodename "$node_name" > /dev/null 2>&1

	  # 获取命令的返回状态码
	  return_code=$?

	  # 如果返回状态码为0，表示节点存在，设置标志为1并退出循环
	  if [ $return_code -eq 0 ]; then
	    node_exists=1
	  else
	    echo "节点 $node_name 不存在，等待5秒后重新检测..."
	    sleep 5
	  fi
	done

done
*/
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

func (t *pbsproClusterSchemaReconciler) genPBSServerEntrypoint(kcluster *kubeclusterorgv1alpha1.KubeCluster) string {
	var entrypointShell = "#!/bin/bash\n"

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
	qmgrCreateNodeCmd = strings.Replace(qmgrCreateNodeCmds, "{{.Qmgr}}", fmt.Sprintf("%s/qmgr", PBSBin), 1)
	entrypointShell = fmt.Sprintf("%s\n%s\n", entrypointShell, qmgrCreateNodeCmd)
	entrypointShell = fmt.Sprintf("%s\n%s\n", entrypointShell, "sleep infinity")
	return entrypointShell
}
