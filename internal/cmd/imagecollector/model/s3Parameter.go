package model

type S3parameterEntry struct {
	S3bucket         string `validate:"required"`
	S3secretKey      string `validate:"required"`
	S3accessKey      string `validate:"required"`
	S3endpoint       string `validate:"required"`
	S3insecure       bool   `validate:"required"`
	S3region         string `validate:"required"`
	S3ForcePathStyle bool   `validate:"required"`
}
