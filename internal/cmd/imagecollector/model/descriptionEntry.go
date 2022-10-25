package model

type DescriptionEntry struct {
	Environment string `validate:"required" json:"environment"`
	Namespace   string `validate:"required" json:"namespace"`
	Team        string `validate:"ascii" json:"team" copier:"must"`
	Description string `validate:"ascii" json:"description" copier:"must"`
}
