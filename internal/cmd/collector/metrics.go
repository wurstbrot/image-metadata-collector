package collector

type ResultProvider func() *Result

type MetricProvider interface {
	Start(scanIntervalInSeconds int64)
}

func NewMetricProvider(teamName string, clusterName string, resultProvider ResultProvider) MetricProvider {
	provider := &MetricProviderImpl{
		teamName:       teamName,
		clusterName:    clusterName,
		resultProvider: resultProvider,
	}
	provider.registerMetrics(teamName, clusterName)
	return provider
}
