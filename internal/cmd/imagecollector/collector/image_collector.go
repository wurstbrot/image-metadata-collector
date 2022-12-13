package collector

import (
	"context"
	"encoding/json"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"sdase.org/collector/internal/cmd/imagecollector/library"
	"sdase.org/collector/internal/cmd/imagecollector/storage"

	"github.com/jinzhu/copier"

	"github.com/go-playground/validator/v10"
	"github.com/rs/zerolog/log"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sdase.org/collector/internal/cmd/imagecollector/model"
)

var validate *validator.Validate
var collectorEntryDefault = model.CollectorEntry{
	ContainerType:                    "application",
	IsScanBaseimageLifetime:          true,
	IsScanDependencyCheck:            true,
	IsScanDependencyTrack:            false,
	IScanDistroless:                  true,
	IScanLifetime:                    true,
	IScanMalware:                     true,
	IsScanNewVersion:                 true,
	IsScanRunAsRoot:                  true,
	IsPotentiallyRunningAsRoot:       true,
	IsScanRunAsPrivileged:            true,
	IsPotentiallyRunningAsPrivileged: true,
	Skip:                             false,
	ScanMaxDaysLifetime:              14,
	Team:                             "nobody",
	EngagementTags:                   []string{},
}

const namespaceFilterAnnotation = "clusterscanner.sdase.org/namespace_filter"
const namespaceFilterNegatedAnnotation = "clusterscanner.sdase.org/negated_namespace_filter"

var replacements []model.RegistyReplacment

const configBasePath = "/configs"

