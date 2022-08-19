package collector

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type MetricProviderImpl struct {
	teamName       string
	clusterName    string
	resultProvider ResultProvider
	appMetric      *prometheus.GaugeVec
	helmMetric     *prometheus.GaugeVec
}

func (mp *MetricProviderImpl) Start(scanIntervalInSeconds int64) {
	go func() {
		for {
			result := mp.resultProvider()
			for _, entry := range result.Entries {
				if entry.AppVersion != nil {
					mp.appMetric.WithLabelValues(entry.Name, "major").Set(float64(entry.AppVersion.Major))
					mp.appMetric.WithLabelValues(entry.Name, "minor").Set(float64(entry.AppVersion.Minor))
					mp.appMetric.WithLabelValues(entry.Name, "patch").Set(float64(entry.AppVersion.Patch))
				}

				if entry.HelmVersion != nil {
					mp.helmMetric.WithLabelValues(entry.Name, "major").Set(float64(entry.HelmVersion.Major))
					mp.helmMetric.WithLabelValues(entry.Name, "minor").Set(float64(entry.HelmVersion.Minor))
					mp.helmMetric.WithLabelValues(entry.Name, "patch").Set(float64(entry.HelmVersion.Patch))
				}
			}
			time.Sleep(time.Duration(scanIntervalInSeconds) * time.Second)
		}
	}()
}

func (mp *MetricProviderImpl) registerMetrics(teamName string, clusterName string) {
	mp.appMetric = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "version_current_app",
			Help: "Specifies the semantic version of the app",
			ConstLabels: prometheus.Labels{
				"cluster_name": clusterName,
				"team":         teamName,
			},
		},
		[]string{"app", "type"},
	)

	mp.helmMetric = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "version_current_helm",
			Help: "Specifies the semantic version of the helm chart",
			ConstLabels: prometheus.Labels{
				"cluster_name": clusterName,
				"team":         teamName,
			},
		},
		[]string{"app", "type"},
	)
}
