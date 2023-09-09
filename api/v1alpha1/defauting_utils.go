package v1alpha1

func CleanPodPolicyPointer(cleanKubeNodePolicy CleanKubeNodePolicy) *CleanKubeNodePolicy {
	return &cleanKubeNodePolicy
}
