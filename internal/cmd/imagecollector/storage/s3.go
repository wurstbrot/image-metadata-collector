package storage

import (
	"bytes"
	"fmt"
	"net/http"
	"path"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/go-playground/validator/v10"
	"github.com/rs/zerolog/log"
	"sdase.org/collector/internal/cmd/imagecollector/model"
)

var s3ParameterEntry = model.S3parameterEntry{}

func Init(providedS3ParameterEntry model.S3parameterEntry) {
	s3ParameterEntry = providedS3ParameterEntry
	var validate = validator.New()
	validate.RegisterStructValidation(model.ValidateCollectorEntry, model.CollectorEntry{})

	err := validate.Struct(s3ParameterEntry)
	if err != nil {
		if _, ok := err.(*validator.InvalidValidationError); ok {
			log.Fatal().Stack().Err(err).Msg("Could not validate struct")
		}

		for _, err := range err.(validator.ValidationErrors) {
			log.Fatal().Stack().Err(err).Msg("Validation Errors")
		}
	}
}

func Store(content []byte, fileName string, environmentName string) {
	sess, err := session.NewSession(&aws.Config{
		Credentials:                   credentials.NewStaticCredentials(s3ParameterEntry.S3accessKey, s3ParameterEntry.S3secretKey, ""),
		CredentialsChainVerboseErrors: aws.Bool(false),
		Endpoint:                      aws.String(s3ParameterEntry.S3endpoint),
		DisableSSL:                    aws.Bool(s3ParameterEntry.S3insecure),
		S3ForcePathStyle:              aws.Bool(s3ParameterEntry.S3ForcePathStyle),
		Region:                        aws.String(s3ParameterEntry.S3region),
		//LogLevel:                      aws.LogLevel(aws.LogDebug | aws.LogDebugWithHTTPBody | aws.LogDebugWithRequestRetries | aws.LogDebugWithRequestErrors | aws.LogDebugWithSigning),
		Logger: aws.NewDefaultLogger(),
	})
	//TODO is debugging für logLevel
	if err != nil {
		log.Error().Msg("Could not create session with s3 bucket")
		return
	}

	var size int64 = int64(len(content))
	fileType := http.DetectContentType(content)

	log.Info().Str("path.Base(file.Name())", path.Base(fileName)).Msg("Storing")
	var res, uploadError = s3.New(sess).PutObject(&s3.PutObjectInput{
		Bucket:             aws.String(s3ParameterEntry.S3bucket),
		Key:                aws.String(environmentName + "/imagecollector/" + path.Base(fileName)),
		Body:               bytes.NewReader(content),
		ContentDisposition: aws.String("attachment"),
		ContentLength:      aws.Int64(size),
		ContentType:        aws.String(fileType),
	})
	if uploadError != nil {
		log.Error().Msg(fmt.Sprintf("Failed to upload to S3 bucket %s, err: %v", s3ParameterEntry.S3bucket, uploadError))
		return
	}
	log.Info().Str("res", res.String()).Msg("Created new file in s3")
}
