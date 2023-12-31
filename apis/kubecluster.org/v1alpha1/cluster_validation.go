package v1alpha1

import (
	"errors"
	"fmt"
	apimachineryvalidation "k8s.io/apimachinery/pkg/api/validation"
	"k8s.io/apimachinery/pkg/util/validation"
	"strings"
)

func ValidateV1alphaCluster(cluster *KubeCluster) error {
	if errs := apimachineryvalidation.NameIsDNS1035Label(cluster.ObjectMeta.Name, false); errs != nil {
		return fmt.Errorf("TFCluster name is invalid: %v", errs)
	}
	if err := validateV1alphaClusterSpecs(cluster.Spec); err != nil {
		return err
	}
	return nil
}

func validateV1alphaClusterSpecs(spec ClusterSpec) error {
	if len(spec.ClusterType) == 0 {
		return fmt.Errorf("KubeCluster is not valid: cluster type expected")
	}
	if len(spec.ClusterReplicaSpec) == 0 {
		return fmt.Errorf("KubeCluster is not valid")
	}
	defaultContainerName := ClusterDefaultContainerName
	if (len(spec.MainContainer)) != 0 {
		defaultContainerName = spec.MainContainer
	}
	for rType, value := range spec.ClusterReplicaSpec {
		if errs := validation.IsDNS1035Label(strings.ToLower(string(rType))); len(errs) != 0 {
			return errors.New(strings.Join(errs, ";"))
		}
		if value == nil || len(value.Template.Spec.Containers) == 0 {
			return fmt.Errorf("KubeCluster is not valid: containers definition expected in %v", rType)
		}
		// Make sure the image is defined in the container.
		numNamedkubenode := 0
		for _, container := range value.Template.Spec.Containers {
			if container.Image == "" {
				msg := fmt.Sprintf("KubeCluster is not valid: Image is undefined in the container of %v", rType)
				return fmt.Errorf(msg)
			}
			if container.Name == defaultContainerName {
				numNamedkubenode++
			}
		}
		// Make sure there has at least one container named "tensorflow".
		if numNamedkubenode == 0 {
			msg := fmt.Sprintf("KubeClusterSpec is not valid: There is no container named %s in %v", ClusterDefaultContainerName, rType)
			return fmt.Errorf(msg)
		}
	}
	return nil
}
