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
	"github.com/go-logr/logr"
	"github.com/kubecluster/apis/kubecluster.org/v1alpha1"
	"github.com/kubecluster/pkg/common/util"
	"github.com/kubecluster/pkg/controller/cluster_schema"
	"github.com/kubecluster/pkg/controller/common"
	"github.com/kubecluster/pkg/controller/control"
	"github.com/kubecluster/pkg/controller/expectation"
	util2 "github.com/kubecluster/pkg/util"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
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

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	commonmetrics "github.com/kubecluster/pkg/metrics"
	schedulerpluginsv1alpha1 "sigs.k8s.io/scheduler-plugins/apis/scheduling/v1alpha1"

	"volcano.sh/apis/pkg/apis/scheduling/v1beta1"
)

const controllerName = "kubecluster-controller"

func NewReconciler(mgr manager.Manager, gangSchedulingSetupFunc common.GangSchedulingSetupFunc) *KubeClusterReconciler {
	r := &KubeClusterReconciler{
		Client:    mgr.GetClient(),
		Scheme:    mgr.GetScheme(),
		recorder:  mgr.GetEventRecorderFor(controllerName),
		apiReader: mgr.GetAPIReader(),
		Log:       log.Log,
	}

	cfg := mgr.GetConfig()
	kubeClientSet := kubeclientset.NewForConfigOrDie(cfg)
	sharedInformers := informers.NewSharedInformerFactory(kubeClientSet, 0)
	priorityClassInformer := sharedInformers.Scheduling().V1().PriorityClasses()

	r.ClusterController = common.ClusterController{
		Controller:                  r,
		Expectations:                expectation.NewControllerExpectations(),
		Recorder:                    r.recorder,
		KubeClientSet:               kubeClientSet,
		PriorityClassLister:         priorityClassInformer.Lister(),
		PriorityClassInformerSynced: priorityClassInformer.Informer().HasSynced,
		PodControl:                  control.RealPodControl{KubeClient: kubeClientSet, Recorder: r.recorder},
		ServiceControl:              control.RealServiceControl{KubeClient: kubeClientSet, Recorder: r.recorder},
		SchemaReconcilerManager:     make(map[cluster_schema.ClusterSchema]cluster_schema.ClusterSchemaReconciler),
	}

	gangSchedulingSetupFunc(&r.ClusterController)

	return r
}

// KubeClusterReconciler reconciles a KubeCluster object
type KubeClusterReconciler struct {
	common.ClusterController
	client.Client
	Scheme *runtime.Scheme

	recorder  record.EventRecorder
	apiReader client.Reader
	Log       logr.Logger
}

