package control

import (
	"context"
	"fmt"
	"github.com/chriskery/kubecluster/pkg/util"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/record"
)

// ConfigMapControlInterface is an interface that knows how to add or delete ConfigMaps
// created as an interface to allow testing.
type ConfigMapControlInterface interface {
	// CreateConfigMap creates new Services according to the spec.
	CreateConfigMap(namespace string, configmap *v1.ConfigMap, object runtime.Object) error
	// CreateConfigMapWithControllerRef creates new services according to the spec, and sets object as the service's controller.
	CreateConfigMapWithControllerRef(namespace string, configmap *v1.ConfigMap, object runtime.Object, controllerRef *metav1.OwnerReference) error
	// UpdateConfigMap patches the configmap.
	UpdateConfigMap(namespace string, configmap *v1.ConfigMap) error
	// DeleteConfigMap deletes the service identified by serviceID.
	DeleteConfigMap(namespace, configMapID string, object runtime.Object) error
}

// RealConfigMapControl is the default implementation of ServiceControlInterface.
type RealConfigMapControl struct {
	KubeClient clientset.Interface
	Recorder   record.EventRecorder
}

func (r RealConfigMapControl) CreateConfigMap(namespace string, configmap *v1.ConfigMap, object runtime.Object) error {
	return r.createConfigMap(namespace, configmap, object, nil)
}

func (r RealConfigMapControl) CreateConfigMapWithControllerRef(namespace string, configmap *v1.ConfigMap, controllerObject runtime.Object, controllerRef *metav1.OwnerReference) error {
	if err := ValidateControllerRef(controllerRef); err != nil {
		return err
	}
	err := r.createConfigMap(namespace, configmap, controllerObject, controllerRef)
	if err != nil {
		return err
	}
	return nil
}

func (r RealConfigMapControl) UpdateConfigMap(namespace string, configmap *v1.ConfigMap) error {
	newConfigMap, err := r.KubeClient.CoreV1().ConfigMaps(namespace).Update(context.Background(), configmap, metav1.UpdateOptions{})
	if err == nil {
		*configmap = *newConfigMap
	}
	return err
}

func (r RealConfigMapControl) DeleteConfigMap(namespace, configMapID string, object runtime.Object) error {
	accessor, err := meta.Accessor(object)
	if err != nil {
		return fmt.Errorf("object does not have ObjectMeta, %v", err)
	}
	logger := util.LoggerForCluster(accessor)
	cm, err := r.KubeClient.CoreV1().ConfigMaps(namespace).Get(context.TODO(), configMapID, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}
	if cm.DeletionTimestamp != nil {
		logger.Infof("ConfigMap %s/%s is terminating, skip deleting", cm.Namespace, cm.Name)
		return nil
	}
	logger.Infof("Controller %v deleting cm %v/%v", accessor.GetName(), namespace, configMapID)
	// delete options
	if err = r.KubeClient.CoreV1().ConfigMaps(namespace).Delete(context.TODO(), configMapID, metav1.DeleteOptions{}); err != nil {
		r.Recorder.Eventf(object, v1.EventTypeWarning, FailedDeletePodReason, "Error deleting: %v", err)
		return fmt.Errorf("unable to delete pods: %v", err)
	} else {
		r.Recorder.Eventf(object, v1.EventTypeNormal, SuccessfulDeletePodReason, "Deleted configMap: %v", configMapID)
	}
	return nil
}

func (r RealConfigMapControl) createConfigMap(
	namespace string,
	template *v1.ConfigMap,
	object runtime.Object,
	controllerRef *metav1.OwnerReference,
) error {
	cm, err := GetConfigMapFromTemplate(template, object, controllerRef)
	if err != nil {
		return err
	}
	logger := util.LoggerForConfigMap(cm, object.GetObjectKind().GroupVersionKind().Kind)
	newConfigMap, err := r.KubeClient.CoreV1().ConfigMaps(namespace).Create(context.TODO(), cm, metav1.CreateOptions{})
	if err != nil {
		r.Recorder.Eventf(object, v1.EventTypeWarning, FailedCreatePodReason, "Error creating: %v", err)
		return err
	}
	accessor, err := meta.Accessor(object)
	if err != nil {
		logger.Errorf("parentObject does not have ObjectMeta, %v", err)
		return nil
	}
	*template = *newConfigMap
	logger.Infof("Controller %v created confgimap %v", accessor.GetName(), newConfigMap.Name)
	r.Recorder.Eventf(object, v1.EventTypeNormal, SuccessfulCreatePodReason, "Created pod: %v", newConfigMap.Name)
	return nil
}

func GetConfigMapFromTemplate(template *v1.ConfigMap, object runtime.Object, controllerRef *metav1.OwnerReference) (*v1.ConfigMap, error) {
	objectMetaCopy := template.ObjectMeta.DeepCopy()
	cm := &v1.ConfigMap{
		ObjectMeta: *objectMetaCopy,
	}
	if controllerRef != nil {
		cm.OwnerReferences = append(cm.OwnerReferences, *controllerRef)
	}
	cm.Data = template.Data
	return cm, nil
}
