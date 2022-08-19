package collector

import (
	"fmt"

	"k8s.io/client-go/kubernetes"
	"sda.se/version-collector/internal/pkg/semantic"
)

type ApplicationEntry struct {
	Name            string
	AppVersion      *semantic.Version
	HelmVersion     *semantic.Version
	IsManagedByHelm bool
}

func (e ApplicationEntry) String() string {
	return fmt.Sprintf("%s: appVersion=%s, helmVersion=%s, managed-by-helm=%t", e.Name, e.AppVersion, e.HelmVersion, e.IsManagedByHelm)
}

type Result struct {
	Entries []ApplicationEntry
}

type Collector interface {
	Execute() *Result
}

func NewCollector(teamName string, client *kubernetes.Clientset) Collector {
	return &collectorImpl{
		client:         client,
		teamListOption: createTeamListOption(teamName),
	}
}
