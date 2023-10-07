package v1alpha1

import (
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"

	"testing"
)

func TestSetDefaults_KubeCluster(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)

	t.Run("test empty kube cluster", func(t *testing.T) {
		var replica2 int32 = 2
		kclusetr := &KubeCluster{Spec: ClusterSpec{
			ClusterType: "",
			ClusterReplicaSpec: map[ReplicaType]*ReplicaSpec{
				"replica1": {}, "replica2": {Replicas: &replica2}},
			MainContainer: "",
			RunPolicy:     RunPolicy{},
		}}
		SetDefaults_KubeCluster(kclusetr)

		gomega.Expect(kclusetr.Spec.MainContainer).To(gomega.Equal(ClusterDefaultContainerName))
		gomega.Expect(*kclusetr.Spec.RunPolicy.CleanKubeNodePolicy).To(gomega.Equal(CleanKubeNodePolicyAll))

		gomega.Expect(*kclusetr.Spec.ClusterReplicaSpec["replica1"].Replicas).To(gomega.Equal(int32(1)))
		gomega.Expect(*kclusetr.Spec.ClusterReplicaSpec["replica2"].Replicas).To(gomega.Equal(int32(2)))
	})
}

func TestSetDefaults_KubeClusterList(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)

	var replica2 int32 = 2
	testCount := 10
	kclusterList := &KubeClusterList{Items: make([]KubeCluster, 0)}
	for i := 0; i < testCount; i++ {
		kclusetr := KubeCluster{Spec: ClusterSpec{
			ClusterType: "",
			ClusterReplicaSpec: map[ReplicaType]*ReplicaSpec{
				"replica1": {}, "replica2": {Replicas: &replica2}},
			MainContainer: "",
			RunPolicy:     RunPolicy{},
		}}
		kclusterList.Items = append(kclusterList.Items, kclusetr)
	}
	SetDefaults_KubeClusterList(kclusterList)
	for i := 0; i < testCount; i++ {
		t.Run("test empty kube cluster", func(t *testing.T) {
			kclusetr := kclusterList.Items[i]
			gomega.Expect(kclusetr.Spec.MainContainer).To(gomega.Equal(ClusterDefaultContainerName))
			gomega.Expect(*kclusetr.Spec.RunPolicy.CleanKubeNodePolicy).To(gomega.Equal(CleanKubeNodePolicyAll))

			gomega.Expect(*kclusetr.Spec.ClusterReplicaSpec["replica1"].Replicas).To(gomega.Equal(int32(1)))
			gomega.Expect(*kclusetr.Spec.ClusterReplicaSpec["replica2"].Replicas).To(gomega.Equal(int32(2)))
		})
	}
}

func Test_setDefaultReplicas(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)

	t.Run("test replicaSpec not set replica", func(t *testing.T) {
		replicaSpec := &ReplicaSpec{}
		setDefaultReplicas(replicaSpec, 1)
		gomega.Expect(*replicaSpec.Replicas).To(gomega.Equal(int32(1)))
	})
}
