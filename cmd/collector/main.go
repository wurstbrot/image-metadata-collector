package main

import (
	"log"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"k8s.io/klog"
	"sda.se/version-collector/internal/cmd"
	"sda.se/version-collector/internal/cmd/collector"
)

func main() {
	defer klog.Flush()

	err := collector.NewCommand().Execute()
	cmd.CheckError(err)
	// Expose the registered metrics via HTTP.
	http.Handle("/metrics", promhttp.HandlerFor(
		prometheus.DefaultGatherer,
		promhttp.HandlerOpts{
			// Opt into OpenMetrics to support exemplars.
			EnableOpenMetrics: true,
		},
	))
	log.Fatal(http.ListenAndServe(":9402", nil))
}
