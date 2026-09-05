package users

import (
	"fmt"
	"net/http"
	"os"
	"shopMe/internal/middleware"
	"shopMe/internal/reuse"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
)

func (uh *UserHandler) UploadProfileImage(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(middleware.UserIDContextKey).(int)

	// Ek hi file expect karte hain: "image"
	file, fileHeader, err := r.FormFile("image")
	if err != nil {
		reuse.Error(w, http.StatusBadRequest, reuse.ErrInvalidInput, "Image file required")
		return
	}
	defer file.Close()

	if fileHeader.Size > 2*1024*1024 { // 2 MB limit
		reuse.Error(w, http.StatusBadRequest, reuse.ErrInvalidInput,
			"Image must be less than 2MB")
		return
	}

	s3Client := reuse.NewR2Client()

	// Unique key for user image
	objectKey := fmt.Sprintf("users/%d_profile_%s", userID, fileHeader.Filename)

	// Upload to R2
	_, err = s3Client.PutObject(&s3.PutObjectInput{
		Bucket: aws.String(os.Getenv("R2_BUCKET")),
		Key:    aws.String(objectKey),
		Body:   file,
		ACL:    aws.String("public-read"),
	})
	if err != nil {
		reuse.Error(w, http.StatusInternalServerError, reuse.ErrDBFailure, "Upload failed")
		return
	}

	// Public URL
	fileURL := fmt.Sprintf("%s/%s/%s",
		os.Getenv("R2_ENDPOINT"),
		os.Getenv("R2_BUCKET"),
		objectKey,
	)

	// Upsert into DB (replace old image if exists)
	query := `
        INSERT INTO user_images (user_id, image_url, updated_at)
        VALUES ($1, $2, NOW())
        ON CONFLICT (user_id) DO UPDATE
        SET image_url = EXCLUDED.image_url, updated_at = NOW()
    `
	_, err = uh.db.Exec(r.Context(), query, userID, fileURL)
	if err != nil {
		reuse.Error(w, http.StatusInternalServerError, reuse.ErrDBFailure, "DB insert failed")
		return
	}

	reuse.Success(w, "Profile image uploaded successfully", map[string]any{
		"user_id":   userID,
		"image_url": fileURL,
	})
}
