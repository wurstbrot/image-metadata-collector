package collector

import (
	"flag"

	"sda.se/version-collector/internal/cmd"

	"github.com/spf13/cobra"
	"k8s.io/klog"
	"sda.se/version-collector/internal/pkg/kubeclient"
)

func NewCommand() *cobra.Command {
	var kubeconfig, kubecontext, masterURL, teamName, clusterName string
	var scanIntervalInSeconds int64

	c := &cobra.Command{
		Use:   "sdase-version-collector",
		Short: "Collect apps and their versions.",
		Long:  `sdase-version-collector is a tool that will scan 'Deployment's 'StatefulSet's and 'DaemonSet's for version information and push these as metrics to Prometheus..`,
		Run: func(cmd *cobra.Command, args []string) {
			run(teamName, clusterName, kubeconfig, kubecontext, masterURL, scanIntervalInSeconds)
		},
	}

	c.PersistentFlags().StringVar(&clusterName, "cluster-name", "", "Name of the team used for extracting data.")
	c.PersistentFlags().StringVar(&teamName, "team-name", "5xx", "Name of the team used for extracting data.")
	c.PersistentFlags().StringVar(&kubeconfig, "kubeconfig", "", "absolute path to the kubeconfig file")
	c.PersistentFlags().StringVar(&kubecontext, "kubecontext", "", "The context to use to talk to the Kubernetes apiserver. If unset defaults to whatever your current-context is (kubectl config current-context)")
	c.PersistentFlags().StringVar(&masterURL, "master", "", "URL of the API server")
	c.PersistentFlags().Int64Var(&scanIntervalInSeconds, "scan-interval", 60, "Rescan intervalInSeconds in seconds")
	cmd.CheckError(c.MarkPersistentFlagRequired("cluster-name"))

	// init and add the klog flags
	klog.InitFlags(flag.CommandLine)
	c.PersistentFlags().AddGoFlagSet(flag.CommandLine)

	return c
}

func run(teamName, collectorName, kubeconfig, kubecontext, masterURL string, scanIntervalInSeconds int64) {
	collector := NewCollector(teamName, kubeclient.CreateClientOrDie(kubeconfig, kubecontext, masterURL))
	NewMetricProvider(teamName, collectorName, createResultProvider(collector)).Start(scanIntervalInSeconds)
}

func createResultProvider(collector Collector) ResultProvider {
	return func() *Result {
		result := collector.Execute()
		for _, entry := range result.Entries {
			klog.Info(entry)
		}
		return result
	}
}
