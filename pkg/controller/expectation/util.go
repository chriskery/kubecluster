package expectation

import (
	"strings"
)

// GenExpectationPodsKey generates an expectation key for pods of a job
func GenExpectationPodsKey(clusterKey string, replicaType string) string {
	return clusterKey + "/" + strings.ToLower(replicaType) + "/pods"
}

// GenExpectationServicesKey generates an expectation key for services of a job
func GenExpectationServicesKey(clusterKey string, replicaType string) string {
	return clusterKey + "/" + strings.ToLower(replicaType) + "/services"
}

// GenExpectationConfigMapKey generates an expectation key for services of a job
func GenExpectationConfigMapKey(clusterKey string) string {
	return clusterKey + "/configmap"
}

// GenPreSatisfiedKey generates an expectation key for services of a job
func GenPreSatisfiedKey(clusterKey string) string {
	return clusterKey + "/presatisfied"
}
