package util

import (
	"github.com/chriskery/kubecluster/apis/kubecluster.org/v1alpha1"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

func TestIsFinished(t *testing.T) {
	cases := map[string]struct {
		ClusterStatus v1alpha1.ClusterStatus
		want          bool
	}{
		"Running Cluster": {
			ClusterStatus: v1alpha1.ClusterStatus{
				Conditions: []v1alpha1.ClusterCondition{
					{
						Type:   v1alpha1.ClusterRunning,
						Status: corev1.ConditionTrue,
					},
				},
			},
			want: false,
		},
		"Failed Cluster": {
			ClusterStatus: v1alpha1.ClusterStatus{
				Conditions: []v1alpha1.ClusterCondition{
					{
						Type:   v1alpha1.ClusterFailed,
						Status: corev1.ConditionTrue,
					},
				},
			},
			want: true,
		},
		"Suspended Cluster": {
			ClusterStatus: v1alpha1.ClusterStatus{
				Conditions: []v1alpha1.ClusterCondition{
					{
						Type:   v1alpha1.ClusterSuspended,
						Status: corev1.ConditionTrue,
					},
				},
			},
			want: false,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got := IsFinished(tc.ClusterStatus)
			if tc.want != got {
				t.Errorf("Unexpected result from IsFinished() \nwant: %v, got: %v\n", tc.want, got)
			}
		})
	}
}

func TestIsFailed(t *testing.T) {
	ClusterStatus := v1alpha1.ClusterStatus{
		Conditions: []v1alpha1.ClusterCondition{
			{
				Type:   v1alpha1.ClusterFailed,
				Status: corev1.ConditionTrue,
			},
		},
	}
	assert.True(t, IsFailed(ClusterStatus))
}

func TestIsRunning(t *testing.T) {
	ClusterStatus := v1alpha1.ClusterStatus{
		Conditions: []v1alpha1.ClusterCondition{
			{
				Type:   v1alpha1.ClusterRunning,
				Status: corev1.ConditionTrue,
			},
		},
	}
	assert.True(t, IsRunning(ClusterStatus))
}

func TestIsSuspended(t *testing.T) {
	ClusterStatus := v1alpha1.ClusterStatus{
		Conditions: []v1alpha1.ClusterCondition{
			{
				Type:   v1alpha1.ClusterSuspended,
				Status: corev1.ConditionTrue,
			},
		},
	}
	assert.True(t, IsSuspended(ClusterStatus))
}

func TestUpdateClusterConditions(t *testing.T) {
	ClusterStatus := v1alpha1.ClusterStatus{}
	conditionType := v1alpha1.ClusterCreated
	reason := "Cluster Created"
	message := "Cluster Created"

	UpdateClusterConditions(&ClusterStatus, conditionType, corev1.ConditionTrue, reason, message)
	// Check ClusterCreated condition is appended
	conditionInStatus := ClusterStatus.Conditions[0]
	assert.Equal(t, conditionInStatus.Type, conditionType)
	assert.Equal(t, conditionInStatus.Reason, reason)
	assert.Equal(t, conditionInStatus.Message, message)

	conditionType = v1alpha1.ClusterRunning
	reason = "Cluster Running"
	message = "Cluster Running"
	UpdateClusterConditions(&ClusterStatus, conditionType, corev1.ConditionTrue, reason, message)
	// Check ClusterRunning condition is appended
	conditionInStatus = ClusterStatus.Conditions[1]
	assert.Equal(t, conditionInStatus.Type, conditionType)
	assert.Equal(t, conditionInStatus.Reason, reason)
	assert.Equal(t, conditionInStatus.Message, message)

	conditionType = v1alpha1.ClusterRestarting
	reason = "Cluster Restarting"
	message = "Cluster Restarting"
	UpdateClusterConditions(&ClusterStatus, conditionType, corev1.ConditionTrue, reason, message)
	// Check ClusterRunning condition is filtered out and ClusterRestarting state is appended
	conditionInStatus = ClusterStatus.Conditions[1]
	assert.Equal(t, conditionInStatus.Type, conditionType)
	assert.Equal(t, conditionInStatus.Reason, reason)
	assert.Equal(t, conditionInStatus.Message, message)

	conditionType = v1alpha1.ClusterRunning
	reason = "Cluster Running"
	message = "Cluster Running"
	UpdateClusterConditions(&ClusterStatus, conditionType, corev1.ConditionTrue, reason, message)
	// Again, Check ClusterRestarting condition is filtered and ClusterRestarting is appended
	conditionInStatus = ClusterStatus.Conditions[1]
	assert.Equal(t, conditionInStatus.Type, conditionType)
	assert.Equal(t, conditionInStatus.Reason, reason)
	assert.Equal(t, conditionInStatus.Message, message)

	conditionType = v1alpha1.ClusterFailed
	reason = "Cluster Failed"
	message = "Cluster Failed"
	UpdateClusterConditions(&ClusterStatus, conditionType, corev1.ConditionTrue, reason, message)
	// Check ClusterRunning condition is set to false
	ClusterRunningCondition := ClusterStatus.Conditions[1]
	assert.Equal(t, ClusterRunningCondition.Type, v1alpha1.ClusterRunning)
	assert.Equal(t, ClusterRunningCondition.Status, corev1.ConditionFalse)
	// Check ClusterFailed state is appended
	conditionInStatus = ClusterStatus.Conditions[2]
	assert.Equal(t, conditionInStatus.Type, conditionType)
	assert.Equal(t, conditionInStatus.Reason, reason)
	assert.Equal(t, conditionInStatus.Message, message)
}