func clusterImageScannerCollectorRun(imageCollectorDefaults model.ImageCollectorDefaults, s3ParameterEntry model.S3parameterEntry, gitParameterEntry model.GitParameterEntry) error {
	// TODO Verify that SI is not using IMAGE_SKIP_POSITIVE_LIST
	// TODO Verify that SI is not using IMAGE_SKIP_NEGATIVE_LIST
	storage.Init(s3ParameterEntry)
	err := storage.InitGit(gitParameterEntry)
	if err != nil {
		return err
	}
	imageManager, err := library.InitImageNegativeList(configBasePath)
	if err != nil {
		return err
	}
	if err := library.InitRegistryRename(configBasePath); err != nil {
		return err
	}
	collectorEntries := []model.CollectorEntry{}
	namespaces, err := imageCollectorDefaults.Client.CoreV1().Namespaces().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		log.Fatal().Stack().Err(err).Msg("failed to get namespaces")
		return err
	}

	for _, namespace := range namespaces.Items {
		log.Info().Str("namespace", namespace.GetName()).Msg("In namespace")

		collectorEntry := getCollectorEntryFromEnv(imageCollectorDefaults)
		setCollectorEntryFromLabelsAndAnnotations(&collectorEntry, &namespace)
		collectorEntry.Namespace = namespace.GetName()

		checkAndSetNamespaceSkipByRegex(namespace, &collectorEntry)

		pods, err := imageCollectorDefaults.Client.CoreV1().Pods(namespace.GetName()).List(context.Background(), metav1.ListOptions{})
		if err != nil {
			log.Fatal().Stack().Err(err).Msg("failed to get pods:")
		}

		collectorEntryContainers := map[string]model.CollectorEntry{}
		for _, pod := range pods.Items {
			log.Info().Str("pod", pod.GetName()).Msg("In Pod")
			setCollectorEntryFromLabelsAndAnnotations(&collectorEntry, &pod)

			collectorEntryContainers = getContainerSpecs(pod, collectorEntryContainers)
			if collectorEntryContainers, err = getContainerStatuses(collectorEntry, pod, collectorEntryContainers, imageManager); err != nil {
				log.Fatal().Err(err).Stack().Msg("Could not get container statuses")
				return err
			}
		}
		for _, entry := range collectorEntryContainers {
			collectorEntries = append(collectorEntries, entry)
		}
	}
	storeFiles(collectorEntries, imageCollectorDefaults)
	return nil
}
func getContainerSpecs(pod v1.Pod, collectorEntryContainers map[string]model.CollectorEntry) map[string]model.CollectorEntry {
	for _, container := range pod.Spec.Containers {
		if container.SecurityContext == nil {
			log.Debug().Str("ContainerName", container.Name).Msg("SecurityContext for container not available ")
		} else {
			log.Debug().Str("ContainerName", container.Name).Msg("SecurityContext for container available ")
			collectorEntryContainer := model.CollectorEntry{}
			if container.SecurityContext.RunAsNonRoot != nil {
				collectorEntryContainer.IsPotentiallyRunningAsRoot = !*container.SecurityContext.RunAsNonRoot
				log.Debug().Str("ContainerName", container.Name).
					Bool("*collectorEntryContainer.IsPotentiallyRunningAsRoot ", collectorEntryContainer.IsPotentiallyRunningAsRoot)
			}
			if container.SecurityContext.AllowPrivilegeEscalation != nil {
				collectorEntryContainer.IsPotentiallyRunningAsPrivileged = *container.SecurityContext.AllowPrivilegeEscalation
			}
			if !collectorEntryContainer.IsPotentiallyRunningAsPrivileged && container.SecurityContext.Privileged != nil {
				collectorEntryContainer.IsPotentiallyRunningAsPrivileged = *container.SecurityContext.Privileged
			}
			collectorEntryContainer.Image = container.Image
			collectorEntryContainers[pod.Name+container.Name] = collectorEntryContainer
		}
	}
	return collectorEntryContainers
}
func getContainerStatuses(collectorEntry model.CollectorEntry, pod v1.Pod, collectorEntryContainers map[string]model.CollectorEntry, imageManager library.ImageManager) (map[string]model.CollectorEntry, error) {
	for _, status := range pod.Status.ContainerStatuses {
		collectorEntryContainer := collectorEntry
		collectorEntryContainer.Image = library.PrepareImageName(status.Image)
		collectorEntryContainer.ImageId = library.PrepareImageName(status.ImageID)

		if status.ImageID == "" {
			collectorEntryContainer.ImageId = status.Image
		}
		if strings.Contains(collectorEntryContainer.Image, "sha256:") {
			if !strings.Contains(collectorEntryContainer.ImageId, "sha256:") {
				collectorEntryContainer.ImageId = collectorEntryContainer.Image
			}
		}

		library.CheckAndSetSkipByImageNegativeList(imageManager, status, &collectorEntry)
		validate = validator.New()
		validate.RegisterStructValidation(model.ValidateCollectorEntry, model.CollectorEntry{})

		err := validate.Struct(collectorEntryContainer)
		if err != nil {
			if _, ok := err.(*validator.InvalidValidationError); ok {
				log.Fatal().Stack().Err(err).Msg("Could not validate struct")
				return nil, err
			}

			for _, err := range err.(validator.ValidationErrors) {
				log.Fatal().Stack().Err(err).Msg("Validation Errors")
			}
		}

		if _, ok := collectorEntryContainers[pod.Name+status.Name]; ok {
			log.Debug().Str("pod.name", pod.Name).Msg("collector entry exists from containers, merging via copy")
			combinedEntry := collectorEntryContainers[pod.Name+status.Name]
			if err := copier.Copy(&combinedEntry, &collectorEntryContainer); err != nil {
				log.Fatal().Stack().Err(err).Msg("Could not copy collectorEntryContainer")
			}
			collectorEntryContainers[pod.Name+status.Name] = combinedEntry
		} else {
			log.Debug().Str("pod.name", pod.Name).Msg("collector entry doesn't exists from containers")
			collectorEntryContainers[pod.Name+status.Name] = collectorEntryContainer
		}
	}

	return collectorEntryContainers, nil
}

