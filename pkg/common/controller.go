package common

import (
	kubeclusterorgv1alpha1 "github.com/chriskery/kubecluster/apis/kubecluster.org/v1alpha1"
	"strings"
)

const (
	ControllerName = "kubecluster-controller"
)

func GenGeneralName(clusterName string, rtype kubeclusterorgv1alpha1.ReplicaType, index string) string {
	n := clusterName + "-" + strings.ToLower(string(rtype)) + "-" + index
	return strings.Replace(n, "/", "-", -1)
}
