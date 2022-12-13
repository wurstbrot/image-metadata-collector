package model

type GitParameterEntry struct {
	Url                  string `validate:"required"`
	Directory            string `validate:"required"`
	PrivateKeyFile       string `validate:"required"`
	Password             string
	GithubAppId          int64
	GithubInstallationId int64
}