func checkAndSetNamespaceSkipByRegex(namespace v1.Namespace, collectorEntry *model.CollectorEntry) {
	namespaceSkipRegex := os.Getenv("NAMESPACE_SKIP_REGEX")
	isNamespaceSkip, _ := regexp.MatchString(namespaceSkipRegex, namespace.GetName())
	if namespaceSkipRegex != "" && isNamespaceSkip {
		log.Debug().Str("NAMESPACE_SKIP_REGEX from env:", namespaceSkipRegex).Msg("Skipping image due to namespace name")
		collectorEntry.Skip = isNamespaceSkip
	}
	var namespaceFilter = ""

	library.SetStringFromAnnotationAndLabel(&namespace, namespaceFilterAnnotation, &namespaceFilter)
	if namespaceFilter != "" && !collectorEntry.Skip {
		isNamespaceSkipFilter, _ := regexp.MatchString(namespaceFilter, namespace.GetName())
		if namespaceSkipRegex != "" && isNamespaceSkipFilter {
			log.Debug().Str("namespaceFilterAnnotation", namespaceFilterAnnotation).Msg("Skipping image due to namespaceFilter")
			collectorEntry.Skip = isNamespaceSkipFilter
		}
	}

	var negatedNamespaceFilter = ""
	library.SetStringFromAnnotationAndLabel(&namespace, namespaceFilterNegatedAnnotation, &negatedNamespaceFilter)
	if negatedNamespaceFilter != "" && !collectorEntry.Skip {
		isNamespaceSkipFilter, _ := regexp.MatchString(negatedNamespaceFilter, namespace.GetName())
		isNamespaceNegatedSkipFilter := !isNamespaceSkipFilter
		if namespaceSkipRegex != "" && isNamespaceNegatedSkipFilter {
			log.Debug().Str("negatedNamespaceFilter", negatedNamespaceFilter).Msg("Skipping image due to negatedNamespaceFilter")
			collectorEntry.Skip = isNamespaceNegatedSkipFilter
		}
	}
}

func typecastStringToBoolOrFalseAndSetIt(value string, key *bool) { //nolint:all
	if value == "" {
		return
	}
	val, err := strconv.ParseBool(value) // copy
	if err != nil {
		log.Warn().Stack().Err(err).Str("value", value).Msg("Couldn't typecast string to bool")
	}
	key = &val //nolint:all
}

//Interface getLabels/getAnnotations
func setBooleanFromAnnotationAndLabel(annotateabbleAndLabelableObject library.AnnotateableAndLabelableInterface, annotationName string, key *bool) {
	var label = annotateabbleAndLabelableObject.GetLabels()[annotationName]
	if label != "" {
		typecastStringToBoolOrFalseAndSetIt(label, key)
	}

	var annotation = annotateabbleAndLabelableObject.GetAnnotations()[annotationName]
	if annotation != "" {
		typecastStringToBoolOrFalseAndSetIt(annotation, key)
	}
}

func typecastNumberToIntAndSetIt(number string, key *int) error { //nolint:all
	if (number) == "" {
		return nil
	}
	val, err := strconv.Atoi(number)
	if err != nil {
		log.Warn().Stack().Err(err).Str("number", number).Msg("Couldn't transform string")
		return err
	}
	key = &val
	return nil
}

var defaultEntryFromEnv model.CollectorEntry

func getCollectorEntryFromEnv(imageCollectorDefaults model.ImageCollectorDefaults) model.CollectorEntry {
	if defaultEntryFromEnv.ScanMaxDaysLifetime == 0 {
		defaultEntryFromEnv = collectorEntryDefault

		if os.Getenv("DEFAULT_CONTAINER_TYPE") != "" {
			defaultEntryFromEnv.ContainerType = os.Getenv("DEFAULT_CONTAINER_TYPE")
		}
		if os.Getenv("DEFAULT_TEAM_NAME") != "" {
			defaultEntryFromEnv.Team = os.Getenv("DEFAULT_TEAM_NAME")
		}
		defaultEntryFromEnv.Environment = imageCollectorDefaults.Environment

		engagementTags := os.Getenv("DEFAULT_ENGAGEMENT_TAGS")
		if engagementTags != "" && engagementTags != "null" {
			defaultEntryFromEnv.EngagementTags = strings.Split(engagementTags, ",")
		}
		jTags, _ := json.Marshal(defaultEntryFromEnv.EngagementTags)
		log.Info().Bytes("defaultEntryFromEnv.EngagementTags ", jTags).Msg("JSON")
		typecastNumberToIntAndSetIt(os.Getenv("DEFAULT_SCAN_LIFETIME_MAX_DAYS"), &defaultEntryFromEnv.ScanMaxDaysLifetime)
		typecastStringToBoolOrFalseAndSetIt(os.Getenv("DEFAULT_SCAN_BASEIMAGE_LIFETIME"), &defaultEntryFromEnv.IsScanBaseimageLifetime)
		typecastStringToBoolOrFalseAndSetIt(os.Getenv("DEFAULT_SCAN_DEPENDENCY_CHECK"), &defaultEntryFromEnv.IsScanDependencyCheck)
		typecastStringToBoolOrFalseAndSetIt(os.Getenv("DEFAULT_SCAN_DEPENDENCY_TRACK"), &defaultEntryFromEnv.IsScanDependencyTrack)
		typecastStringToBoolOrFalseAndSetIt(os.Getenv("DEFAULT_SCAN_DISTROLESS"), &defaultEntryFromEnv.IScanDistroless)
		typecastStringToBoolOrFalseAndSetIt(os.Getenv("DEFAULT_SCAN_LIFETIME"), &defaultEntryFromEnv.IScanLifetime)
		typecastStringToBoolOrFalseAndSetIt(os.Getenv("DEFAULT_SCAN_MALWARE"), &defaultEntryFromEnv.IScanMalware)
		typecastStringToBoolOrFalseAndSetIt(os.Getenv("DEFAULT_SCAN_NEW_VERSION"), &defaultEntryFromEnv.IsScanNewVersion)
		typecastStringToBoolOrFalseAndSetIt(os.Getenv("DEFAULT_SCAN_RUN_AS_ROOT"), &defaultEntryFromEnv.IsScanRunAsRoot)
		typecastStringToBoolOrFalseAndSetIt(os.Getenv("DEFAULT_SCAN_RUN_AS_PRIVILEGED"), &defaultEntryFromEnv.IsScanRunAsPrivileged)
		typecastStringToBoolOrFalseAndSetIt(os.Getenv("DEFAULT_SKIP"), &defaultEntryFromEnv.Skip)
	}
	return defaultEntryFromEnv
}