//+kubebuilder:rbac:groups=kubecluster.org,resources=clusters,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=kubecluster.org,resources=clusters/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=kubecluster.org,resources=clusters/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the KubeCluster object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.16.0/pkg/reconcile
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
		r.Recorder.Eventf(kcluster, corev1.EventTypeWarning, util2.NewReason(v1alpha1.KubeClusterKind, util2.ClusterFailedValidationReason),
			"KubeCLuster failed validation because %s", err)
		return ctrl.Result{}, err
	}

	// Check if reconciliation is needed
	clusterKey, err := common.KeyFunc(kcluster)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("couldn't get clusterKey for kubecluster object %#v: %v", kcluster, err))
	}

	replicaTypes := util.GetReplicaTypes(kcluster.Spec.ClusterReplicaSpec)
	needReconcile := util.SatisfiedExpectations(r.Expectations, clusterKey, replicaTypes)

	if !needReconcile || kcluster.GetDeletionTimestamp() != nil {
		logger.Info("reconcile cancelled, kubecluster does not need to do reconcile or has been deleted",
			"sync", needReconcile, "deleted", kcluster.GetDeletionTimestamp() != nil)
		return ctrl.Result{}, nil
	}

	kcluster = kcluster.DeepCopy()
	// Set default priorities to kubecluster
	r.Scheme.Default(kcluster)

	if err = r.ReconcileKubeCluster(kcluster); err != nil {
		logrus.Warnf("Reconcile Kube CLuster error %v", err)
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *KubeClusterReconciler) SetupWithManager(mgr ctrl.Manager, controllerThreads int) error {
	c, err := controller.New(r.ControllerName(), mgr, controller.Options{
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
		CreateFunc: util.OnDependentCreateFunc(r.Expectations),
		UpdateFunc: util.OnDependentUpdateFunc(&r.ClusterController),
		DeleteFunc: util.OnDependentDeleteFunc(r.Expectations),
	}
	// Create generic predicates
	genericPredicates := predicate.Funcs{
		CreateFunc: util.OnDependentCreateFuncGeneric(r.Expectations),
		UpdateFunc: util.OnDependentUpdateFuncGeneric(&r.ClusterController),
		DeleteFunc: util.OnDependentDeleteFuncGeneric(r.Expectations),
	}
	// inject watching for job related pod
	if err = c.Watch(source.Kind(mgr.GetCache(), &corev1.Pod{}), eventHandler, predicates); err != nil {
		return err
	}
	// inject watching for job related service
	if err = c.Watch(source.Kind(mgr.GetCache(), &corev1.Service{}), eventHandler, predicates); err != nil {
		return err
	}
	// skip watching volcano PodGroup if volcano PodGroup is not installed
	if _, err = mgr.GetRESTMapper().RESTMapping(schema.GroupKind{Group: v1beta1.GroupName, Kind: "PodGroup"},
		v1beta1.SchemeGroupVersion.Version); err == nil {
		// inject watching for job related volcano PodGroup
		if err = c.Watch(source.Kind(mgr.GetCache(), &v1beta1.PodGroup{}), eventHandler, genericPredicates); err != nil {
			return err
		}
	}
	// skip watching scheduler-plugins PodGroup if scheduler-plugins PodGroup is not installed
	if _, err = mgr.GetRESTMapper().RESTMapping(schema.GroupKind{Group: schedulerpluginsv1alpha1.SchemeGroupVersion.Group, Kind: "PodGroup"},
		schedulerpluginsv1alpha1.SchemeGroupVersion.Version); err == nil {
		// inject watching for job related scheduler-plugins PodGroup
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
		util2.UpdateClusterConditions(&kcluster.Status, v1alpha1.ClusterCreated, corev1.ConditionTrue,
			util2.NewReason(v1alpha1.KubeClusterKind, util2.ClusterCreatedReason), msg)
		return true
	}
}

func (r *KubeClusterReconciler) ControllerName() string {
	return controllerName
}

func (r *KubeClusterReconciler) GetAPIGroupVersionKind() schema.GroupVersionKind {
	return v1alpha1.GroupVersion.WithKind(v1alpha1.KubeClusterKind)
}

func (r *KubeClusterReconciler) GetAPIGroupVersion() schema.GroupVersion {
	return v1alpha1.GroupVersion
}

func (r *KubeClusterReconciler) GetPodsForCluster(kcluster *v1alpha1.KubeCluster) ([]*corev1.Pod, error) {
	//TODO implement me
	panic("implement me")
}

func (r *KubeClusterReconciler) GetServicesForCluster(kcluster *v1alpha1.KubeCluster) ([]*corev1.Service, error) {
	//TODO implement me
	panic("implement me")
}

func (r *KubeClusterReconciler) GetConfigMapsForCluster(kcluster *v1alpha1.KubeCluster) ([]*corev1.ConfigMap, error) {
	//TODO implement me
	panic("implement me")
}

func (r *KubeClusterReconciler) DeleteCluster(kcluster metav1.Object) error {
	//TODO implement me
	panic("implement me")
}

func (r *KubeClusterReconciler) UpdateClusterStatusInApiServer(kcluster metav1.Object, clusterStatus *v1alpha1.ClusterStatus) error {
	//TODO implement me
	panic("implement me")
}

func (r *KubeClusterReconciler) UpdateClusterStatus(kcluster metav1.Object, replicas map[v1alpha1.ReplicaType]*v1alpha1.ReplicaSpec, clusterStatus *v1alpha1.ClusterStatus) error {
	//TODO implement me
	panic("implement me")
}

func (r *KubeClusterReconciler) ReconcilePods(kcluster metav1.Object, clusterStatus *v1alpha1.ClusterStatus, pods []*corev1.Pod, rtype v1alpha1.ReplicaType, spec *v1alpha1.ReplicaSpec) error {
	//TODO implement me
	panic("implement me")
}

func (r *KubeClusterReconciler) ReconcileServices(kcluster metav1.Object, clusterStatus *v1alpha1.ClusterStatus, services []*corev1.Service, rtype v1alpha1.ReplicaType, spec *v1alpha1.ReplicaSpec) error {
	//TODO implement me
	panic("implement me")
}

func (r *KubeClusterReconciler) GetClusterFromInformerCache(namespace, name string) (metav1.Object, error) {
	//TODO implement me
	panic("implement me")
}
