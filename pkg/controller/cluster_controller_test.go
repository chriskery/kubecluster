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
// limitations under the License.

package controller

//var _ = Describe("KubeCluster controller", func() {
//
//	Context("When creating the Slurm Schema Cluster", func() {
//		const name = "test-cluster"
//		var (
//			ns         *corev1.Namespace
//			cluster    *v1alpha1.KubeCluster
//			clusterKey types.NamespacedName
//			masterKey  types.NamespacedName
//			worker0Key types.NamespacedName
//			ctx        = context.Background()
//		)
//		BeforeEach(func() {
//			ns = &corev1.Namespace{
//				ObjectMeta: metav1.ObjectMeta{
//					GenerateName: "cluster-test-",
//				},
//			}
//			Expect(k8sClient.Create(ctx, ns)).Should(Succeed())
//
//			cluster = newClusterForTest(name, ns.Name)
//			clusterKey = client.ObjectKeyFromObject(cluster)
//			masterKey = types.NamespacedName{
//				Name:      fmt.Sprintf("%s-master-0", name),
//				Namespace: ns.Name,
//			}
//			worker0Key = types.NamespacedName{
//				Name:      fmt.Sprintf("%s-worker-0", name),
//				Namespace: ns.Name,
//			}
//			cluster.Spec.ClusterReplicaSpec = map[v1alpha1.ReplicaType]*v1alpha1.ReplicaSpec{
//				slurm_schema.SchemaReplicaTypeController: {
//					Replicas: pointer.Int32(1),
//					Template: v1alpha1.ReplicaTemplate{
//						Spec: corev1.PodSpec{
//							Containers: []corev1.Container{
//								{
//									Image: "test-image",
//									Name:  v1alpha1.ClusterDefaultContainerName,
//								},
//							},
//						},
//					},
//				},
//				"worker": {
//					Replicas: pointer.Int32(2),
//					Template: v1alpha1.ReplicaTemplate{
//						Spec: corev1.PodSpec{
//							Containers: []corev1.Container{
//								{
//									Image: "test-image",
//									Name:  v1alpha1.ClusterDefaultContainerName,
//								},
//							},
//						},
//					},
//				},
//			}
//		})
//		AfterEach(func() {
//			Expect(k8sClient.Delete(ctx, cluster)).Should(Succeed())
//			Expect(k8sClient.Delete(ctx, ns)).Should(Succeed())
//		})
//		It("Should get the corresponding resources successfully", func() {
//			By("By creating a new Cluster")
//			Expect(k8sClient.Create(ctx, cluster)).Should(Succeed())
//
//			created := &v1alpha1.KubeCluster{}
//
//			// We'll need to retry getting this newly created Cluster, given that creation may not immediately happen.
//			Eventually(func() bool {
//				err := k8sClient.Get(ctx, clusterKey, created)
//				return err == nil
//			}, testutil.Timeout, testutil.Interval).Should(BeTrue())
//
//			masterPod := &corev1.Pod{}
//			Eventually(func() bool {
//				err := k8sClient.Get(ctx, masterKey, masterPod)
//				return err == nil
//			}, testutil.Timeout, testutil.Interval).Should(BeTrue())
//
//			masterSvc := &corev1.Service{}
//			Eventually(func() bool {
//				err := k8sClient.Get(ctx, masterKey, masterSvc)
//				return err == nil
//			}, testutil.Timeout, testutil.Interval).Should(BeTrue())
//
//			// Check the pod port.
//			Expect(masterPod.Spec.Containers[0].Ports).To(ContainElement(corev1.ContainerPort{
//				Name:     slurm_schema.SlurmdPortName,
//				Protocol: corev1.ProtocolTCP}))
//			Expect(masterPod.Spec.Containers[0].Ports).To(ContainElement(corev1.ContainerPort{
//				Name:     slurm_schema.SlurmctlPortName,
//				Protocol: corev1.ProtocolTCP}))
//
//			// Check service port.
//			Expect(func() bool { return len(masterSvc.Spec.Ports) >= 2 }).To(Succeed())
//			// Check owner reference.
//			trueVal := true
//			Expect(masterPod.OwnerReferences).To(ContainElement(metav1.OwnerReference{
//				APIVersion:         v1alpha1.SchemeGroupVersion.String(),
//				Kind:               v1alpha1.KubeClusterKind,
//				Name:               name,
//				UID:                created.UID,
//				Controller:         &trueVal,
//				BlockOwnerDeletion: &trueVal,
//			}))
//			Expect(masterSvc.OwnerReferences).To(ContainElement(metav1.OwnerReference{
//				APIVersion:         v1alpha1.SchemeGroupVersion.String(),
//				Kind:               v1alpha1.KubeClusterKind,
//				Name:               name,
//				UID:                created.UID,
//				Controller:         &trueVal,
//				BlockOwnerDeletion: &trueVal,
//			}))
//
//			// Test cluster status.
//			Eventually(func() error {
//				Expect(k8sClient.Get(ctx, masterKey, masterPod)).Should(Succeed())
//				masterPod.Status.Phase = corev1.PodRunning
//				return k8sClient.Status().Update(ctx, masterPod)
//			}, testutil.Timeout, testutil.Interval).Should(Succeed())
//			Eventually(func() bool {
//				err := k8sClient.Get(ctx, clusterKey, created)
//				if err != nil {
//					return false
//				}
//				return created.Status.ReplicaStatuses != nil && created.Status.
//					ReplicaStatuses[slurm_schema.SchemaReplicaTypeController].Active == 1
//			}, testutil.Timeout, testutil.Interval).Should(BeTrue())
//			// Check if the cluster is succeeded.
//			cond := getCondition(created.Status, v1alpha1.ClusterRunning)
//			Expect(cond.Status).To(Equal(corev1.ConditionTrue))
//		})
//
//		It("Shouldn't create resources if Cluster is suspended", func() {
//			By("By creating a new Cluster with suspend=true")
//			cluster.Spec.RunPolicy.Suspend = pointer.Bool(true)
//			cluster.Spec.ClusterReplicaSpec["worker"].Replicas = pointer.Int32(1)
//			Expect(k8sClient.Create(ctx, cluster)).Should(Succeed())
//
//			created := &v1alpha1.KubeCluster{}
//			masterPod := &corev1.Pod{}
//			workerPod := &corev1.Pod{}
//			masterSvc := &corev1.Service{}
//			workerSvc := &corev1.Service{}
//
//			By("Checking created Cluster")
//			Eventually(func() bool {
//				err := k8sClient.Get(ctx, clusterKey, created)
//				return err == nil
//			}, testutil.Timeout, testutil.Interval).Should(BeTrue())
//			By("Checking created Cluster has a nil startTime")
//			Consistently(func() *metav1.Time {
//				Expect(k8sClient.Get(ctx, clusterKey, created)).Should(Succeed())
//				return created.Status.StartTime
//			}, testutil.ConsistentDuration, testutil.Interval).Should(BeNil())
//
//			By("Checking if the pods and services aren't created")
//			Consistently(func() bool {
//				errMasterPod := k8sClient.Get(ctx, masterKey, masterPod)
//				errWorkerPod := k8sClient.Get(ctx, worker0Key, workerPod)
//				errMasterSvc := k8sClient.Get(ctx, masterKey, masterSvc)
//				errWorkerSvc := k8sClient.Get(ctx, worker0Key, workerSvc)
//				return errors.IsNotFound(errMasterPod) && errors.IsNotFound(errWorkerPod) &&
//					errors.IsNotFound(errMasterSvc) && errors.IsNotFound(errWorkerSvc)
//			}, testutil.ConsistentDuration, testutil.Interval).Should(BeTrue())
//
//			By("Checking if the Cluster has suspended condition")
//			Eventually(func() []v1alpha1.ClusterCondition {
//				Expect(k8sClient.Get(ctx, clusterKey, created)).Should(Succeed())
//				return created.Status.Conditions
//			}, testutil.ConsistentDuration, testutil.Interval).Should(BeComparableTo([]v1alpha1.ClusterCondition{
//				{
//					Type:    v1alpha1.ClusterCreated,
//					Status:  corev1.ConditionTrue,
//					Reason:  util.NewReason(v1alpha1.KubeClusterKind, util.ClusterCreatedReason),
//					Message: fmt.Sprintf("Cluster %s is created.", name),
//				},
//				{
//					Type:    v1alpha1.ClusterSuspended,
//					Status:  corev1.ConditionTrue,
//					Reason:  util.NewReason(v1alpha1.KubeClusterKind, util.ClusterSuspendedReason),
//					Message: fmt.Sprintf("Cluster %s is suspended.", name),
//				},
//			}, testutil.IgnoreClusterConditionsTimes))
//		})
//
//		It("Should delete resources after Cluster is suspended; Should resume Cluster after Cluster is unsuspended", func() {
//			By("By creating a new Cluster")
//			cluster.Spec.ClusterReplicaSpec["worker"].Replicas = pointer.Int32(1)
//			Expect(k8sClient.Create(ctx, cluster)).Should(Succeed())
//
//			created := &v1alpha1.KubeCluster{}
//			masterPod := &corev1.Pod{}
//			workerPod := &corev1.Pod{}
//			masterSvc := &corev1.Service{}
//			workerSvc := &corev1.Service{}
//
//			// We'll need to retry getting this newly created Cluster, given that creation may not immediately happen.
//			By("Checking created Cluster")
//			Eventually(func() bool {
//				err := k8sClient.Get(ctx, clusterKey, created)
//				return err == nil
//			}, testutil.Timeout, testutil.Interval).Should(BeTrue())
//
//			var startTimeBeforeSuspended *metav1.Time
//			Eventually(func() *metav1.Time {
//				Expect(k8sClient.Get(ctx, clusterKey, created)).Should(Succeed())
//				startTimeBeforeSuspended = created.Status.StartTime
//				return startTimeBeforeSuspended
//			}, testutil.Timeout, testutil.Interval).ShouldNot(BeNil())
//
//			By("Checking the created pods and services")
//			Eventually(func() bool {
//				errMaster := k8sClient.Get(ctx, masterKey, masterPod)
//				errWorker := k8sClient.Get(ctx, worker0Key, workerPod)
//				return errMaster == nil && errWorker == nil
//			}, testutil.Timeout, testutil.Interval).Should(BeTrue())
//			Eventually(func() bool {
//				errMaster := k8sClient.Get(ctx, masterKey, masterSvc)
//				errWorker := k8sClient.Get(ctx, worker0Key, workerSvc)
//				return errMaster == nil && errWorker == nil
//			}, testutil.Timeout, testutil.Interval).Should(BeTrue())
//
//			By("Updating the pod's phase with Running")
//			Eventually(func() error {
//				Expect(k8sClient.Get(ctx, masterKey, masterPod)).Should(Succeed())
//				masterPod.Status.Phase = corev1.PodRunning
//				return k8sClient.Status().Update(ctx, masterPod)
//			}, testutil.Timeout, testutil.Interval).Should(Succeed())
//			Eventually(func() error {
//				Expect(k8sClient.Get(ctx, worker0Key, workerPod)).Should(Succeed())
//				workerPod.Status.Phase = corev1.PodRunning
//				return k8sClient.Status().Update(ctx, workerPod)
//			}, testutil.Timeout, testutil.Interval).Should(Succeed())
//
//			By("Checking the Cluster's condition")
//			Eventually(func() []v1alpha1.ClusterCondition {
//				Expect(k8sClient.Get(ctx, clusterKey, created)).Should(Succeed())
//				return created.Status.Conditions
//			}, testutil.Timeout, testutil.Interval).Should(BeComparableTo([]v1alpha1.ClusterCondition{
//				{
//					Type:    v1alpha1.ClusterCreated,
//					Status:  corev1.ConditionTrue,
//					Reason:  util.NewReason(v1alpha1.KubeClusterKind, util.ClusterCreatedReason),
//					Message: fmt.Sprintf("Cluster %s is created.", name),
//				},
//				{
//					Type:    v1alpha1.ClusterRunning,
//					Status:  corev1.ConditionTrue,
//					Reason:  util.NewReason(v1alpha1.KubeClusterKind, util.ClusterRunningReason),
//					Message: fmt.Sprintf("Cluster %s is running.", name),
//				},
//			}, testutil.IgnoreClusterConditionsTimes))
//
//			By("Updating the Cluster with suspend=true")
//			Eventually(func() error {
//				Expect(k8sClient.Get(ctx, clusterKey, created)).Should(Succeed())
//				created.Spec.RunPolicy.Suspend = pointer.Bool(true)
//				return k8sClient.Update(ctx, created)
//			}, testutil.Timeout, testutil.Interval).Should(Succeed())
//
//			By("Checking if the pods and services are removed")
//			Eventually(func() bool {
//				errMaster := k8sClient.Get(ctx, masterKey, masterPod)
//				errWorker := k8sClient.Get(ctx, worker0Key, workerPod)
//				return errors.IsNotFound(errMaster) && errors.IsNotFound(errWorker)
//			}, testutil.Timeout, testutil.Interval).Should(BeTrue())
//			Eventually(func() bool {
//				errMaster := k8sClient.Get(ctx, masterKey, masterSvc)
//				errWorker := k8sClient.Get(ctx, worker0Key, workerSvc)
//				return errors.IsNotFound(errMaster) && errors.IsNotFound(errWorker)
//			}, testutil.Timeout, testutil.Interval).Should(BeTrue())
//			Consistently(func() bool {
//				errMasterPod := k8sClient.Get(ctx, masterKey, masterPod)
//				errWorkerPod := k8sClient.Get(ctx, worker0Key, workerPod)
//				errMasterSvc := k8sClient.Get(ctx, masterKey, masterSvc)
//				errWorkerSvc := k8sClient.Get(ctx, worker0Key, workerSvc)
//				return errors.IsNotFound(errMasterPod) && errors.IsNotFound(errWorkerPod) &&
//					errors.IsNotFound(errMasterSvc) && errors.IsNotFound(errWorkerSvc)
//			}, testutil.ConsistentDuration, testutil.Interval).Should(BeTrue())
//
//			By("Checking if the Cluster has a suspended condition")
//			Eventually(func() bool {
//				Expect(k8sClient.Get(ctx, clusterKey, created)).Should(Succeed())
//				return created.Status.ReplicaStatuses["worker"].Active == 0 &&
//					created.Status.StartTime.Equal(startTimeBeforeSuspended)
//			}, testutil.Timeout, testutil.Interval).Should(BeTrue())
//			Consistently(func() bool {
//				Expect(k8sClient.Get(ctx, clusterKey, created)).Should(Succeed())
//				return created.Status.ReplicaStatuses[slurm_schema.SchemaReplicaTypeController].Active == 0 &&
//					created.Status.StartTime.Equal(startTimeBeforeSuspended)
//			}, testutil.ConsistentDuration, testutil.Interval).Should(BeTrue())
//			Expect(created.Status.Conditions).Should(BeComparableTo([]v1alpha1.ClusterCondition{
//				{
//					Type:    v1alpha1.ClusterCreated,
//					Status:  corev1.ConditionTrue,
//					Reason:  util.NewReason(v1alpha1.KubeClusterKind, util.ClusterCreatedReason),
//					Message: fmt.Sprintf("Cluster %s is created.", name),
//				},
//				{
//					Type:    v1alpha1.ClusterRunning,
//					Status:  corev1.ConditionFalse,
//					Reason:  util.NewReason(v1alpha1.KubeClusterKind, util.ClusterSuspendedReason),
//					Message: fmt.Sprintf("Cluster %s is suspended.", name),
//				},
//				{
//					Type:    v1alpha1.ClusterSuspended,
//					Reason:  util.NewReason(v1alpha1.KubeClusterKind, util.ClusterSuspendedReason),
//					Message: fmt.Sprintf("Cluster %s is suspended.", name),
//					Status:  corev1.ConditionTrue,
//				},
//			}, testutil.IgnoreClusterConditionsTimes))
//
//			By("Unsuspending the Cluster")
//			Eventually(func() error {
//				Expect(k8sClient.Get(ctx, clusterKey, created)).Should(Succeed())
//				created.Spec.RunPolicy.Suspend = pointer.Bool(false)
//				return k8sClient.Update(ctx, created)
//			}, testutil.Timeout, testutil.Interval).Should(Succeed())
//			Eventually(func() *metav1.Time {
//				Expect(k8sClient.Get(ctx, clusterKey, created)).Should(Succeed())
//				return created.Status.StartTime
//			}, testutil.Timeout, testutil.Interval).ShouldNot(BeNil())
//
//			By("Check if the pods and services are created")
//			Eventually(func() error {
//				return k8sClient.Get(ctx, masterKey, masterPod)
//			}, testutil.Timeout, testutil.Interval).Should(BeNil())
//			Eventually(func() error {
//				return k8sClient.Get(ctx, worker0Key, workerPod)
//			}, testutil.Timeout, testutil.Interval).Should(BeNil())
//			Eventually(func() error {
//				return k8sClient.Get(ctx, masterKey, masterSvc)
//			}, testutil.Timeout, testutil.Interval).Should(BeNil())
//			Eventually(func() error {
//				return k8sClient.Get(ctx, worker0Key, workerSvc)
//			}, testutil.Timeout, testutil.Interval).Should(BeNil())
//
//			By("Updating Pod's condition with running")
//			Eventually(func() error {
//				Expect(k8sClient.Get(ctx, masterKey, masterPod)).Should(Succeed())
//				masterPod.Status.Phase = corev1.PodRunning
//				return k8sClient.Status().Update(ctx, masterPod)
//			}, testutil.Timeout, testutil.Interval).Should(Succeed())
//			Eventually(func() error {
//				Expect(k8sClient.Get(ctx, worker0Key, workerPod)).Should(Succeed())
//				workerPod.Status.Phase = corev1.PodRunning
//				return k8sClient.Status().Update(ctx, workerPod)
//			}, testutil.Timeout, testutil.Interval).Should(Succeed())
//
//			By("Checking if the Cluster has resumed conditions")
//			Eventually(func() []v1alpha1.ClusterCondition {
//				Expect(k8sClient.Get(ctx, clusterKey, created)).Should(Succeed())
//				return created.Status.Conditions
//			}, testutil.Timeout, testutil.Interval).Should(BeComparableTo([]v1alpha1.ClusterCondition{
//				{
//					Type:    v1alpha1.ClusterCreated,
//					Status:  corev1.ConditionTrue,
//					Reason:  util.NewReason(v1alpha1.KubeClusterKind, util.ClusterCreatedReason),
//					Message: fmt.Sprintf("Cluster %s is created.", name),
//				},
//				{
//					Type:    v1alpha1.ClusterSuspended,
//					Reason:  util.NewReason(v1alpha1.KubeClusterKind, util.ClusterResumedReason),
//					Message: fmt.Sprintf("Cluster %s is resumed.", name),
//					Status:  corev1.ConditionFalse,
//				},
//				{
//					Type:    v1alpha1.ClusterRunning,
//					Status:  corev1.ConditionTrue,
//					Reason:  util.NewReason(v1alpha1.KubeClusterKind, util.ClusterRunningReason),
//					Message: fmt.Sprintf("Cluster %s is running.", name),
//				},
//			}, testutil.IgnoreClusterConditionsTimes))
//
//			By("Checking if the startTime is updated")
//			Expect(created.Status.StartTime).ShouldNot(Equal(startTimeBeforeSuspended))
//		})
//	})
//
//})
//
//func newClusterForTest(name, namespace string) *v1alpha1.KubeCluster {
//	return &v1alpha1.KubeCluster{
//		ObjectMeta: metav1.ObjectMeta{
//			Name:      name,
//			Namespace: namespace,
//		},
//	}
//}
//
//// getCondition returns the condition with the provided type.
//func getCondition(status v1alpha1.ClusterStatus, condType v1alpha1.ClusterConditionType) *v1alpha1.ClusterCondition {
//	for _, condition := range status.Conditions {
//		if condition.Type == condType {
//			return &condition
//		}
//	}
//	return nil
//}
