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
// limitations under the License

package common

import (
	kubeclusterorgv1alpha1 "github.com/kubecluster/apis/kubecluster.org/v1alpha1"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

// Define all the prometheus counters for all clusters
var (
	clusterCreatedCount = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kubeclusters_created_total",
			Help: "Counts number of clusters created",
		},
		[]string{"cluster_namespace", "cluster_type"},
	)
	clustersDeletedCount = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kubeclusters_deleted_total",
			Help: "Counts number of clusters deleted",
		},
		[]string{"cluster_namespace", "cluster_type"},
	)
	clustersSuccessfulCount = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kubeclusters_successful_total",
			Help: "Counts number of clusters successful",
		},
		[]string{"cluster_namespace", "cluster_type"},
	)
	clustersFailedCount = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kubeclusters_failed_total",
			Help: "Counts number of clusters failed",
		},
		[]string{"cluster_namespace", "cluster_schema"},
	)
	clustersRestartedCount = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kubeclusters_restarted_total",
			Help: "Counts number of clusters restarted",
		},
		[]string{"cluster_namespace", "cluster_type"},
	)
)

func init() {
	// Register custom metrics with the global prometheus registry
	metrics.Registry.MustRegister(
		clusterCreatedCount,
		clustersDeletedCount,
		clustersSuccessfulCount,
		clustersFailedCount,
		clustersRestartedCount)
}

func CreatedclustersCounterInc(cluster_namespace, cluster_type string) {
	clustersRestartedCount.WithLabelValues(cluster_namespace, cluster_type).Inc()
}

func DeletedclustersCounterInc(cluster_namespace string, cluster_type kubeclusterorgv1alpha1.ClusterType) {
	clustersDeletedCount.WithLabelValues(cluster_namespace, string(cluster_type)).Inc()
}

func SuccessfulclustersCounterInc(cluster_namespace, cluster_type string) {
	clustersSuccessfulCount.WithLabelValues(cluster_namespace, cluster_type).Inc()
}

func FailedclustersCounterInc(cluster_namespace, cluster_type string) {
	clustersFailedCount.WithLabelValues(cluster_namespace, cluster_type).Inc()
}

func RestartedclustersCounterInc(cluster_namespace, cluster_type string) {
	clustersRestartedCount.WithLabelValues(cluster_namespace, cluster_type).Inc()
}
