package model

import (
	"github.com/go-playground/validator/v10"
	"testing"
)

func TestValidateCollectorEntry(t *testing.T) {
	entry := CollectorEntry{
		Slack:         "#test-channel",
		Environment:   "test",
		Namespace:     "test",
		Image:         "quay.io/test",
		ImageId:       "quay.io/test",
		ContainerType: "application",
		Email:         "security-journey@not-existing.de",
	}
	validate := validator.New()
	validate.RegisterStructValidation(ValidateCollectorEntry, CollectorEntry{})

	err := validate.Struct(entry)
	if err != nil {
		t.Fatal("entry slack channel is not valid", entry)
	}

}
