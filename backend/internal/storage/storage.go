// Package storage uploads images to an external object store and returns a public URL.
package storage

import (
	"context"
	"fmt"

	"github.com/plantdoc/backend/internal/config"
)

type Uploader interface {
	Upload(ctx context.Context, filename string, data []byte, mimeType string) (url string, err error)
}

func New(cfg *config.Config) (Uploader, error) {
	switch cfg.StorageProvider {
	case "cloudinary":
		if cfg.CloudinaryCloudName == "" || cfg.CloudinaryAPIKey == "" || cfg.CloudinaryAPISecret == "" {
			return nil, fmt.Errorf("cloudinary credentials are required")
		}
		return NewCloudinary(cfg.CloudinaryCloudName, cfg.CloudinaryAPIKey, cfg.CloudinaryAPISecret, cfg.CloudinaryFolder), nil
	case "supabase":
		if cfg.SupabaseURL == "" || cfg.SupabaseServiceKey == "" {
			return nil, fmt.Errorf("SUPABASE_URL and SUPABASE_SERVICE_KEY are required")
		}
		return NewSupabase(cfg.SupabaseURL, cfg.SupabaseServiceKey, cfg.SupabaseBucket), nil
	default:
		return nil, fmt.Errorf("unknown STORAGE_PROVIDER %q (use cloudinary or supabase)", cfg.StorageProvider)
	}
}
