// Copyright 2021 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License

package slurm_schema

import (
	"github.com/chriskery/kubecluster/apis/kubecluster.org/v1alpha1"
	"github.com/chriskery/kubecluster/pkg/common"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strconv"
	"testing"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"k8s.io/utils/pointer"
)

func TestInitContainer(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)
	config.slurmSchemaInitContainerImage = slurmSchemaInitContainerImageDefault
	config.slurmSchemaInitContainerTemplateFile = slurmSchemaInitContainerTemplateFileDefault
	config.slurmSchemaInitContainerMaxTries = slurmSchemaInitContainerMaxTriesDefault

	var replicaTypeWorker v1alpha1.ReplicaType = "Worker"

	testCases := []struct {
		kubecluster *v1alpha1.KubeCluster
		rtype       v1alpha1.ReplicaType
		index       string
		expected    int
		exepctedErr error
	}{
		{
			kubecluster: &v1alpha1.KubeCluster{
				Spec: v1alpha1.ClusterSpec{
					ClusterReplicaSpec: map[v1alpha1.ReplicaType]*v1alpha1.ReplicaSpec{
						SchemaReplicaTypeController: {
							Replicas: pointer.Int32(1),
						},
					},
				},
			},
			rtype:       SchemaReplicaTypeController,
			index:       "0",
			expected:    1,
			exepctedErr: nil,
		},
		{
			kubecluster: &v1alpha1.KubeCluster{
				Spec: v1alpha1.ClusterSpec{
					ClusterReplicaSpec: map[v1alpha1.ReplicaType]*v1alpha1.ReplicaSpec{
						replicaTypeWorker: {
							Replicas: pointer.Int32(1),
						},
						SchemaReplicaTypeController: {
							Replicas: pointer.Int32(1),
						},
					},
				},
			},
			rtype:       replicaTypeWorker,
			index:       "0",
			expected:    1,
			exepctedErr: nil,
		},
		{
			kubecluster: &v1alpha1.KubeCluster{
				Spec: v1alpha1.ClusterSpec{
					ClusterReplicaSpec: map[v1alpha1.ReplicaType]*v1alpha1.ReplicaSpec{
						replicaTypeWorker: {
							Replicas: pointer.Int32(1),
						},
						SchemaReplicaTypeController: {
							Replicas: pointer.Int32(1),
						},
					},
				},
			},
			rtype:       SchemaReplicaTypeController,
			index:       "0",
			expected:    1,
			exepctedErr: nil,
		},
	}

	for _, t := range testCases {
		podTemplateSpec := t.kubecluster.Spec.ClusterReplicaSpec[t.rtype].Template
		err := setInitContainer(t.kubecluster, (*corev1.PodTemplateSpec)(&podTemplateSpec), t.rtype)
		if t.exepctedErr == nil {
			gomega.Expect(err).To(gomega.BeNil())
		} else {
			gomega.Expect(err).To(gomega.Equal(t.exepctedErr))
		}
		gomega.Expect(len(podTemplateSpec.Spec.InitContainers) >= t.expected).To(gomega.BeTrue())
	}
}

func TestGetInitContainer(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)

	g := getInitContainerGenerator()
	testCases := []struct {
		kubecluster *v1alpha1.KubeCluster
		expected    int
		exepctedErr error
	}{
		{
			kubecluster: &v1alpha1.KubeCluster{
				ObjectMeta: metav1.ObjectMeta{Name: "test"},
				Spec: v1alpha1.ClusterSpec{
					ClusterReplicaSpec: map[v1alpha1.ReplicaType]*v1alpha1.ReplicaSpec{
						SchemaReplicaTypeController: {
							Replicas: pointer.Int32(1),
						},
					},
				},
			},
			expected:    1,
			exepctedErr: nil,
		},
	}

	for _, t := range testCases {
		controllerAddress := common.GenGeneralName(t.kubecluster.Name, SchemaReplicaTypeController, strconv.Itoa(0))
		initContainers, err := g.GetInitContainer(controllerAddress)
		if t.exepctedErr == nil {
			gomega.Expect(err).To(gomega.BeNil())
		} else {
			gomega.Expect(err).To(gomega.Equal(t.exepctedErr))
		}
		gomega.Expect(len(initContainers) >= t.expected).To(gomega.BeTrue())
	}
}
