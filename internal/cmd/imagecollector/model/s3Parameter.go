package model

type S3parameterEntry struct {
	S3bucket         string `validate:"required"`
	S3secretKey      string
	S3accessKey      string
	S3endpoint       string `validate:"required"`
	S3insecure       bool   `validate:"required"`
	S3region         string `validate:"required"`
	S3ForcePathStyle bool   `validate:"required"`
}
