package testutil

import (
	"github.com/chriskery/kubecluster/apis/kubecluster.org/v1alpha1"
	"time"

	"github.com/google/go-cmp/cmp/cmpopts"
)

const (
	Timeout            = 30 * time.Second
	Interval           = 250 * time.Millisecond
	ConsistentDuration = 3 * time.Second
)

var (
	IgnoreClusterConditionsTimes = cmpopts.IgnoreFields(v1alpha1.ClusterCondition{}, "LastUpdateTime", "LastTransitionTime")
)
