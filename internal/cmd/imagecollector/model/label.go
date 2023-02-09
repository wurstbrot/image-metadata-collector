package model

type Label struct {
	Team                   string `validate:"required"`
	Product                string `validate:"required"`
	Slack                  string
	Rocketchat             string
	Email                  string
	NamespaceFilter        string
	NamespaceFilterNegated string
	ContainerType          string
	EngagementTags         string
}
