package collector

import (
	"context"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	mockedClient "k8s.io/client-go/kubernetes/fake"
	"sda.se/version-collector/internal/pkg/semantic"
)

var container1 = corev1.Container{
	Name:  "container1",
	Image: "bla:1.1.1",
}
var container2 = corev1.Container{
	Name:  "container2",
	Image: "bla:2.2.2",
}

var podSpecSingleContainer = corev1.PodSpec{
	Containers: []corev1.Container{
		container1,
	},
}
var podSpecTwoContainers = corev1.PodSpec{
	Containers: []corev1.Container{
		container1,
		container2,
	},
}

var metaNoVersion = metav1.ObjectMeta{
	Name: "deploymentNoVersion",
	Labels: map[string]string{
		"app.kubernetes.io/name": "my-name-no-version",
		"contact.sdase.org/team": "5xx",
	},
}
var metaWithVersion = metav1.ObjectMeta{
	Name: "deploymentWithVersion",
	Labels: map[string]string{
		"app.kubernetes.io/name":    "my-name-with-version",
		"contact.sdase.org/team":    "5xx",
		"app.kubernetes.io/version": "3.3.3",
	},
}
var metaNoVersionWithContainer = metav1.ObjectMeta{
	Name: "deploymentNoVersionWithContainer",
	Labels: map[string]string{
		"app.kubernetes.io/name":  "my-name-no-version-with-container",
		"contact.sdase.org/team":  "5xx",
		"app.sdase.org/container": "container2",
	},
}
var metaWithVersionAndContainer = metav1.ObjectMeta{
	Name: "deploymentWithVersionAndContainer",
	Labels: map[string]string{
		"app.kubernetes.io/name":    "my-name-with-version-and-container",
		"contact.sdase.org/team":    "5xx",
		"app.kubernetes.io/version": "3.3.3",
		"app.sdase.org/container":   "container2",
	},
}
var metaNoVersionNoContainer = metav1.ObjectMeta{
	Name: "deploymentWithVersionAndContainer",
	Labels: map[string]string{
		"app.kubernetes.io/name": "my-name-no-version-no-container",
		"contact.sdase.org/team": "5xx",
	},
}
var metaNoTeam = metav1.ObjectMeta{
	Name: "deployment4",
	Labels: map[string]string{
		"app.kubernetes.io/name": "my-name-no-team",
	},
}
var metaWrongTeam = metav1.ObjectMeta{
	Name: "deployment4",
	Labels: map[string]string{
		"app.kubernetes.io/name": "my-name-wrong-team",
		"contact.sdase.org/team": "4xx",
	},
}

var entryNoVersion = collectionInputEntry{
	name:   metaNoVersion.Name,
	labels: metaNoVersion.Labels,
	containers: []collectionInputContainer{
		{
			name:  container1.Name,
			image: container1.Image,
		},
	},
}
var entryWithVersion = collectionInputEntry{
	name:   metaWithVersion.Name,
	labels: metaWithVersion.Labels,
	containers: []collectionInputContainer{
		{
			name:  container1.Name,
			image: container1.Image,
		},
	},
}
var entryNoVersionWithContainer = collectionInputEntry{
	name:   metaNoVersionWithContainer.Name,
	labels: metaNoVersionWithContainer.Labels,
	containers: []collectionInputContainer{
		{
			name:  container1.Name,
			image: container1.Image,
		},
		{
			name:  container2.Name,
			image: container2.Image,
		},
	},
}
var entryWithVersionAndContainer = collectionInputEntry{
	name:   metaWithVersionAndContainer.Name,
	labels: metaWithVersionAndContainer.Labels,
	containers: []collectionInputContainer{
		{
			name:  container1.Name,
			image: container1.Image,
		},
		{
			name:  container2.Name,
			image: container2.Image,
		},
	},
}
var entryNoVersionNoContainer = collectionInputEntry{
	name:   metaNoVersion.Name,
	labels: metaNoVersion.Labels,
	containers: []collectionInputContainer{
		{
			name:  container1.Name,
			image: container1.Image,
		},
		{
			name:  container2.Name,
			image: container2.Image,
		},
	},
}

