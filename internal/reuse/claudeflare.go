// Package reuse reduce repetition task.
package reuse

import (
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/url"
	"path/filepath"
	"shopMe/internal/utils"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

const (
	MaxImageSizeBytes = 2 * 1024 * 1024 // 2MB strict limit to manage storage costs
)

var (
	ErrImageTooLarge     = errors.New("image size exceeds maximum allowed 2MB")
	ErrInvalidImageType  = errors.New("invalid image format; only JPEG, PNG, and WebP are allowed")
	allowedContentTypes  = map[string]bool{
		"image/jpeg": true,
		"image/jpg":  true,
		"image/png":  true,
		"image/webp": true,
	}
)

func NewR2Client() *s3.S3 {
	cfg := utils.MustLoad()
	sess := session.Must(session.NewSession(&aws.Config{
		Region:   aws.String("auto"),
		Endpoint: aws.String(cfg.Endpoint),
		Credentials: credentials.NewStaticCredentials(
			cfg.AccessKey,
			cfg.SecretKey,
			"",
		),
	}))
	return s3.New(sess)
}

// UploadImage uploads an image to Cloudflare R2 under the specified folder with strict size & mime checks
func UploadImage(file multipart.File, header *multipart.FileHeader, folder string) (string, error) {
	if header.Size > MaxImageSizeBytes {
		return "", ErrImageTooLarge
	}

	contentType := header.Header.Get("Content-Type")
	if !allowedContentTypes[strings.ToLower(contentType)] {
		return "", ErrInvalidImageType
	}

	ext := filepath.Ext(header.Filename)
	if ext == "" {
		ext = ".jpg"
	}
	// Deterministic, collision-free filename
	filename := fmt.Sprintf("%d%s", time.Now().UnixNano(), ext)
	key := fmt.Sprintf("%s/%s", strings.Trim(folder, "/"), filename)

	cfg := utils.MustLoad()
	svc := NewR2Client()

	_, err := svc.PutObject(&s3.PutObjectInput{
		Bucket:      aws.String(cfg.Bucket),
		Key:         aws.String(key),
		Body:        file,
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload image to storage: %w", err)
	}

	// Construct public/accessible URL
	if cfg.PublicURL != "" {
		return fmt.Sprintf("%s/%s", strings.TrimRight(cfg.PublicURL, "/"), key), nil
	}
	// Default to backend /images proxy endpoint
	return fmt.Sprintf("/images/%s", key), nil
}

// GetImageStream fetches an object from Cloudflare R2 and returns its stream and MIME content-type
func GetImageStream(key string) (io.ReadCloser, string, error) {
	cfg := utils.MustLoad()
	svc := NewR2Client()

	cleanKey := strings.TrimPrefix(key, "/")
	// If path starts with /images/, strip it
	cleanKey = strings.TrimPrefix(cleanKey, "images/")
	// If path starts with bucket name, strip it
	prefix := cfg.Bucket + "/"
	if strings.HasPrefix(cleanKey, prefix) {
		cleanKey = strings.TrimPrefix(cleanKey, prefix)
	}

	out, err := svc.GetObject(&s3.GetObjectInput{
		Bucket: aws.String(cfg.Bucket),
		Key:    aws.String(cleanKey),
	})
	if err != nil {
		return nil, "", err
	}

	contentType := "image/jpeg"
	if out.ContentType != nil && *out.ContentType != "" {
		contentType = *out.ContentType
	}

	return out.Body, contentType, nil
}

// DeleteImage removes an image from Cloudflare R2 to prevent storage bloat
func DeleteImage(imageURL string) error {
	if imageURL == "" {
		return nil
	}

	u, err := url.Parse(imageURL)
	if err != nil {
		return nil
	}

	cfg := utils.MustLoad()
	path := strings.TrimPrefix(u.Path, "/")
	path = strings.TrimPrefix(path, "images/")
	prefix := cfg.Bucket + "/"
	if strings.HasPrefix(path, prefix) {
		path = strings.TrimPrefix(path, prefix)
	}

	if path == "" {
		return nil
	}

	svc := NewR2Client()
	_, err = svc.DeleteObject(&s3.DeleteObjectInput{
		Bucket: aws.String(cfg.Bucket),
		Key:    aws.String(path),
	})
	return err
}
