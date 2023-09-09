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
	"github.com/kubecluster/pkg/common/util"
	"github.com/kubecluster/pkg/controller/common"
	"github.com/kubecluster/pkg/controller/control"
	"github.com/kubecluster/pkg/controller/expectation"
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

	kubeclusterorgv1alpha1 "github.com/kubecluster/api/v1alpha1"
	commonmetrics "github.com/kubecluster/pkg/metrics"
	schedulerpluginsv1alpha1 "sigs.k8s.io/scheduler-plugins/apis/scheduling/v1alpha1"

	"volcano.sh/apis/pkg/apis/scheduling/v1beta1"
)

const controllerName = "kubecluster-controller"

func NewReconciler(mgr manager.Manager, gangSchedulingSetupFunc common.GangSchedulingSetupFunc) *ClusterReconciler {
	r := &ClusterReconciler{
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
	}

	gangSchedulingSetupFunc(&r.ClusterController)

	return r
}

// ClusterReconciler reconciles a KubeCluster object
type ClusterReconciler struct {
	common.ClusterController
	client.Client
	Scheme *runtime.Scheme

	recorder  record.EventRecorder
	apiReader client.Reader
	Log       logr.Logger
}

func (r *ClusterReconciler) GetClusterFromInformerCache(namespace, name string) (metav1.Object, error) {
	//TODO implement me
	panic("implement me")
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
func (r *ClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)
	logger := r.Log.WithValues(kubeclusterorgv1alpha1.KubeClusterSingular, req.NamespacedName)

	kcluster := &kubeclusterorgv1alpha1.KubeCluster{}
	err := r.Get(ctx, req.NamespacedName, kcluster)
	if err != nil {
		logger.Info(err.Error(), "unable to fetch kubecluster", req.NamespacedName.String())
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if err = kubeclusterorgv1alpha1.ValidateV1alphaCluster(kcluster); err != nil {
		r.Recorder.Eventf(kcluster, corev1.EventTypeWarning, util.NewReason(kubeclusterorgv1alpha1.KubeClusterKind, util.ClusterFailedValidationReason),
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

	// Set default priorities to tfjob
	r.Scheme.Default(kcluster)

	// Use common to reconcile the job related pod and service
	err = r.ReconcileJobs(tfjob, tfjob.Spec.TFReplicaSpecs, tfjob.Status, &tfjob.Spec.RunPolicy)
	if err != nil {
		logrus.Warnf("Reconcile Tensorflow Job error %v", err)
		return ctrl.Result{}, err
	}

	t, err := util.DurationUntilExpireTime(&tfjob.Spec.RunPolicy, tfjob.Status)
	if err != nil {
		logrus.Warnf("Reconcile Tensorflow Job error %v", err)
		return ctrl.Result{}, err
	}
	if t >= 0 {
		return ctrl.Result{Requeue: true, RequeueAfter: t}, nil
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ClusterReconciler) SetupWithManager(mgr ctrl.Manager, controllerThreads int) error {
	c, err := controller.New(r.ControllerName(), mgr, controller.Options{
		Reconciler:              r,
		MaxConcurrentReconciles: controllerThreads,
	})
	if err != nil {
		return err
	}

	// using onOwnerCreateFunc is easier to set defaults
	if err = c.Watch(source.Kind(mgr.GetCache(), &kubeclusterorgv1alpha1.KubeCluster{}), &handler.EnqueueRequestForObject{},
		predicate.Funcs{CreateFunc: r.onOwnerCreateFunc()},
	); err != nil {
		return err
	}

	// eventHandler for owned objects
	eventHandler := handler.EnqueueRequestForOwner(mgr.GetScheme(), mgr.GetRESTMapper(), &kubeclusterorgv1alpha1.KubeCluster{}, handler.OnlyControllerOwner())
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
func (r *ClusterReconciler) onOwnerCreateFunc() func(event.CreateEvent) bool {
	return func(e event.CreateEvent) bool {
		kcluster, ok := e.Object.(*kubeclusterorgv1alpha1.KubeCluster)
		if !ok {
			return true
		}

		r.Scheme.Default(kcluster)
		msg := fmt.Sprintf("Kubecluster %s is created.", e.Object.GetName())
		logrus.Info(msg)
		commonmetrics.CreatedclustersCounterInc(kcluster.Namespace, string(kcluster.Spec.ClusterType))
		util.UpdateClusterConditions(&kcluster.Status, kubeclusterorgv1alpha1.ClusterCreated, corev1.ConditionTrue,
			util.NewReason(kubeclusterorgv1alpha1.KubeClusterKind, util.ClusterCreatedReason), msg)
		return true
	}
}

func (r *ClusterReconciler) ControllerName() string {
	return controllerName
}

func (r *ClusterReconciler) GetAPIGroupVersionKind() schema.GroupVersionKind {
	return kubeclusterorgv1alpha1.GroupVersion.WithKind(kubeclusterorgv1alpha1.KubeClusterKind)
}

func (r *ClusterReconciler) GetAPIGroupVersion() schema.GroupVersion {
	return kubeclusterorgv1alpha1.GroupVersion
}
