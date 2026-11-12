// Package ai wraps the Google Gemini REST API (generateContent).
package ai

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

const baseURL = "https://generativelanguage.googleapis.com/v1beta/models"

// Image is an optional image attached to a prompt.
type Image struct {
	Data     []byte
	MimeType string
}

type Gemini struct {
	apiKey string
	models []string // primary first, then fallbacks
	http   *http.Client
}

const (
	attemptTimeout   = 30 * time.Second
	attemptsPerModel = 2
	// totalBudget caps retries + fallbacks for one generate call so a user is
	// never left waiting for minutes when every model is overloaded.
	totalBudget = 75 * time.Second
)

// NewGemini creates a client. models[0] is the primary model; the rest are
// fallbacks used when a model is overloaded (429/5xx), times out, or is unavailable.
func NewGemini(apiKey string, models []string) *Gemini {
	return &Gemini{apiKey: apiKey, models: models, http: &http.Client{}}
}

// retryableError marks failures worth retrying / falling back on.
type retryableError struct{ err error }

func (e retryableError) Error() string { return e.err.Error() }
func (e retryableError) Unwrap() error { return e.err }

// ---- request / response types ----

type part struct {
	Text       string      `json:"text,omitempty"`
	InlineData *inlineData `json:"inline_data,omitempty"`
}

type inlineData struct {
	MimeType string `json:"mime_type"`
	Data     string `json:"data"`
}

type content struct {
	Role  string `json:"role,omitempty"`
	Parts []part `json:"parts"`
}

type generationConfig struct {
	Temperature      float64 `json:"temperature"`
	ResponseMimeType string  `json:"responseMimeType,omitempty"`
	ResponseSchema   any     `json:"responseSchema,omitempty"`
}

type generateRequest struct {
	SystemInstruction *content         `json:"system_instruction,omitempty"`
	Contents          []content        `json:"contents"`
	GenerationConfig  generationConfig `json:"generationConfig"`
}

