package cluster_schema

import (
	"github.com/chriskery/kubecluster/pkg/controller/cluster_schema/pbspro_schema"
	"github.com/chriskery/kubecluster/pkg/controller/cluster_schema/slurm_schema"
	"testing"
)

func TestEnabledSchemes(t *testing.T) {
	testES := EnabledSchemes{}

	if testES.String() != "" {
		t.Errorf("empty EnabledSchemes converted no-empty string %s", testES.String())
	}

	if !testES.Empty() {
		t.Error("Empty method returned false for empty EnabledSchemes")
	}

	if testES.Set(slurm_schema.ClusterSchemaKind) != nil {
		t.Error("failed to restore Slurm schema")
	} else {
		stored := false
		for _, kind := range testES {
			if kind == slurm_schema.ClusterSchemaKind {
				stored = true
			}
		}
		if !stored {
			t.Errorf("%s not successfully registered", slurm_schema.ClusterSchemaKind)
		}
	}

	if testES.Set(pbspro_schema.ClusterSchemaKind) != nil {
		t.Error("failed to restore Slurm schema")
	} else {
		stored := false
		for _, kind := range testES {
			if kind == pbspro_schema.ClusterSchemaKind {
				stored = true
			}
		}
		if !stored {
			t.Errorf("%s not successfully registered", pbspro_schema.ClusterSchemaKind)
		}
	}
	dummycluster := "dummycluster"
	if testES.Set(dummycluster) == nil {
		t.Errorf("successfully registered non-supported cluster %s", dummycluster)
	}

	if testES.Empty() {
		t.Error("Empty method returned true for non-empty EnabledSchemes")
	}

	es2 := EnabledSchemes{}
	es2.FillAll()
	if es2.Empty() {
		t.Error("Empty method returned true for fully registered EnabledSchemes")
	}
}