var expectedCollection = []collectionInputEntry{
	entryNoVersion,
	entryNoVersionWithContainer,
	entryWithVersion,
	entryWithVersionAndContainer,
}

var expectedResult = &Result{
	Entries: []ApplicationEntry{
		{
			Name:            "my-name-no-version",
			AppVersion:      &semantic.Version{Major: 1, Minor: 1, Bugfix: 1},
			HelmVersion:     nil,
			IsManagedByHelm: false,
		},
		{
			Name:            "my-name-no-version-with-container",
			AppVersion:      &semantic.Version{Major: 2, Minor: 2, Bugfix: 2},
			HelmVersion:     nil,
			IsManagedByHelm: false,
		},
		{
			Name:            "my-name-with-version",
			AppVersion:      &semantic.Version{Major: 3, Minor: 3, Bugfix: 3},
			HelmVersion:     nil,
			IsManagedByHelm: false,
		},
		{
			Name:            "my-name-with-version-and-container",
			AppVersion:      &semantic.Version{Major: 3, Minor: 3, Bugfix: 3},
			HelmVersion:     nil,
			IsManagedByHelm: false,
		},
	},
}

func Test_collectionInputEntry_extractVersion(t *testing.T) {
	// no version defined on metadata, no container (single) => extracted from container
	received, _ := entryNoVersion.extractAppVersion()
	expected, _ := semantic.ParseFromString("1.1.1")
	assert.Equal(t, expected, received)

	// no version defined on metadata, container defined (multiple) => extracted from container
	received, _ = entryNoVersionWithContainer.extractAppVersion()
	expected, _ = semantic.ParseFromString("2.2.2")
	assert.Equal(t, expected, received)

	// version defined on metadata
	received, _ = entryWithVersion.extractAppVersion()
	expected, _ = semantic.ParseFromString("3.3.3")
	assert.Equal(t, expected, received)

	// version defined on metadata, container defined (multiple) => extracted from metadata
	received, _ = entryWithVersionAndContainer.extractAppVersion()
	expected, _ = semantic.ParseFromString("3.3.3")
	assert.Equal(t, expected, received)

	// no version defined on metadata, no container defined (multiple) => nil
	received, _ = entryNoVersionNoContainer.extractAppVersion()
	assert.Nil(t, received)
}

func Test_collectorImpl_Execute(t *testing.T) {
	deploy := []appv1.Deployment{
		{
			ObjectMeta: metaNoVersion,
			Spec: appv1.DeploymentSpec{
				Template: corev1.PodTemplateSpec{
					Spec: podSpecSingleContainer,
				},
			},
		},
		{
			ObjectMeta: metaNoVersionWithContainer,
			Spec: appv1.DeploymentSpec{
				Template: corev1.PodTemplateSpec{
					Spec: podSpecTwoContainers,
				},
			},
		},
		{
			ObjectMeta: metaWithVersion,
			Spec: appv1.DeploymentSpec{
				Template: corev1.PodTemplateSpec{
					Spec: podSpecSingleContainer,
				},
			},
		},
		{
			ObjectMeta: metaWithVersionAndContainer,
			Spec: appv1.DeploymentSpec{
				Template: corev1.PodTemplateSpec{
					Spec: podSpecTwoContainers,
				},
			},
		},
		{
			ObjectMeta: metaNoVersionNoContainer,
			Spec: appv1.DeploymentSpec{
				Template: corev1.PodTemplateSpec{
					Spec: podSpecTwoContainers,
				},
			},
		},
		{
			ObjectMeta: metaWrongTeam,
			Spec: appv1.DeploymentSpec{
				Template: corev1.PodTemplateSpec{
					Spec: podSpecSingleContainer,
				},
			},
		},
		{
			ObjectMeta: metaNoTeam,
			Spec: appv1.DeploymentSpec{
				Template: corev1.PodTemplateSpec{
					Spec: podSpecSingleContainer,
				},
			},
		},
	}

	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "some-namespace",
		},
		Spec: corev1.NamespaceSpec{},
	}

	client := mockedClient.NewSimpleClientset()
	_, _ = client.CoreV1().Namespaces().Create(context.TODO(), namespace, metav1.CreateOptions{})
	for _, d := range deploy {
		_, _ = client.AppsV1().Deployments("some-namespace").Create(context.TODO(), &d, metav1.CreateOptions{})
	}

	collector := &collectorImpl{
		client:         client,
		teamListOption: createTeamListOption("5xx"),
	}

	receivedResult := collector.Execute()
	sort.SliceStable(receivedResult.Entries, func(i, j int) bool {
		return receivedResult.Entries[i].Name < receivedResult.Entries[j].Name
	})
	assert.Equal(t, expectedResult, receivedResult)
}