func setCollectorEntryFromLabelsAndAnnotations(collectorEntry *model.CollectorEntry, annotateabbleAndLabelableObject library.AnnotateableAndLabelableInterface) {
	setBooleanFromAnnotationAndLabel(annotateabbleAndLabelableObject, "clusterscanner.sdase.org/is-scan-baseimage-lifetime", &collectorEntry.IsScanBaseimageLifetime)
	setBooleanFromAnnotationAndLabel(annotateabbleAndLabelableObject, "clusterscanner.sdase.org/is-scan-dependency-check", &collectorEntry.IsScanDependencyCheck)
	setBooleanFromAnnotationAndLabel(annotateabbleAndLabelableObject, "clusterscanner.sdase.org/is-scan-dependency-track", &collectorEntry.IsScanDependencyTrack)
	setBooleanFromAnnotationAndLabel(annotateabbleAndLabelableObject, "clusterscanner.sdase.org/is-scan-lifetime", &collectorEntry.IScanLifetime)
	setBooleanFromAnnotationAndLabel(annotateabbleAndLabelableObject, "clusterscanner.sdase.org/is-scan-distroless", &collectorEntry.IScanDistroless)
	setBooleanFromAnnotationAndLabel(annotateabbleAndLabelableObject, "clusterscanner.sdase.org/is-scan-malware", &collectorEntry.IScanMalware)
	setBooleanFromAnnotationAndLabel(annotateabbleAndLabelableObject, "clusterscanner.sdase.org/is-scan-new-version", &collectorEntry.IsScanNewVersion)
	setBooleanFromAnnotationAndLabel(annotateabbleAndLabelableObject, "clusterscanner.sdase.org/is-scan-runasroot", &collectorEntry.IsScanRunAsRoot)
	setBooleanFromAnnotationAndLabel(annotateabbleAndLabelableObject, "clusterscanner.sdase.org/skip", &collectorEntry.Skip)

	library.SetStringFromAnnotationAndLabel(annotateabbleAndLabelableObject, "contact.sdase.org/slack", &collectorEntry.Slack)
	library.SetStringFromAnnotationAndLabel(annotateabbleAndLabelableObject, "contact.sdase.org/rocketchat", &collectorEntry.Rocketchat)
	library.SetStringFromAnnotationAndLabel(annotateabbleAndLabelableObject, "contact.sdase.org/email", &collectorEntry.Email)
	library.SetStringFromAnnotationAndLabel(annotateabbleAndLabelableObject, "contact.sdase.org/container_type", &collectorEntry.ContainerType)
	library.SetStringFromAnnotationAndLabel(annotateabbleAndLabelableObject, "contact.sdase.org/team", &collectorEntry.Team)

	library.SetStringFromAnnotationAndLabel(annotateabbleAndLabelableObject, "app.kubernetes.io/name", &collectorEntry.AppKubernetesName)
	library.SetStringFromAnnotationAndLabel(annotateabbleAndLabelableObject, "app.kubernetes.io/version", &collectorEntry.AppKubernetesVersion)

	var engagementTags = ""
	engagementTagsAnnotationName := getEngagementTagsAnnotationName()
	library.SetStringFromAnnotationAndLabel(annotateabbleAndLabelableObject, engagementTagsAnnotationName, &engagementTags)
	jTags, _ := json.Marshal(defaultEntryFromEnv.EngagementTags)
	log.Info().Bytes("engagementTags ", jTags).Msg("JSON")
	if engagementTags != "" && engagementTags != "null" {
		engagementTagsAsList := strings.Split(engagementTags, ",")
		collectorEntry.EngagementTags = append(collectorEntry.EngagementTags, engagementTagsAsList...)
	}
}
func getEngagementTagsAnnotationName() string {
	engagementTagsAnnotationName := os.Getenv("ANNOTATION_NAME_ENGAGEMENT_TAG")
	if engagementTagsAnnotationName == "" {
		engagementTagsAnnotationName = "defectdojo.sdase.org/engagement-tags"
	}
	return engagementTagsAnnotationName
}

