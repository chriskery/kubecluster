package util

import (
	"fmt"
	"github.com/chriskery/kubecluster/apis/kubecluster.org/v1alpha1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// ClusterCreatedReason is added in a KubeCluster when it is created.
	ClusterCreatedReason = "Created"
	// ClusterRunningReason is added in a KubeCluster when it is running.
	ClusterRunningReason = "Running"
	// ClusterFailedReason is added in a KubeCluster when it is failed.
	ClusterFailedReason = "Failed"
	// ClusterRestartingReason is added in a KubeCluster when it is restarting.
	ClusterRestartingReason = "Restarting"
	// ClusterFailedValidationReason is added in a KubeCluster when it failed validation
	ClusterFailedValidationReason = "FailedValidation"
	// ClusterSuspendedReason is added in a KubeCluster when it is suspended.
	ClusterSuspendedReason = "Suspended"
	// ClusterResumedReason is added in a KubeCluster when it is unsuspended.
	ClusterResumedReason = "Resumed"
)

func NewReason(kind, reason string) string {
	return fmt.Sprintf("%s%s", kind, reason)
}

// IsFinished checks if the cluster is succeeded or failed
func IsFinished(status v1alpha1.ClusterStatus) bool {
	return IsFailed(status)
}

// IsFailed checks if the KubeCluster is failed
func IsFailed(status v1alpha1.ClusterStatus) bool {
	return isStatusConditionTrue(status, v1alpha1.ClusterFailed)
}

func IsRunning(status v1alpha1.ClusterStatus) bool {
	return isStatusConditionTrue(status, v1alpha1.ClusterRunning)
}

func IsSuspended(status v1alpha1.ClusterStatus) bool {
	return isStatusConditionTrue(status, v1alpha1.ClusterSuspended)
}

// UpdateClusterConditions adds to the ClusterStatus a new condition if needed, with the conditionType, reason, and message
func UpdateClusterConditions(
	ClusterStatus *v1alpha1.ClusterStatus,
	conditionType v1alpha1.ClusterConditionType,
	conditionStatus v1.ConditionStatus,
	reason, message string,
) {
	condition := newCondition(conditionType, conditionStatus, reason, message)
	setCondition(ClusterStatus, condition)
}

func isStatusConditionTrue(status v1alpha1.ClusterStatus, condType v1alpha1.ClusterConditionType) bool {
	for _, condition := range status.Conditions {
		if condition.Type == condType && condition.Status == v1.ConditionTrue {
			return true
		}
	}
	return false
}

// newCondition creates a new KubeCluster condition.
func newCondition(conditionType v1alpha1.ClusterConditionType, conditionStatus v1.ConditionStatus, reason, message string) v1alpha1.ClusterCondition {
	return v1alpha1.ClusterCondition{
		Type:               conditionType,
		Status:             conditionStatus,
		LastUpdateTime:     metav1.Now(),
		LastTransitionTime: metav1.Now(),
		Reason:             reason,
		Message:            message,
	}
}

// getCondition returns the condition with the provided type.
func getCondition(status v1alpha1.ClusterStatus, condType v1alpha1.ClusterConditionType) *v1alpha1.ClusterCondition {
	for _, condition := range status.Conditions {
		if condition.Type == condType {
			return &condition
		}
	}
	return nil
}

// setCondition updates the KubeCluster to include the provided condition.
// If the condition that we are about to add already exists
// and has the same status and reason then we are not going to update.
func setCondition(status *v1alpha1.ClusterStatus, condition v1alpha1.ClusterCondition) {
	// Do nothing if ClusterStatus have failed condition
	if IsFailed(*status) {
		return
	}

	currentCond := getCondition(*status, condition.Type)

	// Do nothing if condition doesn't change
	if currentCond != nil && currentCond.Status == condition.Status && currentCond.Reason == condition.Reason {
		return
	}

	// Do not update lastTransitionTime if the status of the condition doesn't change.
	if currentCond != nil && currentCond.Status == condition.Status {
		condition.LastTransitionTime = currentCond.LastTransitionTime
	}

	// Append the updated condition to the conditions
	newConditions := filterOutCondition(status.Conditions, condition.Type)
	status.Conditions = append(newConditions, condition)
}

// filterOutCondition returns a new slice of KubeCluster conditions without conditions with the provided type.
func filterOutCondition(conditions []v1alpha1.ClusterCondition, condType v1alpha1.ClusterConditionType) []v1alpha1.ClusterCondition {
	var newConditions []v1alpha1.ClusterCondition
	for _, c := range conditions {
		if condType == v1alpha1.ClusterRestarting && c.Type == v1alpha1.ClusterRunning {
			continue
		}
		if condType == v1alpha1.ClusterRunning && c.Type == v1alpha1.ClusterRestarting {
			continue
		}

		if c.Type == condType {
			continue
		}

		// Set the running condition status to be false when current condition failed or succeeded
		if condType == v1alpha1.ClusterFailed && c.Type == v1alpha1.ClusterRunning {
			c.Status = v1.ConditionFalse
		}

		newConditions = append(newConditions, c)
	}
	return newConditions
}