func Test_collectorImpl_collectDaemonSets(t *testing.T) {
	deploy := []appv1.DaemonSet{
		{
			ObjectMeta: metaNoVersion,
			Spec: appv1.DaemonSetSpec{
				Template: corev1.PodTemplateSpec{
					Spec: podSpecSingleContainer,
				},
			},
		},
		{
			ObjectMeta: metaWithVersion,
			Spec: appv1.DaemonSetSpec{
				Template: corev1.PodTemplateSpec{
					Spec: podSpecSingleContainer,
				},
			},
		},
		{
			ObjectMeta: metaNoVersionWithContainer,
			Spec: appv1.DaemonSetSpec{
				Template: corev1.PodTemplateSpec{
					Spec: podSpecTwoContainers,
				},
			},
		},
		{
			ObjectMeta: metaWithVersionAndContainer,
			Spec: appv1.DaemonSetSpec{
				Template: corev1.PodTemplateSpec{
					Spec: podSpecTwoContainers,
				},
			},
		},
		{
			ObjectMeta: metaWrongTeam,
			Spec: appv1.DaemonSetSpec{
				Template: corev1.PodTemplateSpec{
					Spec: podSpecSingleContainer,
				},
			},
		},
		{
			ObjectMeta: metaNoTeam,
			Spec: appv1.DaemonSetSpec{
				Template: corev1.PodTemplateSpec{
					Spec: podSpecSingleContainer,
				},
			},
		},
	}

	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "some-namespace",
		},
		Spec: corev1.NamespaceSpec{},
	}

	client := mockedClient.NewSimpleClientset()
	_, _ = client.CoreV1().Namespaces().Create(context.TODO(), namespace, metav1.CreateOptions{})
	for _, d := range deploy {
		_, _ = client.AppsV1().DaemonSets("some-namespace").Create(context.TODO(), &d, metav1.CreateOptions{})
	}

	collector := &collectorImpl{
		client:         client,
		teamListOption: createTeamListOption("5xx"),
	}

	receivedCollection := collector.collectDaemonSets()
	sort.SliceStable(receivedCollection, func(i, j int) bool {
		return receivedCollection[i].name < receivedCollection[j].name
	})
	assert.Equal(t, expectedCollection, receivedCollection)
}

