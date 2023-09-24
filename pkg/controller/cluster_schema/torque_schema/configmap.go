package torque_schema

import (
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
	TorqueConfDir       = "/etc"
	PBSConfKey          = "pbs.conf"
	PBSConf             = TorqueConfDir + "/" + PBSConfKey
	PBCMonPrivConfig    = "/var/spool/pbs/mom_priv/config"
	PBCMonPrivConfigKey = "privConfig"

	PBS_START_MOM_VALUE     = "1"
	NOT_PBS_START_MOM_VALUE = "0"
)

var (
	pbsConfTemplate = `
#修改PBS_SERVER和PBS_START_MOM
PBS_SERVER="${PBS_SERVER}"
PBS_START_MOM="${PBS_START_MOM}"
PBS_EXEC="${PBS_EXEC}"
`
	pbsMonPrivConfigTemplate = `
#修改clienthost为pbs1
clienthost "${CLIENT_HOST}"
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

	_, exists := configMap.Data[PBSConfKey]
	if !exists {
		configMap.Data[PBSConfKey] = t.genPBSConf(kcluster)
	}

	_, exists = configMap.Data[PBCMonPrivConfigKey]
	if !exists {
		configMap.Data[PBCMonPrivConfigKey] = t.genPBCMonPrivConfig(kcluster)
	}

	return nil
}

func (t *TorqueClusterSchemaReconciler) isNeedReconcileConfigMap(configMap *corev1.ConfigMap) bool {
	_, exists := configMap.Data[PBSConfKey]
	if !exists {
		return true
	}

	_, exists = configMap.Data[PBCMonPrivConfigKey]
	if !exists {
		return true
	}
	return false
}

func (s *TorqueClusterSchemaReconciler) genPBSConf(kcluster *kubeclusterorgv1alpha1.KubeCluster) string {
	serverName := common.GenGeneralName(kcluster.Name, SchemaReplicaTypeServer, strconv.Itoa(0))
	pbsConf := strings.Replace(pbsConfTemplate, placeHolderPBS_SERVER, serverName, 1)
	pbsConf = strings.Replace(pbsConf, placeHolderPBS_START_MOM, PBS_START_MOM_VALUE, 1)
	pbsConf = strings.Replace(pbsConf, placeHolderPBS_EXEC, PBSBin, 1)
	return pbsConf
}

func (s *TorqueClusterSchemaReconciler) genPBCMonPrivConfig(kcluster *kubeclusterorgv1alpha1.KubeCluster) string {
	serverName := common.GenGeneralName(kcluster.Name, SchemaReplicaTypeServer, strconv.Itoa(0))
	pbsMonPrivConfig := strings.Replace(pbsMonPrivConfigTemplate, placeHolderClientHost, serverName, 1)
	return pbsMonPrivConfig
}

func (t *TorqueClusterSchemaReconciler) UpdateConfigMap(_ *kubeclusterorgv1alpha1.KubeCluster, _ *corev1.ConfigMap) error {
	return nil
}
