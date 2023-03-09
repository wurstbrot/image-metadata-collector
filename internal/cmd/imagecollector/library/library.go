package library

import (
	"os"

	"github.com/rs/zerolog/log"
)

type AnnotateableAndLabelableInterface interface {
	GetAnnotations() map[string]string
	GetLabels() map[string]string
}

func SetStringFromAnnotationAndLabel(annotateableAndLabelableObject AnnotateableAndLabelableInterface, annotationName string, entryAttribute *string) { //nolint:all
	var label = annotateableAndLabelableObject.GetLabels()[annotationName]
	if label != "" {
		*entryAttribute = label //nolint:all
	}

	var annotation = annotateableAndLabelableObject.GetAnnotations()[annotationName]
	if annotation != "" {
		*entryAttribute = annotation //nolint:all
	}
}

func SaveFile(path string, content []byte) {
	err := os.WriteFile(path, content, 0755)
	if err != nil {
		log.Info().Stack().Err(err).Str("path", path).Msg("Error during opening file")
	}
}
