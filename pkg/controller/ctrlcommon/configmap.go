package ctrlcommon

import (
	"fmt"
	"github.com/chriskery/kubecluster/apis/kubecluster.org/v1alpha1"
	"github.com/chriskery/kubecluster/pkg/controller/expectation"
	utillabels "github.com/chriskery/kubecluster/pkg/util/labels"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"reflect"
)

var (
	succeededConfigMapCreationCount = promauto.NewCounter(prometheus.CounterOpts{
		Name: "succeeded_configmap_creation_total",
		Help: "The total number of succeeded configmap creation",
	})
	failedConfigMapCreationCount = promauto.NewCounter(prometheus.CounterOpts{
		Name: "failed_configmap_creation_total",
		Help: "The total number of failed configmap creation",
	})
)

func (cc *ClusterController) ReconcileConfigMap(kcluster *v1alpha1.KubeCluster) (*v1.ConfigMap, error) {
	configMap, err := cc.Controller.GetConfigMapForCluster(kcluster)
	if err != nil {
		return nil, err
	}

	schemaReconciler := cc.GetSchemaReconciler(kcluster.Spec.ClusterType)
	if configMap == nil {
		configMap, err = cc.CreateNewConfigMap(kcluster, schemaReconciler)
		if err != nil {
			return nil, err
		}
	}
	deepCopy := configMap.DeepCopy()
	if err = schemaReconciler.ReconcileConfigMap(kcluster, deepCopy); err != nil {
		return nil, err
	}
	// No need to update the cluster status if the status hasn't changed since last time.
	if !reflect.DeepEqual(deepCopy.Data, configMap.Data) {
		err = cc.Controller.UpdateConfigMapInApiServer(kcluster, deepCopy)
	}
	return deepCopy, err
}

func (cc *ClusterController) CreateNewConfigMap(kcluster *v1alpha1.KubeCluster, expectations expectation.ControllerExpectationsInterface) (*v1.ConfigMap, error) {
	clusetrKey, err := KeyFunc(kcluster)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("couldn't get key for cluster object %#v: %v", kcluster, err))
		return nil, err
	}

	labels := cc.GenLabels(kcluster.GetName())
	utillabels.SetClusterType(labels, string(kcluster.Spec.ClusterType))

	// Create OwnerReference.
	controllerRef := cc.GenOwnerReference(kcluster)

	// Creation is expected when there is no error returned
	expectationConfigmapsKey := expectation.GenExpectationConfigMapKey(clusetrKey)
	expectations.RaiseExpectations(expectationConfigmapsKey, 1, 0)

	configMap := &v1.ConfigMap{Data: make(map[string]string)}
	configMap.SetName(kcluster.GetName())

	err = cc.ConfigMapControl.CreateConfigMapWithControllerRef(kcluster.GetNamespace(), configMap, kcluster, controllerRef)
	if err != nil && errors.IsTimeout(err) {
		// configmap is created but its initialization has timed out.
		// If the initialization is successful eventually, the
		// controller will observe the creation via the informer.
		// If the initialization fails, or if the configmap keeps
		// uninitialized for a long time, the informer will not
		// receive any update, and the controller will create a new
		// configmap when the expectation expires.
		succeededConfigMapCreationCount.Inc()
		return configMap, nil
	} else if err != nil {
		// Since error occurred(the informer won't observe this configmap),
		// we decrement the expected number of creates
		// and wait until next reconciliation
		expectations.CreationObserved(expectationConfigmapsKey)
		failedConfigMapCreationCount.Inc()
		return nil, err
	}
	succeededConfigMapCreationCount.Inc()
	return configMap, nil
}
