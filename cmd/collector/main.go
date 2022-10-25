package main

import (
	"flag"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"sdase.org/collector/internal/cmd"
	"sdase.org/collector/internal/cmd/imagecollector/collector"
	"sdase.org/collector/internal/cmd/imagecollector/model"
	"sdase.org/collector/internal/cmd/versioncollector"
	"sdase.org/collector/internal/pkg/kubeclient"
)

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.With().Caller().Logger()

	err := newCommand().Execute()
	cmd.CheckError(err)
}

var kubeconfig, kubecontext, masterURL, teamName, environmentName string
var scanIntervalInSecondsVersionCollector int64
var isVersionCollector, isImageCollector bool
var s3ParameterEntry = model.S3parameterEntry{}
var imageCollectorDefaults = model.ImageCollectorDefaults{}

func newCommand() *cobra.Command {

	c := &cobra.Command{
		Use:   "collector",
		Short: "Collect images, apps, and their versions.",
		Long: `collector is a tool that will scan 'Deployment's 'StatefulSet's and 'DaemonSet's 'Namespace's, and 'Pod's for version and image information and push these as metrics to Prometheus.
			Environment variables for image-collector:
				DEFAULT_SCAN_BASEIMAGE_LIFETIME
				DEFAULT_SCAN_DEPENDENCY_CHECK
				DEFAULT_SCAN_DEPENDENCY_TRACK
				DEFAULT_SCAN_DISTROLESS
				DEFAULT_SCAN_LIFETIME
				DEFAULT_SCAN_MALWARE
				DEFAULT_SCAN_NEW_VERSION
				DEFAULT_SCAN_RUN_AS_ROOT
				DEFAULT_SCAN_RUN_AS_PRIVILEGED`,
		Run: func(cmd *cobra.Command, args []string) {
			run()
		},
	}
	kubeclient.CreateClientOrDie(kubeconfig, kubecontext, masterURL)
	c.PersistentFlags().StringVar(&environmentName, "cluster-name", "", "Name of the team used for extracting data.")

	c.PersistentFlags().StringVar(&teamName, "team-name", "5xx", "Name of the team used for extracting data in the version collector.")
	c.PersistentFlags().StringVar(&kubeconfig, "kubeconfig", "", "absolute path to the kubeconfig file")
	c.PersistentFlags().StringVar(&kubecontext, "kubecontext", "", "The context to use to talk to the Kubernetes apiserver. If unset defaults to whatever your current-context is (kubectl config current-context)")
	c.PersistentFlags().StringVar(&masterURL, "master", "", "URL of the API server")
	cmd.CheckError(c.MarkPersistentFlagRequired("cluster-name"))

	c.PersistentFlags().BoolVar(&isVersionCollector, "is-versioncollector", true, "Enable the version collector")
	c.PersistentFlags().Int64Var(&scanIntervalInSecondsVersionCollector, "scan-interval-versioncollector", 60, "Rescan intervalInSeconds in seconds for version collector")

	c.PersistentFlags().BoolVar(&isImageCollector, "is-imagecollector", true, "Enable the image collector")
	c.PersistentFlags().Int64Var(&imageCollectorDefaults.ScanIntervalInSeconds, "image-collector-scan-interval", 3600, "Rescan intervalInSeconds in seconds for image collector")
	c.PersistentFlags().StringVar(&imageCollectorDefaults.ConfigBasePath, "image-collector-config-basepath", "/config", "Configuration folder for the image collector")
	c.PersistentFlags().BoolVar(&imageCollectorDefaults.Skip, "image-collector-default-skip", false, "Images in namespaces are skipped without annotations/labels")
	c.PersistentFlags().BoolVar(&imageCollectorDefaults.IsSaveFiles, "image-collector-save-files", false, "In addition to uploading the files to S3, store the files on the disk")

	c.PersistentFlags().StringVar(&s3ParameterEntry.S3bucket, "image-collector-s3-bucket", "cluster-image-scanner-collector", "S3 Bucket to store image collector results")
	c.PersistentFlags().StringVar(&s3ParameterEntry.S3accessKey, "image-collector-s3-access-key", "", "S3 Access Key")
	c.PersistentFlags().StringVar(&s3ParameterEntry.S3secretKey, "image-collector-s3-secret-key", "", "S3 Secret Key")
	c.PersistentFlags().StringVar(&s3ParameterEntry.S3endpoint, "image-collector-s3-endpoint", "", "S3 Endpoint (e.g. minio)")
	c.PersistentFlags().StringVar(&s3ParameterEntry.S3region, "image-collector-s3-region", "eu-west-1", "S3 region")
	c.PersistentFlags().BoolVar(&s3ParameterEntry.S3insecure, "image-collector-s3-insecure", false, "Insecure bucket connection")
	c.PersistentFlags().BoolVar(&s3ParameterEntry.S3ForcePathStyle, "image-collector-s3-force-path-style", false, "Enforce S3 Force Path Style (should be true for minio)")

	var isDebug = false
	c.PersistentFlags().BoolVar(&isDebug, "debug", false, "Set logging level to debug, default logging level is info")
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if isDebug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	c.PersistentFlags().AddGoFlagSet(flag.CommandLine)
	return c
}

func run() {
	client := kubeclient.CreateClientOrDie(kubeconfig, kubecontext, masterURL)
	imageCollectorDefaults.Client = client
	go versioncollector.Run(isVersionCollector, teamName, environmentName, client, scanIntervalInSecondsVersionCollector)
	imageCollectorDefaults.Environment = environmentName
	go collector.Run(isImageCollector, imageCollectorDefaults, s3ParameterEntry)

	// Expose the registered metrics via HTTP.
	http.Handle("/metrics", promhttp.HandlerFor(
		prometheus.DefaultGatherer,
		promhttp.HandlerOpts{
			// Opt into OpenMetrics to support exemplars.
			EnableOpenMetrics: true,
		},
	))
	err := http.ListenAndServe(":9402", nil)
	log.Fatal().Stack().Err(err).Msg("Could not start listener for version collector")

}
