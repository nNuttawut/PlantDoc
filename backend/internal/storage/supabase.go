package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"path"
	"time"
)

type Supabase struct {
	baseURL, serviceKey, bucket string
	http                        *http.Client
}

func NewSupabase(baseURL, serviceKey, bucket string) *Supabase {
	return &Supabase{baseURL, serviceKey, bucket, &http.Client{Timeout: 60 * time.Second}}
}

// Upload stores the object in a PUBLIC bucket and returns its public URL.
func (s *Supabase) Upload(ctx context.Context, filename string, data []byte, mimeType string) (string, error) {
	objectPath := fmt.Sprintf("%s/%d%s", time.Now().Format("2006/01"), time.Now().UnixNano(), path.Ext(filename))
	url := fmt.Sprintf("%s/storage/v1/object/%s/%s", s.baseURL, s.bucket, objectPath)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+s.serviceKey)
	req.Header.Set("apikey", s.serviceKey)
	req.Header.Set("Content-Type", mimeType)

	resp, err := s.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("supabase upload: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("supabase upload failed (%d): %s", resp.StatusCode, b)
	}
	return fmt.Sprintf("%s/storage/v1/object/public/%s/%s", s.baseURL, s.bucket, objectPath), nil
}
