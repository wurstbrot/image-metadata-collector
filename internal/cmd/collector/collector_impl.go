package collector

import (
	"context"
	"fmt"
	"strings"

	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"
	"sda.se/version-collector/internal/pkg/semantic"

	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const appNameLabel = "app.kubernetes.io/name"
const versionLabel = "app.kubernetes.io/version"
const managedByLabel = "app.kubernetes.io/managed-by"
const teamLabel = "contact.sdase.org/team"
const containerLabel = "app.sdase.org/container"
const chartLabel = "helm.sh/chart"

type collectorImpl struct {
	client         kubernetes.Interface
	teamListOption metav1.ListOptions
}

func (c *collectorImpl) Execute() *Result {
	deployments := c.collectDeployments()
	statefulSets := c.collectStatefulSets()
	daemonSets := c.collectDaemonSets()

	entries := append([]collectionInputEntry{}, deployments...)
	entries = append(entries, statefulSets...)
	entries = append(entries, daemonSets...)

	entryByName := map[string]ApplicationEntry{}

	for _, d := range entries {
		appVersion, appErr := d.extractAppVersion()
		helmVersion, helmErr := d.extractHelmVersion()
		if appErr != nil && helmErr != nil {
			klog.Warningf("%s. %s. Entry will be ignored", appErr.Error(), helmErr.Error())
			continue
		}

		name := d.labels[appNameLabel]
		if name == "" {
			klog.Warningf("Mandatory label '%s' is missing. Entry will be ignored", appNameLabel)
			continue
		}
		entryByName[name] = ApplicationEntry{
			Name:            name,
			AppVersion:      appVersion,
			HelmVersion:     helmVersion,
			IsManagedByHelm: strings.ToLower(d.labels[managedByLabel]) == "helm",
		}
	}

	result := &Result{
		Entries: []ApplicationEntry{},
	}

	for _, v := range entryByName {
		result.Entries = append(result.Entries, v)
	}

	return result
}

func createTeamListOption(teamName string) metav1.ListOptions {
	return metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s", teamLabel, teamName),
	}
}

type collectionInputContainer struct {
	name  string
	image string
}

type collectionInputEntry struct {
	name       string
	labels     map[string]string
	containers []collectionInputContainer
}

func (c *collectorImpl) collectDeployments() []collectionInputEntry {
	client := c.client.AppsV1().Deployments(apiv1.NamespaceAll)
	var entries []collectionInputEntry
	list, err := client.List(context.TODO(), c.teamListOption)
	if err != nil {
		klog.Warning(err)
		return []collectionInputEntry{}
	}
	for _, l := range list.Items {
		var containers []collectionInputContainer
		for _, container := range l.Spec.Template.Spec.Containers {
			containers = append(containers, collectionInputContainer{
				name:  container.Name,
				image: container.Image,
			})
		}
		entries = append(entries, collectionInputEntry{
			name:       l.Name,
			labels:     l.ObjectMeta.Labels,
			containers: containers,
		})
	}
	return entries
}

func (c *collectorImpl) collectStatefulSets() []collectionInputEntry {
	client := c.client.AppsV1().StatefulSets(apiv1.NamespaceAll)
	var entries []collectionInputEntry
	list, err := client.List(context.TODO(), c.teamListOption)
	if err != nil {
		klog.Warning(err)
		return []collectionInputEntry{}
	}
	for _, l := range list.Items {
		var containers []collectionInputContainer
		for _, container := range l.Spec.Template.Spec.Containers {
			containers = append(containers, collectionInputContainer{
				name:  container.Name,
				image: container.Image,
			})
		}
		entries = append(entries, collectionInputEntry{
			name:       l.Name,
			labels:     l.ObjectMeta.Labels,
			containers: containers,
		})
	}
	return entries
}

func (c *collectorImpl) collectDaemonSets() []collectionInputEntry {
	client := c.client.AppsV1().DaemonSets(apiv1.NamespaceAll)
	var entries []collectionInputEntry
	list, err := client.List(context.TODO(), c.teamListOption)
	if err != nil {
		klog.Warning(err)
		return []collectionInputEntry{}
	}
	for _, l := range list.Items {
		var containers []collectionInputContainer
		for _, container := range l.Spec.Template.Spec.Containers {
			containers = append(containers, collectionInputContainer{
				name:  container.Name,
				image: container.Image,
			})
		}
		entries = append(entries, collectionInputEntry{
			name:       l.Name,
			labels:     l.ObjectMeta.Labels,
			containers: containers,
		})
	}
	return entries
}

func (c *collectionInputEntry) extractHelmVersion() (*semantic.Version, error) {
	var found bool
	// If present, format is as '{{ .Chart.Name }}-{{ .Chart.Version | replace "+" "_" }}', c.f.
	// https://helm.sh/docs/chart_best_practices/labels/#standard-labels
	versionFromLabel, found := c.labels[chartLabel]
	if !found {
		return nil, fmt.Errorf("%s: No Helm chart version information found", c.name)
	}

	index := strings.LastIndex(versionFromLabel, "_")
	if index == -1 {
		return semantic.ParseFromString(versionFromLabel)
	}

	return semantic.ParseFromString(versionFromLabel[:index] + "+" + versionFromLabel[index+1:])
}

func (c *collectionInputEntry) extractAppVersion() (*semantic.Version, error) {
	var found bool
	versionFromLabel, found := c.labels[versionLabel]
	if found {
		return semantic.ParseFromString(versionFromLabel)
	}

	containers := c.containers
	if len(containers) == 1 {
		return extractVersionFromTag(c.name, containers[0].image)
	}

	mainContainer, found := c.labels[containerLabel]
	if len(containers) > 1 && !found {
		return nil, fmt.Errorf("%s: PodSpec includes multiple container, but '%s' label is missing", c.name, containerLabel)
	}

	for _, container := range containers {
		if container.name == mainContainer {
			return extractVersionFromTag(c.name, container.image)
		}
	}

	return nil, fmt.Errorf("%s: PodSpec doesn't include container named '%s'", c.name, mainContainer)
}

func extractVersionFromTag(name string, image string) (*semantic.Version, error) {
	imageWithoutSha := strings.Split(image, "@")[0]
	parts := strings.Split(imageWithoutSha, ":")
	if len(parts) != 2 {
		return nil, fmt.Errorf("%s: image name doesn't include a tag", name)
	}

	return semantic.ParseFromString(parts[1])
}
