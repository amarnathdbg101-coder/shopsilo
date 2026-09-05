package shop

import (
	"fmt"
	"net/http"
	"os"
	"shopMe/internal/middleware"
	"shopMe/internal/reuse"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
)
func (sh *ShopHandler) UploadImage(w http.ResponseWriter, r *http.Request) {
    userID := r.Context().Value(middleware.UserIDContextKey).(int)

    // Find shop id for this user
    var shopID int
    query := `SELECT id FROM shops WHERE user_id=$1`
    err := sh.db.QueryRow(r.Context(), query, userID).Scan(&shopID)
    if err != nil {
        reuse.Error(w, http.StatusNotFound, reuse.ErrNotFound, "Shop not found for this user")
        return
    }

    // Parse multipart form (10 MB limit)
    if err := r.ParseMultipartForm(10 << 20); err != nil {
        reuse.Error(w, http.StatusBadRequest, reuse.ErrInvalidInput, "File size must be < 10MB")
        return
    }

    files := r.MultipartForm.File["images"]
    uploaded := []string{}

    // Max 5 files per request
    maxFiles := 5
    if len(files) > maxFiles {
        reuse.Error(w, http.StatusBadRequest, reuse.ErrInvalidInput,
            fmt.Sprintf("You can upload maximum %d images", maxFiles))
        return
    }

    // Check existing images in DB
    var count int
    err = sh.db.QueryRow(r.Context(),
        "SELECT COUNT(*) FROM shop_images WHERE shop_id=$1", shopID).Scan(&count)
    if err != nil {
        reuse.Error(w, http.StatusInternalServerError, reuse.ErrDBFailure, "DB check failed")
        return
    }
    if count >= 5 {
        reuse.Error(w, http.StatusBadRequest, reuse.ErrInvalidInput,
            "Maximum 5 images allowed per shop")
        return
    }

    s3Client := reuse.NewR2Client() // helper that returns *s3.S3

    for _, fileHeader := range files {
        file, err := fileHeader.Open()
        if err != nil {
            fmt.Println("File open error:", err)
            continue
        }
        defer file.Close()

        if fileHeader.Size > 2*1024*1024 { // 2 MB per file
            reuse.Error(w, http.StatusBadRequest, reuse.ErrInvalidInput,
                "Each image must be less than 2MB")
            return
        }

        // Unique key for R2
        objectKey := fmt.Sprintf("shopsilo/%d_%s", shopID, fileHeader.Filename)

        // Upload to R2
        _, err = s3Client.PutObject(&s3.PutObjectInput{
            Bucket: aws.String(os.Getenv("R2_BUCKET")),
            Key:    aws.String(objectKey),
            Body:   file,
            ACL:    aws.String("public-read"),
        })
        if err != nil {
            fmt.Println("R2 upload error:", err)
            continue
        }

        // Public URL
        fileURL := fmt.Sprintf("%s/%s/%s",
            os.Getenv("R2_ENDPOINT"),
            os.Getenv("R2_BUCKET"),
            objectKey,
        )

        // Save path in DB
        query2 := `INSERT INTO shop_images (shop_id, image_url) VALUES ($1, $2)`
        _, err = sh.db.Exec(r.Context(), query2, shopID, fileURL)
        if err != nil {
            fmt.Println("DB insert error:", err)
            continue
        }

        uploaded = append(uploaded, fileURL)
    }

    if len(uploaded) == 0 {
        reuse.Error(w, http.StatusInternalServerError, reuse.ErrDBFailure,
            "No images were uploaded successfully")
        return
    }

    reuse.Success(w, "Images uploaded successfully", map[string]any{"images": uploaded})
}
