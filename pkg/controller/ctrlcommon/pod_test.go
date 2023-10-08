// Copyright 2018 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ctrlcommon

import (
	"github.com/chriskery/kubecluster/apis/kubecluster.org/v1alpha1"
	"github.com/chriskery/kubecluster/pkg/core"
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestSetRestartPolicy(t *testing.T) {
	testCases := map[string]struct {
		replicaSpec           *v1alpha1.ReplicaSpec
		expectedRestartPolicy v1.RestartPolicy
	}{
		"restartPolicy is ExitCode": {
			replicaSpec: &v1alpha1.ReplicaSpec{
				RestartPolicy: v1alpha1.RestartPolicyExitCode,
			},
			expectedRestartPolicy: v1.RestartPolicyNever,
		},
		"restartPolicy is Never": {
			replicaSpec: &v1alpha1.ReplicaSpec{
				RestartPolicy: v1alpha1.RestartPolicyNever,
			},
			expectedRestartPolicy: v1.RestartPolicyNever,
		},
		"restartPolicy is Always": {
			replicaSpec: &v1alpha1.ReplicaSpec{
				RestartPolicy: v1alpha1.RestartPolicyAlways,
			},
			expectedRestartPolicy: v1.RestartPolicyAlways,
		},
		"restartPolicy is OnFailure": {
			replicaSpec: &v1alpha1.ReplicaSpec{
				RestartPolicy: v1alpha1.RestartPolicyOnFailure,
			},
			expectedRestartPolicy: v1.RestartPolicyOnFailure,
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			podTemplate := &tc.replicaSpec.Template
			core.SetRestartPolicy((*v1.PodTemplateSpec)(podTemplate), tc.replicaSpec)
			if podTemplate.Spec.RestartPolicy != tc.expectedRestartPolicy {
				t.Errorf("Unexpected restartPolicy from SetRetartPolicy:\nwant:%v\ngot:%v\n", tc.expectedRestartPolicy, podTemplate.Spec.RestartPolicy)
			}
		})
	}
}

func TestIsCustomSchedulerSet(t *testing.T) {
	testCases := map[string]struct {
		replicaSpecs      map[v1alpha1.ReplicaType]*v1alpha1.ReplicaSpec
		gangSchedulerName string
		want              bool
	}{
		"replicaSpecs aren't set custom schedulerName": {
			replicaSpecs: map[v1alpha1.ReplicaType]*v1alpha1.ReplicaSpec{
				v1alpha1.ReplicaType("A"): {},
				v1alpha1.ReplicaType("B"): {},
			},
			gangSchedulerName: "alpha",
			want:              false,
		},
		"all replicaSpecs are set custom schedulerName": {
			replicaSpecs: map[v1alpha1.ReplicaType]*v1alpha1.ReplicaSpec{
				v1alpha1.ReplicaType("A"): {
					Template: v1alpha1.ReplicaTemplate(v1.PodTemplateSpec{
						Spec: v1.PodSpec{
							SchedulerName: "custom-a",
						},
					}),
				},
				v1alpha1.ReplicaType("B"): {
					Template: v1alpha1.ReplicaTemplate(v1.PodTemplateSpec{
						Spec: v1.PodSpec{
							SchedulerName: "custom-b",
						},
					}),
				},
			},
			gangSchedulerName: "beta",
			want:              true,
		},
		"one of replicaSpecs is set custom schedulerName": {
			replicaSpecs: map[v1alpha1.ReplicaType]*v1alpha1.ReplicaSpec{
				v1alpha1.ReplicaType("A"): {},
				v1alpha1.ReplicaType("B"): {
					Template: v1alpha1.ReplicaTemplate(v1.PodTemplateSpec{
						Spec: v1.PodSpec{
							SchedulerName: "custom-b",
						},
					}),
				},
			},
			gangSchedulerName: "gamma",
			want:              true,
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got := isCustomSchedulerSet(tc.replicaSpecs, tc.gangSchedulerName)
			if tc.want != got {
				t.Errorf("Unexpected value from isCustomSchedulerSet:\nwant:%v\ngot:%v\n", tc.want, got)
			}
		})
	}
}

func TestCalculatePodSliceSize(t *testing.T) {
	type testCase struct {
		pods         []*v1.Pod
		replicas     int
		expectedSize int
	}

	pods := []*v1.Pod{
		{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{v1alpha1.ReplicaIndexLabel: "0"},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{v1alpha1.ReplicaIndexLabel: "1"},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{v1alpha1.ReplicaIndexLabel: "2"},
			},
		},
	}

	var testCases = []testCase{
		{
			pods:         pods,
			replicas:     3,
			expectedSize: 3,
		},
		{
			pods:         pods,
			replicas:     4,
			expectedSize: 4,
		},
		{
			pods:         pods,
			replicas:     2,
			expectedSize: 3,
		},
		{
			pods: append(pods, &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{v1alpha1.ReplicaIndexLabel: "4"},
				},
			}),
			replicas:     3,
			expectedSize: 5,
		},
	}

	for _, tc := range testCases {
		result := core.CalculatePodSliceSize(tc.pods, tc.replicas)
		assert.Equal(t, tc.expectedSize, result)
	}
}

func TestFilterPodsForReplicaType(t *testing.T) {
	pods := []*v1.Pod{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:   "a",
				Labels: map[string]string{v1alpha1.ReplicaTypeLabel: "foo"},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:   "b",
				Labels: map[string]string{v1alpha1.ReplicaTypeLabel: "bar"},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:   "c",
				Labels: map[string]string{v1alpha1.ReplicaTypeLabel: "foo"},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:   "d",
				Labels: map[string]string{v1alpha1.ReplicaTypeLabel: "bar"},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "e",
				Labels: map[string]string{
					v1alpha1.ReplicaTypeLabel: "foo",
				},
			},
		},
	}
	c := &ClusterController{}
	got, err := c.FilterPodsForReplicaType(pods, "foo")
	if err != nil {
		t.Fatalf("FilterPodsForReplicaType returned error: %v", err)
	}
	want := []*v1.Pod{pods[0], pods[2], pods[4]}
	assert.Equal(t, want, got)
}
