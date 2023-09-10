package util

import (
	kubeclusterorgv1alpha1 "github.com/kubecluster/apis/kubecluster.org/v1alpha1"
	"k8s.io/utils/pointer"
)

func IsClusterSuspended(runPolicy *kubeclusterorgv1alpha1.RunPolicy) bool {
	return runPolicy != nil && pointer.BoolDeref(runPolicy.Suspend, false)
}
