package reuse

import (
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

func NewR2Client() *s3.S3 {
	sess := session.Must(session.NewSession(&aws.Config{
		Region:   aws.String("auto"),
		Endpoint: aws.String(os.Getenv("R2_ENDPOINT")),
		Credentials: credentials.NewStaticCredentials(
			os.Getenv("R2_ACCESS_KEY"),
			os.Getenv("R2_SECRET_KEY"),
			"",
		),
	}))
	return s3.New(sess)
}
