package model

type RegistyReplacments struct {
	Collection []RegistyReplacment
}
type RegistyReplacment struct {
	Original    string `validate:"string,required" json:"original"`
	Replacement string `validate:"string,required" json:"replacement"`
}
