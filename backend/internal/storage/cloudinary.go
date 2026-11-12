package storage

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"strconv"
	"time"
)

type Cloudinary struct {
	cloudName, apiKey, apiSecret, folder string
	http                                 *http.Client
}

func NewCloudinary(cloudName, apiKey, apiSecret, folder string) *Cloudinary {
	return &Cloudinary{cloudName, apiKey, apiSecret, folder, &http.Client{Timeout: 60 * time.Second}}
}

// Upload uses the signed upload API and returns a URL with automatic
// format/quality optimisation (f_auto,q_auto -> WebP/AVIF when supported).
func (c *Cloudinary) Upload(ctx context.Context, filename string, data []byte, _ string) (string, error) {
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	// Signature: params sorted alphabetically, joined with &, then append secret.
	toSign := fmt.Sprintf("folder=%s&timestamp=%s%s", c.folder, ts, c.apiSecret)
	sum := sha1.Sum([]byte(toSign))

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	_ = w.WriteField("api_key", c.apiKey)
	_ = w.WriteField("timestamp", ts)
	_ = w.WriteField("folder", c.folder)
	_ = w.WriteField("signature", hex.EncodeToString(sum[:]))
	fw, err := w.CreateFormFile("file", filename)
	if err != nil {
		return "", err
	}
	if _, err := fw.Write(data); err != nil {
		return "", err
	}
	w.Close()

	url := fmt.Sprintf("https://api.cloudinary.com/v1_1/%s/image/upload", c.cloudName)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, &buf)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())

	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("cloudinary upload: %w", err)
	}
	defer resp.Body.Close()

	var out struct {
		PublicID  string `json:"public_id"`
		SecureURL string `json:"secure_url"`
		Error     *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", fmt.Errorf("cloudinary decode (status %d): %w", resp.StatusCode, err)
	}
	if out.Error != nil {
		return "", fmt.Errorf("cloudinary error: %s", out.Error.Message)
	}
	return fmt.Sprintf("https://res.cloudinary.com/%s/image/upload/f_auto,q_auto/%s", c.cloudName, out.PublicID), nil
}
