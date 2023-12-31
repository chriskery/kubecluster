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

package main

import (
	"context"
	"errors"
	"flag"
	kubeclusterorgv1alpha1 "github.com/chriskery/kubecluster/apis/kubecluster.org/v1alpha1"
	"github.com/chriskery/kubecluster/pkg/controller/cluster_schema"
	"github.com/chriskery/kubecluster/pkg/controller/ctrlcommon"
	"os"
	"strings"
	volcanoclient "volcano.sh/apis/pkg/client/clientset/versioned"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	"github.com/chriskery/kubecluster/pkg/controller"
	//+kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(kubeclusterorgv1alpha1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	var enabledSchemes cluster_schema.EnabledSchemes
	var controllerThreads int
	var gangSchedulerName string
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metrics endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.Var(&enabledSchemes, "enable-scheme", "Enable scheme(s) as --enable-scheme=tfcluster --enable-scheme=pytorchcluster, case insensitive."+
		" Now supporting TFCluster, PyTorchCluster, MXNetCluster, XGBoostCluster, PaddleCluster. By default, all supported schemes will be enabled.")
	flag.IntVar(&controllerThreads, "controller-threads", 1, "Number of worker threads used by the controller.")
	flag.StringVar(&gangSchedulerName, "gang-scheduler-name", "", "Now Supporting volcano and scheduler-plugins."+
		" Note: If you set another scheduler name, the training-operator assumes it's the scheduler-plugins.")
	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		Metrics:                metricsserver.Options{BindAddress: metricsAddr},
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "33b15c27",
		// LeaderElectionReleaseOnCancel defines if the leader should step down voluntarily
		// when the Manager ends. This requires the binary to immediately end when the
		// Manager is stopped, otherwise, this setting is unsafe. Setting this significantly
		// speeds up voluntary leader transitions as the new leader don't have to wait
		// LeaseDuration time first.
		//
		// In the default scaffold provided, the program ends immediately after
		// the manager stops, so would be fine to enable this option. However,
		// if you are doing or is intended to do any operation such as perform cleanups
		// after the manager stops then its usage might be unsafe.
		// LeaderElectionReleaseOnCancel: true,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	setupLog.Info("registering controllers...")

	setupController(mgr, enabledSchemes, gangSchedulerName, controllerThreads)
	//+kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

func setupController(mgr ctrl.Manager, enabledSchemes cluster_schema.EnabledSchemes, gangSchedulerName string, controllerThreads int) {
	// Prepare GangSchedulingSetupFunc
	gangSchedulingSetupFunc := ctrlcommon.GenNonGangSchedulerSetupFunc()
	if strings.EqualFold(gangSchedulerName, string(ctrlcommon.GangSchedulerVolcano)) {
		cfg := mgr.GetConfig()
		volcanoClientSet := volcanoclient.NewForConfigOrDie(cfg)
		gangSchedulingSetupFunc = ctrlcommon.GenVolcanoSetupFunc(volcanoClientSet)
	} else if gangSchedulerName != "" {
		gangSchedulingSetupFunc = ctrlcommon.GenSchedulerPluginsSetupFunc(mgr.GetClient(), gangSchedulerName)
	}

	reconciler := controller.NewReconciler(mgr, gangSchedulingSetupFunc)

	// TODO: We need a general manager. all rest util addsToManager
	// Based on the user configuration, we start different controllers
	if enabledSchemes.Empty() {
		enabledSchemes.FillAll()
	}
	errMsg := "failed to set up controller"
	for _, s := range enabledSchemes {
		schemaFactory, supported := cluster_schema.SupportedClusterSchemaReconciler[s]
		if !supported {
			setupLog.Error(errors.New(errMsg), "cluster scheme is not supported", "scheme", s)
			os.Exit(1)
		}
		schemaReconciler, err := schemaFactory(context.TODO(), mgr)
		if err != nil {
			setupLog.Error(errors.New(errMsg), "unable to create schema", "scheme", s)
			os.Exit(1)
		}
		if err = reconciler.ClusterController.RegisterSchema(s, schemaReconciler); err != nil {
			setupLog.Error(errors.New(errMsg), "unable to register schema", "scheme", s)
			os.Exit(1)
		}
	}
	if err := reconciler.SetupWithManager(mgr, controllerThreads); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "KubeCluster")
		os.Exit(1)
	}
}
