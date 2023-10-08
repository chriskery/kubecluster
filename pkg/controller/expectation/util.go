package expectation

import (
	"strings"
)

// GenExpectationPodsKey generates an expectation key for pods of a cluster
func GenExpectationPodsKey(clusterKey string, replicaType string) string {
	return clusterKey + "/" + strings.ToLower(replicaType) + "/pods"
}

// GenExpectationServicesKey generates an expectation key for services of a cluster
func GenExpectationServicesKey(clusterKey string, replicaType string) string {
	return clusterKey + "/" + strings.ToLower(replicaType) + "/services"
}

// GenExpectationConfigMapKey generates an expectation key for services of a cluster
func GenExpectationConfigMapKey(clusterKey string) string {
	return clusterKey + "/configmap"
}

// GenPreSatisfiedKey generates an expectation key for services of a cluster
func GenPreSatisfiedKey(clusterKey string) string {
	return clusterKey + "/presatisfied"
}
