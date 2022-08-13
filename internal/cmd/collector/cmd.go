package collector

import (
	"flag"

	"github.com/spf13/cobra"
	"k8s.io/klog"
	"sda.se/version-collector/internal/pkg/kubeclient"
)

func NewCommand() *cobra.Command {
	var kubeconfig, kubecontext, masterURL string
	var collectorConfig Configuration

	c := &cobra.Command{
		Use:   "sdase-version-collector",
		Short: "Collect apps and their versions.",
		Long:  `sdase-version-collector is a tool that will scan 'Deployment's 'StatefulSet's and 'DaemonSet's for version information and push these as metrics to Prometheus..`,
		Run: func(cmd *cobra.Command, args []string) {
			run(collectorConfig, kubeconfig, kubecontext, masterURL)
		},
	}

	c.PersistentFlags().StringVar(&collectorConfig.teamName, "team-name", "5xx", "Name of the team used for extracting data.")
	c.PersistentFlags().StringVar(&kubeconfig, "kubeconfig", "", "absolute path to the kubeconfig file")
	c.PersistentFlags().StringVar(&kubecontext, "kubecontext", "", "The context to use to talk to the Kubernetes apiserver. If unset defaults to whatever your current-context is (kubectl config current-context)")
	c.PersistentFlags().StringVar(&masterURL, "master", "", "URL of the API server")

	// init and add the klog flags
	klog.InitFlags(flag.CommandLine)
	c.PersistentFlags().AddGoFlagSet(flag.CommandLine)

	return c
}

func run(config Configuration, kubeconfig, kubecontext, masterURL string) {
	result := NewCollector(config, kubeclient.CreateClientOrDie(kubeconfig, kubecontext, masterURL)).Execute()
	for _, entry := range result.Entries {
		klog.Info(entry)
	}
}
