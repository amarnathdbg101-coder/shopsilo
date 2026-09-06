package reuse

import (
	"shopMe/internal/utils"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

func NewR2Client() *s3.S3 {
	sess := session.Must(session.NewSession(&aws.Config{
		Region:   aws.String("auto"),
		Endpoint: aws.String(utils.MustLoad().Endpoint),
		Credentials: credentials.NewStaticCredentials(
			utils.MustLoad().AccessKey,
			utils.MustLoad().SecretKey,
			"",
		),
	}))
	return s3.New(sess)
}
