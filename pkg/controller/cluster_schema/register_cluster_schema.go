package cluster_schema

import (
	"context"
	"fmt"
	"github.com/kubecluster/pkg/common"
	"github.com/kubecluster/pkg/controller/cluster_schema/slurm"
	"strings"
)

const ErrTemplateSchemeNotSupported = "cluster scheme %s is not supported yet"

type ClusterSchema string

type ClusterSchemaFactory func(ctx context.Context) (common.ClusterSchemaReconciler, error)

var SupportedClusterSchemaReconciler = map[ClusterSchema]ClusterSchemaFactory{
	slurm.ClusterSchemaKind: func(ctx context.Context) (common.ClusterSchemaReconciler, error) {
		return slurm.NewSlurmClusterReconciler(ctx)
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