type generateResponse struct {
	Candidates []struct {
		Content content `json:"content"`
	} `json:"candidates"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func (g *Gemini) generate(ctx context.Context, system, prompt string, img *Image, cfg generationConfig) (string, error) {
	parts := []part{}
	if img != nil && len(img.Data) > 0 {
		parts = append(parts, part{InlineData: &inlineData{
			MimeType: img.MimeType,
			Data:     base64.StdEncoding.EncodeToString(img.Data),
		}})
	}
	parts = append(parts, part{Text: prompt})

	body, err := json.Marshal(generateRequest{
		SystemInstruction: &content{Parts: []part{{Text: system}}},
		Contents:          []content{{Role: "user", Parts: parts}},
		GenerationConfig:  cfg,
	})
	if err != nil {
		return "", err
	}

	ctx, cancel := context.WithTimeout(ctx, totalBudget)
	defer cancel()

	var lastErr error
	for _, model := range g.models {
		for attempt := 1; attempt <= attemptsPerModel; attempt++ {
			text, err := g.call(ctx, model, body)
			if err == nil {
				return text, nil
			}
			lastErr = err
			var re retryableError
			if !errors.As(err, &re) || ctx.Err() != nil {
				return "", fmt.Errorf("gemini failed (last model %s): %w", model, err)
			}
			slog.Warn("gemini call failed, retrying", "model", model, "attempt", attempt, "err", err)
			if attempt < attemptsPerModel {
				select {
				case <-time.After(time.Duration(attempt) * 2 * time.Second):
				case <-ctx.Done():
					return "", ctx.Err()
				}
			}
		}
	}
	return "", fmt.Errorf("all gemini models failed: %w", lastErr)
}

func (g *Gemini) call(ctx context.Context, model string, body []byte) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, attemptTimeout)
	defer cancel()

	url := fmt.Sprintf("%s/%s:generateContent", baseURL, model)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-goog-api-key", g.apiKey)

	resp, err := g.http.Do(req)
	if err != nil {
		return "", retryableError{fmt.Errorf("gemini %s request: %w", model, err)}
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	retryable := resp.StatusCode == http.StatusTooManyRequests ||
		resp.StatusCode == http.StatusNotFound || // model retired / not available to this key
		resp.StatusCode >= 500

	var out generateResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		err = fmt.Errorf("gemini %s decode (status %d): %w", model, resp.StatusCode, err)
		if retryable {
			return "", retryableError{err}
		}
		return "", err
	}
	if out.Error != nil {
		err := fmt.Errorf("gemini %s error (status %d): %s", model, resp.StatusCode, out.Error.Message)
		if retryable {
			return "", retryableError{err}
		}
		return "", err
	}
	if len(out.Candidates) == 0 || len(out.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("gemini %s returned no content", model)
	}

	var sb strings.Builder
	for _, p := range out.Candidates[0].Content.Parts {
		sb.WriteString(p.Text)
	}
	return sb.String(), nil
}

// Extraction is the result of the cheap first pass used for cache lookup.
type Extraction struct {
	// Species is the general Thai common name of the plant/animal (e.g. "ข้าว", "ทุเรียน", "วัว"),
	// or "" if it cannot be determined.
	Species  string   `json:"species"`
	Keywords []string `json:"keywords"`
}

// ExtractKeywords identifies the species and short Thai keywords describing the
// main problem (e.g. "ใบเหลือง", "เพลี้ยกระโดด", "ซึม").
func (g *Gemini) ExtractKeywords(ctx context.Context, question string, img *Image) (*Extraction, error) {
	system := `คุณเป็นผู้ช่วยสกัดคำสำคัญด้านการเกษตร (พืชและสัตว์เลี้ยง)
หน้าที่: อ่านคำถามและ/หรือดูรูปภาพ แล้วตอบเป็น JSON
- species: ชื่อสามัญภาษาไทยแบบทั่วไปของพืช/สัตว์ เช่น "ข้าว", "ทุเรียน", "มะม่วง", "วัว", "ไก่"
  (ใช้ชื่อกลางๆ ไม่ต้องระบุสายพันธุ์ย่อย) ถ้าระบุไม่ได้ให้เป็น ""
- keywords: keyword อาการ/ปัญหา 2-4 คำ ภาษาไทยสั้นๆ ไม่ต้องรวมชื่อชนิด
  ครอบคลุม: ส่วนที่มีอาการ, อาการ, และชื่อโรค/ศัตรูพืชถ้ามั่นใจ
  เช่น ["ใบเหลือง", "เพลี้ยกระโดดสีน้ำตาล"] หรือ ["ซึม", "ไม่กินอาหาร"]
ใช้คำมาตรฐาน ไม่ใส่คำฟุ่มเฟือย`

	prompt := "คำถาม: " + question
	if strings.TrimSpace(question) == "" {
		prompt = "ไม่มีคำถาม ให้สกัดข้อมูลจากรูปภาพ"
	}

	text, err := g.generate(ctx, system, prompt, img, generationConfig{
		Temperature:      0,
		ResponseMimeType: "application/json",
		ResponseSchema: map[string]any{
			"type": "OBJECT",
			"properties": map[string]any{
				"species":  map[string]any{"type": "STRING"},
				"keywords": map[string]any{"type": "ARRAY", "items": map[string]any{"type": "STRING"}},
			},
			"required": []string{"species", "keywords"},
		},
	})
	if err != nil {
		return nil, err
	}

	var out Extraction
	if err := json.Unmarshal([]byte(text), &out); err != nil {
		return nil, fmt.Errorf("parse extraction %q: %w", text, err)
	}
	return &out, nil
}

// Diagnosis is the full AI analysis.
type Diagnosis struct {
	SpeciesName    string `json:"species_name"`    // ชื่อไทย เช่น "ข้าว", "ทุเรียนหมอนทอง"
	ScientificName string `json:"scientific_name"` // เช่น "Oryza sativa"
	Answer         string `json:"answer"`          // Markdown
}

// Diagnose identifies the species, analyses the problem and returns treatment advice.
func (g *Gemini) Diagnose(ctx context.Context, question string, img *Image) (*Diagnosis, error) {
	system := `คุณเป็นผู้เชี่ยวชาญด้านโรคพืชและสุขภาพสัตว์เลี้ยงในฟาร์มของไทย
ตอบเป็น JSON:
- species_name: ระบุว่าเป็นพืช/สัตว์ชนิดอะไร เป็นชื่อภาษาไทย (ระบุสายพันธุ์ได้ถ้ามั่นใจ เช่น "ทุเรียนหมอนทอง")
  ถ้าระบุไม่ได้ให้เป็น "ไม่สามารถระบุได้"
- scientific_name: ชื่อวิทยาศาสตร์ เช่น "Oryza sativa" ถ้าไม่ทราบให้เป็น ""
- answer: คำตอบภาษาไทย Markdown กระชับ ใช้ภาษาง่ายเหมือนพูดกับผู้สูงอายุ เลี่ยงศัพท์เทคนิค
  (ถ้าจำเป็นต้องใช้ให้อธิบายสั้นๆ) ใช้ข้อสั้นๆ ไม่เกิน 3-4 ข้อต่อหัวข้อ
  ไม่ต้องเขียนชื่อชนิดซ้ำเป็นหัวข้อ (แสดงแยกอยู่แล้ว) ในรูปแบบ:
  ## วินิจฉัยเบื้องต้น
  ## สาเหตุที่เป็นไปได้
  ## วิธีแก้ไข / รักษา
  ## การป้องกัน
  ถ้าข้อมูลไม่พอให้บอกว่าต้องการข้อมูลอะไรเพิ่ม (เช่น ขอรูปใบ/ลำต้นให้ชัดขึ้น)
  และถ้ารุนแรงให้แนะนำปรึกษาเจ้าหน้าที่เกษตรหรือสัตวแพทย์`

	prompt := question
	if strings.TrimSpace(prompt) == "" {
		prompt = "ช่วยระบุชนิดและวิเคราะห์โรคหรืออาการจากรูปนี้ พร้อมวิธีแก้"
	}

	text, err := g.generate(ctx, system, prompt, img, generationConfig{
		Temperature:      0.3,
		ResponseMimeType: "application/json",
		ResponseSchema: map[string]any{
			"type": "OBJECT",
			"properties": map[string]any{
				"species_name":    map[string]any{"type": "STRING"},
				"scientific_name": map[string]any{"type": "STRING"},
				"answer":          map[string]any{"type": "STRING"},
			},
			"required": []string{"species_name", "scientific_name", "answer"},
		},
	})
	if err != nil {
		return nil, err
	}

	var d Diagnosis
	if err := json.Unmarshal([]byte(text), &d); err != nil {
		return nil, fmt.Errorf("parse diagnosis: %w", err)
	}
	if strings.TrimSpace(d.Answer) == "" {
		return nil, fmt.Errorf("gemini returned empty answer")
	}
	return &d, nil
}
