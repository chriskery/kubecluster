package clusterutil

import (
	kubeclusterorgv1alpha1 "github.com/chriskery/kubecluster/apis/kubecluster.org/v1alpha1"
	"k8s.io/utils/pointer"
)

func IsRetryableExitCode(exitCode int32) bool {
	return exitCode >= 128
}

func IsClusterSuspended(runPolicy *kubeclusterorgv1alpha1.RunPolicy) bool {
	return runPolicy != nil && pointer.BoolDeref(runPolicy.Suspend, false)
}
