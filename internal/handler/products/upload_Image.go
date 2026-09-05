package products

import (
	"fmt"
	"net/http"
	"os"
	"shopMe/internal/reuse"
	"strconv"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
)

func (ph *ProductHandler) UploadImage(w http.ResponseWriter, r *http.Request) {
	// Product ID from form-data
	productIDStr := r.FormValue("product_id")
	productID, err := strconv.Atoi(productIDStr)
	if err != nil {
		reuse.Error(w, http.StatusBadRequest, reuse.ErrInvalidInput, "Invalid product ID")
		return
	}

	// Parse multipart form
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		reuse.Error(w, http.StatusBadRequest, reuse.ErrInvalidInput, "File size must be < 10MB")
		return
	}

	// Verify product exists and get shop_id
	var shopID int
	query := `SELECT shop_id FROM products WHERE id=$1`
	err = ph.db.QueryRow(r.Context(), query, productID).Scan(&shopID)
	if err != nil {
		reuse.Error(w, http.StatusNotFound, reuse.ErrNotFound, "Product not found")
		return
	}
	// Check existing images in DB
	var count int
	err = ph.db.QueryRow(r.Context(),
		"SELECT COUNT(*) FROM product_images WHERE product_id=$1", productID).Scan(&count)
	if err != nil {
		reuse.Error(w, http.StatusInternalServerError, reuse.ErrDBFailure, "DB check failed")
		return
	}
	if count >= 5 {
		reuse.Error(w, http.StatusBadRequest, reuse.ErrInvalidInput,
			"Maximum 5 images allowed per shop")
		return
	}
	files := r.MultipartForm.File["images"]
	uploaded := []string{}

	s3Client := reuse.NewR2Client()

	for _, fileHeader := range files {
		file, err := fileHeader.Open()
		if err != nil {
			fmt.Println("File open error:", err)
			continue
		}
		defer file.Close()

		if fileHeader.Size > 2*1024*1024 {
			reuse.Error(w, http.StatusBadRequest, reuse.ErrInvalidInput,
				"Each image must be less than 2MB")
			return
		}

		// Unique key with product_id
		objectKey := fmt.Sprintf("products/%d_%s", productID, fileHeader.Filename)

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

		fileURL := fmt.Sprintf("%s/%s/%s",
			os.Getenv("R2_ENDPOINT"),
			os.Getenv("R2_BUCKET"),
			objectKey,
		)

		// Save in DB
		query2 := `INSERT INTO product_images (product_id, image_url) VALUES ($1, $2)`
		_, err = ph.db.Exec(r.Context(), query2, productID, fileURL)
		if err != nil {
			fmt.Println("DB insert error:", err)
			continue
		}

		uploaded = append(uploaded, fileURL)
	}

	reuse.Success(w, "Product images uploaded successfully", map[string]any{
		"product_id": productID,
		"shop_id":    shopID,
		"images":     uploaded,
	})
}
