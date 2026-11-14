// Package main provides a quick developer CLI tool to publish app versions directly into PostgreSQL.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"strings"
	"time"

	"shopMe/internal/handler/model"
	"shopMe/internal/handler/repository"
	"shopMe/internal/utils"
)

func main() {
	appType := flag.String("app", "merchant_pos", "Target application (merchant_pos, customer_app)")
	platform := flag.String("platform", "android", "Platform (android, windows, ios)")
	code := flag.Int("code", 0, "Build version code (integer, e.g. 102)")
	name := flag.String("name", "", "Display version name (string, e.g. 1.0.2)")
	apk := flag.String("apk", "", "Relative or absolute APK download URL (e.g. /downloads/shopsilo-merchant-v1.0.2.apk)")
	size := flag.Int64("size", 28500000, "File size in bytes (e.g. 28500000)")
	minCode := flag.Int("min", 100, "Minimum supported version code threshold (e.g. 100)")
	force := flag.Bool("force", false, "Whether this update is strictly mandatory (true/false)")
	notesStr := flag.String("notes", "", "Comma-separated release notes (e.g. 'Feature 1, Feature 2')")

	flag.Parse()

	if *code <= 0 || *name == "" || *apk == "" {
		fmt.Println("Usage Example:")
		fmt.Println(`  go run cmd/publish_release/main.go -code 102 -name "1.0.2" -apk "/downloads/shopsilo-merchant-v1.0.2.apk" -notes "⚡ वॉइस बिलिंग, 🗄️ सेफ ड्रॉप"`)
		return
	}

	// Load DB config from .env
	cfg := utils.MustLoad()
	db, err := utils.ConnectDB(cfg.DBURL)
	if err != nil {
		log.Fatalf("❌ Database connection failed: %v", err)
	}
	defer db.Close()

	// Parse notes
	var notes []string
	if *notesStr != "" {
		parts := strings.Split(*notesStr, ",")
		for _, p := range parts {
			trimmed := strings.TrimSpace(p)
			if trimmed != "" {
				notes = append(notes, trimmed)
			}
		}
	}

	repo := repository.NewAppVersionRepo(db, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	v := &model.AppVersion{
		AppType:                 *appType,
		Platform:                *platform,
		VersionCode:             *code,
		VersionName:             *name,
		MinSupportedVersionCode: *minCode,
		ApkURL:                  *apk,
		FileSizeBytes:           *size,
		ReleaseNotes:            notes,
		IsForceUpdate:           *force,
		IsActive:                true,
	}

	published, err := repo.PublishVersion(ctx, v)
	if err != nil {
		log.Fatalf("❌ Failed to publish release: %v", err)
	}

	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("✅ App Release v%s (Code %d) Published Successfully!\n", published.VersionName, published.VersionCode)
	fmt.Printf("   App Type:    %s\n", published.AppType)
	fmt.Printf("   Platform:    %s\n", published.Platform)
	fmt.Printf("   Download:    %s\n", published.ApkURL)
	fmt.Printf("   Force Lock:  %v\n", published.IsForceUpdate)
	fmt.Printf("   Release ID:  %d\n", published.ID)
	fmt.Println("   All connected merchant devices will now receive this update!")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
}
