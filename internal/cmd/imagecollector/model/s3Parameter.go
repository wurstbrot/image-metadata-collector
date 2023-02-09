package model

type S3parameterEntry struct {
	Disabled         bool
	S3bucket         string
	S3secretKey      string
	S3accessKey      string
	S3endpoint       string
	S3insecure       bool
	S3region         string
	S3ForcePathStyle bool
}
