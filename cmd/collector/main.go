package main

import (
	"k8s.io/klog"
	"sda.se/version-collector/internal/cmd"
	"sda.se/version-collector/internal/cmd/collector"
)

func main() {
	defer klog.Flush()

	err := collector.NewCommand().Execute()
	cmd.CheckError(err)
}
