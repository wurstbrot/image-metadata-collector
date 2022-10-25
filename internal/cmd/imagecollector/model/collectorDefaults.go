package model

import "k8s.io/client-go/kubernetes"

type ImageCollectorDefaults struct {
	Skip                  bool                  `validate:"required"`
	Environment           string                `validate:"required"`
	ScanIntervalInSeconds int64                 `validate:"required"`
	Client                *kubernetes.Clientset `validate:"required"`
	ConfigBasePath        string                `validate:"required"`
	IsSaveFiles           bool                  `validate:"required"`
}
