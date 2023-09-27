package cluster_schema

import (
	"context"
	"fmt"
	"github.com/kubecluster/pkg/common"
	"github.com/kubecluster/pkg/controller/cluster_schema/slurm_schema"
	"github.com/kubecluster/pkg/controller/cluster_schema/torque_schema"
	ctrl "sigs.k8s.io/controller-runtime"
	"strings"
)

const ErrTemplateSchemeNotSupported = "cluster scheme %s is not supported yet"

type ClusterSchema string

type ClusterSchemaFactory func(ctx context.Context, mgr ctrl.Manager) (common.ClusterSchemaReconciler, error)

var SupportedClusterSchemaReconciler = map[ClusterSchema]ClusterSchemaFactory{
	slurm_schema.ClusterSchemaKind: func(ctx context.Context, mgr ctrl.Manager) (common.ClusterSchemaReconciler, error) {
		return slurm_schema.NewSlurmClusterReconciler(ctx, mgr)
	},
	torque_schema.ClusterSchemaKind: func(ctx context.Context, mgr ctrl.Manager) (common.ClusterSchemaReconciler, error) {
		return torque_schema.NewTorqueClusterReconciler(ctx, mgr)
	},
}

type EnabledSchemes []ClusterSchema

func (es *EnabledSchemes) String() string {
	if es == nil {
		return "nil"
	}
	var s []string
	for _, enabledSchema := range *es {
		s = append(s, string(enabledSchema))
	}
	return strings.Join(s, ",")
}

func (es *EnabledSchemes) Set(kind string) error {
	kindStr := strings.ToLower(kind)
	for supportedKind := range SupportedClusterSchemaReconciler {
		if strings.ToLower(string(supportedKind)) == kindStr {
			*es = append(*es, supportedKind)
			return nil
		}
	}
	return fmt.Errorf(ErrTemplateSchemeNotSupported, kind)
}

func (es *EnabledSchemes) FillAll() {
	for supportedKind := range SupportedClusterSchemaReconciler {
		*es = append(*es, supportedKind)
	}
}

func (es *EnabledSchemes) Empty() bool {
	return len(*es) == 0
}
