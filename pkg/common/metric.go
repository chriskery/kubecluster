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
	kubeclusterorgv1alpha1 "github.com/chriskery/kubecluster/apis/kubecluster.org/v1alpha1"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

// Define all the prometheus counters for all clusters
var (
	clustersCreatedCount = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "training_operator_clusters_created_total",
			Help: "Counts number of clusters created",
		},
		[]string{"cluster_namespace", "cluster_type"},
	)
	clustersDeletedCount = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "training_operator_clusters_deleted_total",
			Help: "Counts number of clusters deleted",
		},
		[]string{"cluster_namespace", "cluster_type"},
	)
	clustersFailedCount = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "training_operator_clusters_failed_total",
			Help: "Counts number of clusters failed",
		},
		[]string{"cluster_namespace", "cluster_type"},
	)
	clustersRestartedCount = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "training_operator_clusters_restarted_total",
			Help: "Counts number of clusters restarted",
		},
		[]string{"cluster_namespace", "cluster_type"},
	)
)

func init() {
	// Register custom metrics with the global prometheus registry
	metrics.Registry.MustRegister(clustersCreatedCount,
		clustersDeletedCount,
		clustersFailedCount,
		clustersRestartedCount)
}

func CreatedClustersCounterInc(clusterNamespace string, clusterType kubeclusterorgv1alpha1.ClusterType) {
	clustersCreatedCount.WithLabelValues(clusterNamespace, string(clusterType)).Inc()
}

func DeletedClustersCounterInc(clusterNamespace string, clusterType kubeclusterorgv1alpha1.ClusterType) {
	clustersDeletedCount.WithLabelValues(clusterNamespace, string(clusterType)).Inc()
}

func FailedClustersCounterInc(clusterNamespace string, clusterType kubeclusterorgv1alpha1.ClusterType) {
	clustersFailedCount.WithLabelValues(clusterNamespace, string(clusterType)).Inc()
}

func RestartedClustersCounterInc(clusterNamespace string, clusterType kubeclusterorgv1alpha1.ClusterType) {
	clustersRestartedCount.WithLabelValues(clusterNamespace, string(clusterType)).Inc()
}