func Test_collectorImpl_collectDeployments(t *testing.T) {
	deploy := []appv1.Deployment{
		{
			ObjectMeta: metaNoVersion,
			Spec: appv1.DeploymentSpec{
				Template: corev1.PodTemplateSpec{
					Spec: podSpecSingleContainer,
				},
			},
		},
		{
			ObjectMeta: metaWithVersion,
			Spec: appv1.DeploymentSpec{
				Template: corev1.PodTemplateSpec{
					Spec: podSpecSingleContainer,
				},
			},
		},
		{
			ObjectMeta: metaNoVersionWithContainer,
			Spec: appv1.DeploymentSpec{
				Template: corev1.PodTemplateSpec{
					Spec: podSpecTwoContainers,
				},
			},
		},
		{
			ObjectMeta: metaWithVersionAndContainer,
			Spec: appv1.DeploymentSpec{
				Template: corev1.PodTemplateSpec{
					Spec: podSpecTwoContainers,
				},
			},
		},
		{
			ObjectMeta: metaWrongTeam,
			Spec: appv1.DeploymentSpec{
				Template: corev1.PodTemplateSpec{
					Spec: podSpecSingleContainer,
				},
			},
		},
		{
			ObjectMeta: metaNoTeam,
			Spec: appv1.DeploymentSpec{
				Template: corev1.PodTemplateSpec{
					Spec: podSpecSingleContainer,
				},
			},
		},
	}

	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "some-namespace",
		},
		Spec: corev1.NamespaceSpec{},
	}

	client := mockedClient.NewSimpleClientset()
	_, _ = client.CoreV1().Namespaces().Create(context.TODO(), namespace, metav1.CreateOptions{})
	for _, d := range deploy {
		_, _ = client.AppsV1().Deployments("some-namespace").Create(context.TODO(), &d, metav1.CreateOptions{})
	}

	collector := &collectorImpl{
		client:         client,
		teamListOption: createTeamListOption("5xx"),
	}

	receivedCollection := collector.collectDeployments()
	sort.SliceStable(receivedCollection, func(i, j int) bool {
		return receivedCollection[i].name < receivedCollection[j].name
	})
	assert.Equal(t, expectedCollection, receivedCollection)
}

func Test_collectorImpl_collectStatefulSets(t *testing.T) {
	deploy := []appv1.StatefulSet{
		{
			ObjectMeta: metaNoVersion,
			Spec: appv1.StatefulSetSpec{
				Template: corev1.PodTemplateSpec{
					Spec: podSpecSingleContainer,
				},
			},
		},
		{
			ObjectMeta: metaWithVersion,
			Spec: appv1.StatefulSetSpec{
				Template: corev1.PodTemplateSpec{
					Spec: podSpecSingleContainer,
				},
			},
		},
		{
			ObjectMeta: metaNoVersionWithContainer,
			Spec: appv1.StatefulSetSpec{
				Template: corev1.PodTemplateSpec{
					Spec: podSpecTwoContainers,
				},
			},
		},
		{
			ObjectMeta: metaWithVersionAndContainer,
			Spec: appv1.StatefulSetSpec{
				Template: corev1.PodTemplateSpec{
					Spec: podSpecTwoContainers,
				},
			},
		},
		{
			ObjectMeta: metaWrongTeam,
			Spec: appv1.StatefulSetSpec{
				Template: corev1.PodTemplateSpec{
					Spec: podSpecSingleContainer,
				},
			},
		},
		{
			ObjectMeta: metaNoTeam,
			Spec: appv1.StatefulSetSpec{
				Template: corev1.PodTemplateSpec{
					Spec: podSpecSingleContainer,
				},
			},
		},
	}

	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "some-namespace",
		},
		Spec: corev1.NamespaceSpec{},
	}

	client := mockedClient.NewSimpleClientset()
	_, _ = client.CoreV1().Namespaces().Create(context.TODO(), namespace, metav1.CreateOptions{})
	for _, d := range deploy {
		_, _ = client.AppsV1().StatefulSets("some-namespace").Create(context.TODO(), &d, metav1.CreateOptions{})
	}

	collector := &collectorImpl{
		client:         client,
		teamListOption: createTeamListOption("5xx"),
	}

	receivedCollection := collector.collectStatefulSets()
	sort.SliceStable(receivedCollection, func(i, j int) bool {
		return receivedCollection[i].name < receivedCollection[j].name
	})
	assert.Equal(t, expectedCollection, receivedCollection)
}
