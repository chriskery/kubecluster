package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
	"testing"
)

func TestValidateV1alphaCluster(t *testing.T) {
	validClusterSpec := ClusterSpec{
		ClusterType: "test",
		ClusterReplicaSpec: map[ReplicaType]*ReplicaSpec{"test": {
			Replicas: pointer.Int32(1),
			Template: ReplicaTemplate{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:  ClusterDefaultContainerName,
						Image: "centos",
					}},
				},
			},
		}},
	}
	type args struct {
		cluster *KubeCluster
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			"valid kubeCluster",
			args{
				&KubeCluster{
					ObjectMeta: metav1.ObjectMeta{Name: "test"},
					Spec:       validClusterSpec,
				},
			},
			false,
		},
		{
			"kubeCluster name does not meet DNS1035",
			args{
				&KubeCluster{
					ObjectMeta: metav1.ObjectMeta{Name: "0-test"},
					Spec:       validClusterSpec,
				},
			},
			true,
		},
		{
			"cluster type is empty",
			args{
				&KubeCluster{
					ObjectMeta: metav1.ObjectMeta{Name: "test"},
					Spec:       ClusterSpec{ClusterType: ""},
				},
			},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ValidateV1alphaCluster(tt.args.cluster); (err != nil) != tt.wantErr {
				t.Errorf("ValidateV1alphaCluster() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
	Test_validateV1alphaClusterSpecs(t)
}

func Test_validateV1alphaClusterSpecs(t *testing.T) {
	type args struct {
		spec ClusterSpec
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			"replica specs are empty",
			args{
				ClusterSpec{
					ClusterType:        "test",
					ClusterReplicaSpec: make(map[ReplicaType]*ReplicaSpec),
				},
			},
			true,
		},
		{
			"no containers",
			args{
				ClusterSpec{
					ClusterType: "test",
					ClusterReplicaSpec: map[ReplicaType]*ReplicaSpec{"test": {
						Replicas: pointer.Int32(1),
						Template: ReplicaTemplate{
							Spec: corev1.PodSpec{},
						},
					}},
				},
			},
			true,
		},
		{
			"main container name doesn't present",
			args{
				ClusterSpec{
					ClusterType: "test",
					ClusterReplicaSpec: map[ReplicaType]*ReplicaSpec{"test": {
						Replicas: pointer.Int32(1),
						Template: ReplicaTemplate{
							Spec: corev1.PodSpec{
								Containers: []corev1.Container{{
									Image: "centos",
								}},
							},
						},
					}},
				},
			},
			true,
		},
		{
			"main container name not consistent",
			args{
				ClusterSpec{
					ClusterType:   "test",
					MainContainer: "testmain",
					ClusterReplicaSpec: map[ReplicaType]*ReplicaSpec{"test": {
						Replicas: pointer.Int32(1),
						Template: ReplicaTemplate{
							Spec: corev1.PodSpec{
								Containers: []corev1.Container{{
									Name:  ClusterDefaultContainerName,
									Image: "centos",
								}},
							},
						},
					}},
				},
			},
			true,
		},
		{
			"image is empty",
			args{
				ClusterSpec{
					ClusterType: "test",
					ClusterReplicaSpec: map[ReplicaType]*ReplicaSpec{"test": {
						Replicas: pointer.Int32(1),
						Template: ReplicaTemplate{
							Spec: corev1.PodSpec{
								Containers: []corev1.Container{{
									Name: ClusterDefaultContainerName,
								}},
							},
						},
					}},
				},
			},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := validateV1alphaClusterSpecs(tt.args.spec); (err != nil) != tt.wantErr {
				t.Errorf("validateV1alphaClusterSpecs() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