<<<<<<< HEAD
func storeFiles(collectorEntries []model.CollectorEntry, imageCollectorDefaults model.ImageCollectorDefaults) {
	saveFilesPath := "/tmp"
=======
func storeAndUploadFiles(collectorEntries []model.CollectorEntry, imageCollectorDefaults model.ImageCollectorDefaults) error {
	filename := imageCollectorDefaults.Environment + "-output.json"
>>>>>>> 180f1bf (Feat/git (#15))
	sort.Slice(collectorEntries, func(i, j int) bool {
		return collectorEntries[i].Image < collectorEntries[j].Image
	})
	dataCollectionEntries, err := json.Marshal(collectorEntries)
	if err != nil {
		log.Fatal().Stack().Err(err).Msg("Could not marshal json collectorEntries")
		return
	}
	if imageCollectorDefaults.IsSaveFiles {
		saveFilesPath := "/tmp"
		filePath := saveFilesPath + "/" + filename
		library.SaveFile(filePath, []byte(dataCollectionEntries))
	}
	if err = storage.Upload([]byte(dataCollectionEntries), filename, imageCollectorDefaults.Environment); err != nil {
		return err
	}
	if err = storage.GitUpload([]byte(dataCollectionEntries), filename); err != nil {
		return err
	}
	return nil
}

func Run(isImageCollector bool, imageCollectorDefaults model.ImageCollectorDefaults, s3ParameterEntry model.S3parameterEntry, gitParameterEntry model.GitParameterEntry) {
	if isImageCollector {
		log := log.With().
			Str("component", "image-collector").Logger()
		log.Info().Str("environmentName", imageCollectorDefaults.Environment).Int64("scanInterval", imageCollectorDefaults.ScanIntervalInSeconds).Msg("imageCollector is enabled")
		for {
			err := clusterImageScannerCollectorRun(imageCollectorDefaults, s3ParameterEntry, gitParameterEntry)
			if err != nil {
				log.Fatal().Stack().Err(err).Msg("Stopping due to error in clusterImageScannerCollectorRun")
				return
			}
			if err := ClusterImageScannerDescriptionCollectorRun(imageCollectorDefaults); err != nil {
				log.Fatal().Stack().Err(err).Msg("Stopping due to error in ClusterImageScannerDescriptionCollectorRun")
				return
			}

			if imageCollectorDefaults.ScanIntervalInSeconds == int64(-1) {
				log.Info().Msg("ScanIntervalInSeconds is -1, stopping collector")
				return
			}
			log.Info().Str("environmentName", imageCollectorDefaults.Environment).Int64("scanInterval", imageCollectorDefaults.ScanIntervalInSeconds).Msg("sleeping")
			time.Sleep(time.Duration(imageCollectorDefaults.ScanIntervalInSeconds) * time.Second)
		}
	}
}
