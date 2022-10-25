package model

import (
	"regexp"

	"github.com/go-playground/validator/v10"
)

type CollectorEntry struct {
	Environment                      string `validate:"required" json:"environment"`
	Namespace                        string `validate:"required" json:"namespace"`
	Image                            string `validate:"required" json:"image"`
	ImageId                          string `validate:"required" json:"image_id"`
	Team                             string `validate:"ascii" json:"team" copier:"must"`
	Slack                            string `json:"slack" copier:"must"`
	Rocketchat                       string `json:"rocketchat"`
	Email                            string `validate:"omitempty,email" json:"email"`
	AppKubernetesName                string `validate:"ascii" json:"app_kubernetes_name"`
	AppKubernetesVersion             string `validate:"ascii" json:"app_kubernetes_version"`
	ContainerType                    string `validate:"oneof=application third-party,required" json:"container_type"`
	IsScanBaseimageLifetime          bool   `json:"is_scan_baseimage_lifetime"`
	IsScanDependencyCheck            bool   `json:"is_scan_dependency_check"`
	IsScanDependencyTrack            bool   `json:"is_scan_dependency_track"`
	IScanDistroless                  bool   `json:"is_scan_distroless"`
	IScanLifetime                    bool   `json:"is_scan_lifetime"`
	IScanMalware                     bool   `json:"is_scan_malware"`
	IsScanNewVersion                 bool   `json:"is_scan_new_version"`
	IsScanRunAsRoot                  bool   `json:"is_scan_run_as_root"`
	IsPotentiallyRunningAsRoot       bool   `json:"is_potentially_running_as_root"`
	IsScanRunAsPrivileged            bool   `json:"is_scan_run_as_privileged"`
	IsPotentiallyRunningAsPrivileged bool   `json:"is_potentially_running_as_privileged"`
	ScanMaxDaysLifetime              int    `validate:"numeric" json:"is_scan_runasroot"`
	Skip                             bool   `json:"skip"`
	ScmSourceUrl                     bool   `json:"scm_source_url"`
}

func ValidateCollectorEntry(sl validator.StructLevel) {
	entry := sl.Current().Interface().(CollectorEntry)
	validChannel := regexp.MustCompile(`^#\w+$`)

	if entry.Slack != "" && !validChannel.MatchString(entry.Slack) {
		sl.ReportError(entry.Slack, "Slack", "Slack", "", "`^#\\w$`")
	}
	if entry.Rocketchat != "" && !validChannel.MatchString(entry.Rocketchat) {
		sl.ReportError(entry.Rocketchat, "Rocketchat", "Rocketchat", "", "`^#\\w$`")
	}
}
