/*
Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"fmt"
	"github.com/chriskery/kubecluster/apis/kubecluster.org/v1alpha1"
	"github.com/chriskery/kubecluster/pkg/common"
	util3 "github.com/chriskery/kubecluster/pkg/common/util"
	"github.com/chriskery/kubecluster/pkg/controller/cluster_schema"
	"github.com/chriskery/kubecluster/pkg/controller/control"
	"github.com/chriskery/kubecluster/pkg/controller/ctrlcommon"
	"github.com/chriskery/kubecluster/pkg/core"
	"github.com/chriskery/kubecluster/pkg/util"
	utillabels "github.com/chriskery/kubecluster/pkg/util/labels"
	"github.com/go-logr/logr"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/informers"
	kubeclientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/source"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	commonmetrics "github.com/chriskery/kubecluster/pkg/metrics"
	schedulerpluginsv1alpha1 "sigs.k8s.io/scheduler-plugins/apis/scheduling/v1alpha1"

	"volcano.sh/apis/pkg/apis/scheduling/v1beta1"
)

const (
	// ErrResourceDoesNotExist is used as part of the Event 'reason' when some
	// resource is missing in yaml
	ErrResourceDoesNotExist = "ErrResourceDoesNotExist"
	// MessageResourceDoesNotExist is used for Events when some
	// resource is missing in yaml
	MessageResourceDoesNotExist = "Resource %q is missing in yaml"
	// ErrResourceExists is used as part of the Event 'reason' when an SlurmClusterCluster
	// fails to sync due to dependent resources of the same name already
	// existing.
	ErrResourceExists = "ErrResourceExists"
	// MessageResourceExists is the message used for Events when a resource
	// fails to sync due to dependent resources already existing.
	MessageResourceExists = "Resource %q of SlurmClusterClusterKind %q already exists and is not managed by SlurmClusterCluster"
)

func NewReconciler(mgr manager.Manager, gangSchedulingSetupFunc ctrlcommon.GangSchedulingSetupFunc) *KubeClusterReconciler {
	r := &KubeClusterReconciler{
		Client:    mgr.GetClient(),
		Scheme:    mgr.GetScheme(),
		recorder:  mgr.GetEventRecorderFor(common.ControllerName),
		apiReader: mgr.GetAPIReader(),
		Log:       log.Log,
	}

	cfg := mgr.GetConfig()
	kubeClientSet := kubeclientset.NewForConfigOrDie(cfg)
	sharedInformers := informers.NewSharedInformerFactory(kubeClientSet, 0)
	priorityClassInformer := sharedInformers.Scheduling().V1().PriorityClasses()

	r.ClusterController = ctrlcommon.ClusterController{
		Controller:                  r,
		Recorder:                    r.recorder,
		KubeClientSet:               kubeClientSet,
		PriorityClassLister:         priorityClassInformer.Lister(),
		PriorityClassInformerSynced: priorityClassInformer.Informer().HasSynced,
		PodControl:                  control.RealPodControl{KubeClient: kubeClientSet, Recorder: r.recorder},
		ServiceControl:              control.RealServiceControl{KubeClient: kubeClientSet, Recorder: r.recorder},
		ConfigMapControl:            control.RealConfigMapControl{KubeClient: kubeClientSet, Recorder: r.recorder},
		SchemaReconcilerManager:     make(map[cluster_schema.ClusterSchema]common.ClusterSchemaReconciler),
		WorkQueue:                   &util3.FakeWorkQueue{},
	}

	gangSchedulingSetupFunc(&r.ClusterController)

	return r
}

// KubeClusterReconciler reconciles a KubeCluster object
type KubeClusterReconciler struct {
	ctrlcommon.ClusterController
	client.Client
	Scheme *runtime.Scheme

	recorder  record.EventRecorder
	apiReader client.Reader
	Log       logr.Logger
}

func (r *KubeClusterReconciler) GetClusterFromInformerCache(namespace, name string) (metav1.Object, error) {
	kcluster := &v1alpha1.KubeCluster{}
	err := r.Get(context.TODO(), types.NamespacedName{
		Namespace: namespace, Name: name,
	}, kcluster)
	return kcluster, err
}

func (r *KubeClusterReconciler) UpdateClusterStatusInApiServer(
	metaObject metav1.Object,
	clusterStatus *v1alpha1.ClusterStatus,
) error {
	if clusterStatus.ReplicaStatuses == nil {
		clusterStatus.ReplicaStatuses = map[v1alpha1.ReplicaType]*v1alpha1.ReplicaStatus{}
	}

	kcluster, ok := metaObject.(*v1alpha1.KubeCluster)
	if !ok {
		return fmt.Errorf("%+v is not a type of KubeCLuster", metaObject)
	}
	common.ClearGeneratedFields(&kcluster.ObjectMeta)

	// Cluster status passed in differs with status in cluster, update in basis of the passed in one.
	if !equality.Semantic.DeepEqual(&kcluster.Status, clusterStatus) {
		kcluster = kcluster.DeepCopy()
		kcluster.Status = *clusterStatus.DeepCopy()
	}

	result := r.Status().Update(context.Background(), kcluster)

	if result != nil {
		r.Log.WithValues("kcluster", types.NamespacedName{
			Namespace: kcluster.GetNamespace(),
			Name:      kcluster.GetName(),
		})
		return result
	}

	return nil
}

func (r *KubeClusterReconciler) UpdateClusterStatus(
	kcluster *v1alpha1.KubeCluster,
	replicas map[v1alpha1.ReplicaType]*v1alpha1.ReplicaSpec,
	clusterStatus *v1alpha1.ClusterStatus,
) error {
	clusterKey, err := ctrlcommon.KeyFunc(kcluster)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("couldn't get key for kube cluster object %#v: %v", kcluster, err))
		return err
	}

	logger := util.LoggerForCluster(kcluster)

	// Set StartTime.
	if clusterStatus.StartTime == nil {
		now := metav1.Now()
		clusterStatus.StartTime = &now
		// clusterStatus a sync to check if cluster past ActiveDeadlineSeconds
		if kcluster.Spec.RunPolicy.ActiveDeadlineSeconds != nil {
			logger.Infof("cluster with ActiveDeadlineSeconds will sync after %d seconds", *kcluster.Spec.RunPolicy.ActiveDeadlineSeconds)
			r.WorkQueue.AddAfter(clusterKey, time.Duration(*kcluster.Spec.RunPolicy.ActiveDeadlineSeconds)*time.Second)
		}
	}

	schemaReconciler := r.GetSchemaReconciler(kcluster.Spec.ClusterType)
	for rtype, spec := range replicas {
		schemaReconciler.UpdateClusterStatus(kcluster, clusterStatus, rtype, spec)
	}

	return nil
}

func (r *KubeClusterReconciler) UpdateConfigMapInApiServer(metaObject metav1.Object, configMap *corev1.ConfigMap) error {
	kcluster, ok := metaObject.(*v1alpha1.KubeCluster)
	common.ClearGeneratedFields(&kcluster.ObjectMeta)
	if !ok {
		return fmt.Errorf("%+v is not a type of KubeCLuster", metaObject)
	}

	result := r.ConfigMapControl.UpdateConfigMap(metaObject.GetNamespace(), configMap)
	if result != nil {
		r.Log.WithValues("kcluster", types.NamespacedName{
			Namespace: kcluster.GetNamespace(),
			Name:      kcluster.GetName(),
		}, "configmap update", types.NamespacedName{
			Namespace: configMap.GetNamespace(),
			Name:      configMap.GetName(),
		})
		return result
	}
	return nil
}

//+kubebuilder:rbac:groups=kubecluster.org,resources=kubeclusters,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=kubecluster.org,resources=kubeclusters/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=kubecluster.org,resources=kubeclusters/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;delete
//+kubebuilder:rbac:groups="",resources=configmaps,verbs=get;update;list;watch;create;delete
//+kubebuilder:rbac:groups=scheduling.volcano.sh,resources=podgroups,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=scheduling.x-k8s.io,resources=podgroups,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=events,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *KubeClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)
	logger := r.Log.WithValues(v1alpha1.KubeClusterSingular, req.NamespacedName)

	kcluster := &v1alpha1.KubeCluster{}
	err := r.Get(ctx, req.NamespacedName, kcluster)
	if err != nil {
		logger.Info(err.Error(), "unable to fetch kubecluster", req.NamespacedName.String())
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if err = v1alpha1.ValidateV1alphaCluster(kcluster); err != nil {
		r.Recorder.Eventf(kcluster, corev1.EventTypeWarning, util.NewReason(v1alpha1.KubeClusterKind, util.ClusterFailedValidationReason),
			"KubeCLuster failed validation because %s", err)
		return ctrl.Result{}, err
	}

	schemaReconciler := r.GetSchemaReconciler(kcluster.Spec.ClusterType)
	if err = schemaReconciler.ValidateV1KubeCluster(kcluster); err != nil {
		r.Recorder.Eventf(kcluster, corev1.EventTypeWarning, util.NewReason(v1alpha1.KubeClusterKind, util.ClusterFailedValidationReason),
			"KubeCluster failed validation because %s", err)
		return ctrl.Result{}, err
	}

	needReconcile := r.needReconcile(kcluster, schemaReconciler)
	if !needReconcile || kcluster.GetDeletionTimestamp() != nil {
		logger.Info("reconcile cancelled, kubecluster does not need to do reconcile or has been deleted",
			"sync", needReconcile, "deleted", kcluster.GetDeletionTimestamp() != nil)
		return ctrl.Result{}, nil
	}

	kcluster = kcluster.DeepCopy()
	// Set default priorities to kubecluster
	r.Scheme.Default(kcluster)
	schemaReconciler.Default(kcluster)

	if err = r.ReconcileKubeCluster(kcluster, schemaReconciler); err != nil {
		logrus.Warnf("Reconcile Kube Cluster error %v", err)
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *KubeClusterReconciler) needReconcile(kcluster *v1alpha1.KubeCluster, schemaReconciler common.ClusterSchemaReconciler) bool {
	// Check if reconciliation is needed
	clusterKey, err := ctrlcommon.KeyFunc(kcluster)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("couldn't get clusterKey for kubecluster object %#v: %v", kcluster, err))
	}
	replicaTypes := common.GetReplicaTypes(kcluster.Spec.ClusterReplicaSpec)
	needReconcile := !util3.SatisfiedExpectations(schemaReconciler, clusterKey, replicaTypes)
	return needReconcile
}

// SetupWithManager sets up the controller with the Manager.
func (r *KubeClusterReconciler) SetupWithManager(mgr ctrl.Manager, controllerThreads int) error {
	c, err := controller.New(common.ControllerName, mgr, controller.Options{
		Reconciler:              r,
		MaxConcurrentReconciles: controllerThreads,
	})
	if err != nil {
		return err
	}

	// using onOwnerCreateFunc is easier to set defaults
	if err = c.Watch(source.Kind(mgr.GetCache(), &v1alpha1.KubeCluster{}), &handler.EnqueueRequestForObject{},
		predicate.Funcs{CreateFunc: r.onOwnerCreateFunc()},
	); err != nil {
		return err
	}

	// eventHandler for owned objects
	eventHandler := handler.EnqueueRequestForOwner(mgr.GetScheme(), mgr.GetRESTMapper(), &v1alpha1.KubeCluster{}, handler.OnlyControllerOwner())
	predicates := predicate.Funcs{
		CreateFunc: util3.OnDependentCreateFunc(r.SchemaReconcilerManager),
		UpdateFunc: util3.OnDependentUpdateFunc(&r.ClusterController),
		DeleteFunc: util3.OnDependentDeleteFunc(r.SchemaReconcilerManager),
	}
	// Create generic predicates
	genericPredicates := predicate.Funcs{
		CreateFunc: util3.OnDependentCreateFuncGeneric(r.SchemaReconcilerManager),
		UpdateFunc: util3.OnDependentUpdateFuncGeneric(&r.ClusterController),
		DeleteFunc: util3.OnDependentDeleteFuncGeneric(r.SchemaReconcilerManager),
	}
	// inject watching for cluster related pod
	if err = c.Watch(source.Kind(mgr.GetCache(), &corev1.Pod{}), eventHandler, predicates); err != nil {
		return err
	}
	// inject watching for cluster related service
	if err = c.Watch(source.Kind(mgr.GetCache(), &corev1.Service{}), eventHandler, predicates); err != nil {
		return err
	}
	// skip watching volcano PodGroup if volcano PodGroup is not installed
	if _, err = mgr.GetRESTMapper().RESTMapping(schema.GroupKind{Group: v1beta1.GroupName, Kind: "PodGroup"},
		v1beta1.SchemeGroupVersion.Version); err == nil {
		// inject watching for cluster related volcano PodGroup
		if err = c.Watch(source.Kind(mgr.GetCache(), &v1beta1.PodGroup{}), eventHandler, genericPredicates); err != nil {
			return err
		}
	}
	// skip watching scheduler-plugins PodGroup if scheduler-plugins PodGroup is not installed
	if _, err = mgr.GetRESTMapper().RESTMapping(schema.GroupKind{Group: schedulerpluginsv1alpha1.SchemeGroupVersion.Group, Kind: "PodGroup"},
		schedulerpluginsv1alpha1.SchemeGroupVersion.Version); err == nil {
		// inject watching for cluster related scheduler-plugins PodGroup
		if err = c.Watch(source.Kind(mgr.GetCache(), &schedulerpluginsv1alpha1.PodGroup{}), eventHandler, genericPredicates); err != nil {
			return err
		}
	}
	return nil
}

// onOwnerCreateFunc modify creation condition.
func (r *KubeClusterReconciler) onOwnerCreateFunc() func(event.CreateEvent) bool {
	return func(e event.CreateEvent) bool {
		kcluster, ok := e.Object.(*v1alpha1.KubeCluster)
		if !ok {
			return true
		}

		r.Scheme.Default(kcluster)
		msg := fmt.Sprintf("Kubecluster %s is created.", e.Object.GetName())
		logrus.Info(msg)
		commonmetrics.CreatedclustersCounterInc(kcluster.Namespace, string(kcluster.Spec.ClusterType))
		util.UpdateClusterConditions(&kcluster.Status, v1alpha1.ClusterCreated, corev1.ConditionTrue,
			util.NewReason(v1alpha1.KubeClusterKind, util.ClusterCreatedReason), msg)
		return true
	}
}

func (r *KubeClusterReconciler) GetAPIGroupVersionKind() schema.GroupVersionKind {
	return v1alpha1.GroupVersion.WithKind(v1alpha1.KubeClusterKind)
}

func (r *KubeClusterReconciler) GetAPIGroupVersion() schema.GroupVersion {
	return v1alpha1.GroupVersion
}

// GetPodsForCluster returns the set of pods that this cluster should manage.
// It also reconciles ControllerRef by adopting/orphaning.
// Note that the returned Pods are pointers into the cache.
func (r *KubeClusterReconciler) GetPodsForCluster(kcluster *v1alpha1.KubeCluster) ([]*corev1.Pod, error) {
	// Create selector.
	selector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
		MatchLabels: utillabels.GenLabels(common.ControllerName, kcluster.GetName()),
	})

	if err != nil {
		return nil, fmt.Errorf("couldn't convert Cluster selector: %v", err)
	}
	// List all pods to include those that don't match the selector anymore
	// but have a ControllerRef pointing to this controller.
	podlist := &corev1.PodList{}
	err = r.List(context.Background(), podlist,
		client.MatchingLabelsSelector{Selector: selector}, client.InNamespace(kcluster.GetNamespace()))
	if err != nil {
		return nil, err
	}

	return common.KubeClusterControlledPodList(podlist.Items, kcluster), nil
}

func (r *KubeClusterReconciler) GetServicesForCluster(kcluster *v1alpha1.KubeCluster) ([]*corev1.Service, error) {
	// Create selector.
	selector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
		MatchLabels: utillabels.GenLabels(common.ControllerName, kcluster.GetName()),
	})

	if err != nil {
		return nil, fmt.Errorf("couldn't convert Cluster selector: %v", err)
	}
	// List all pods to include those that don't match the selector anymore
	// but have a ControllerRef pointing to this controller.
	serviceList := &corev1.ServiceList{}
	err = r.List(context.Background(), serviceList,
		client.MatchingLabelsSelector{Selector: selector}, client.InNamespace(kcluster.GetNamespace()))
	if err != nil {
		return nil, err
	}

	ret := common.ConvertServiceList(serviceList.Items)
	return ret, nil
}

func (r *KubeClusterReconciler) DeleteCluster(metaObject metav1.Object) error {
	kubecluster, ok := metaObject.(*v1alpha1.KubeCluster)
	if !ok {
		return fmt.Errorf("%+v is not a type of KubeCLustr", metaObject)
	}
	if err := r.Delete(context.Background(), kubecluster); err != nil {
		r.recorder.Eventf(kubecluster, corev1.EventTypeWarning, control.FailedDeletePodReason, "Error deleting: %v", err)
		logrus.Error(err, " failed to delete cluster ", " namespace ", kubecluster.Namespace, " name ", kubecluster.Name)
		return err
	}
	r.recorder.Eventf(kubecluster, corev1.EventTypeNormal, control.SuccessfulDeletePodReason, "Deleted cluster: %v", kubecluster.Name)
	logrus.Info("cluster deleted", "namespace", kubecluster.Namespace, "name", kubecluster.Name)
	commonmetrics.DeletedclustersCounterInc(kubecluster.Namespace, kubecluster.Spec.ClusterType)
	return nil
}

func (r *KubeClusterReconciler) GetSchemaReconciler(clusterType v1alpha1.ClusterType) common.ClusterSchemaReconciler {
	r.SchemaReconcilerMutux.Lock()
	defer r.SchemaReconcilerMutux.Unlock()
	return r.SchemaReconcilerManager[cluster_schema.ClusterSchema(clusterType)]
}

func (r *KubeClusterReconciler) FilterServicesForReplicaType(services []*corev1.Service, replicaType string) ([]*corev1.Service, error) {
	return core.FilterServicesForReplicaType(services, replicaType)
}

// GetServiceSlices returns a slice, which element is the slice of service.
// Assume the return object is serviceSlices, then serviceSlices[i] is an
// array of pointers to services corresponding to Services for replica i.
func (r *KubeClusterReconciler) GetServiceSlices(services []*corev1.Service, replicas int, logger *logrus.Entry) [][]*corev1.Service {
	return core.GetServiceSlices(services, replicas, logger)
}

func (r *KubeClusterReconciler) GetGroupNameLabelValue() string {
	return v1alpha1.GroupVersion.Group
}

func (r *KubeClusterReconciler) GetConfigMapForCluster(kcluster *v1alpha1.KubeCluster) (*corev1.ConfigMap, error) {
	configMap := &corev1.ConfigMap{}
	if err := r.Get(context.Background(), client.ObjectKey{Namespace: kcluster.Namespace, Name: kcluster.Name}, configMap); err != nil {
		r.Log.Info(err.Error(), "unable to fetch configMap for slurmClusterCluster", kcluster.Name)
		return nil, client.IgnoreNotFound(err)
	}
	// If the ConfigMap is not controlled by this SlurmClusterCluster resource, we
	// should log a warning to the event recorder and return.
	if !metav1.IsControlledBy(configMap, kcluster) {
		msg := fmt.Sprintf(MessageResourceExists, configMap.Name, configMap.Kind)
		r.Recorder.Event(kcluster, corev1.EventTypeWarning, ErrResourceExists, msg)
		return nil, fmt.Errorf(msg)
	}
	return configMap, nil
}

func (r *KubeClusterReconciler) ControllerName() string {
	return common.ControllerName
}

func (r *KubeClusterReconciler) GenLabelSelector(clusterName string,
	rtype v1alpha1.ReplicaType) *metav1.LabelSelector {
	labels := r.ClusterController.GenLabels(clusterName)
	labels[v1alpha1.ReplicaTypeLabel] = utillabels.GenReplicaTypeLabel(rtype)

	return &metav1.LabelSelector{
		MatchLabels: labels,
	}
}
