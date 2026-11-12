// Package service implements the ask/diagnose workflow:
//
//	extract keywords -> search DB -> (hit) return cached answer
//	                              -> (miss) ask AI -> upload image -> save -> return
package service

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/plantdoc/backend/internal/ai"
	"github.com/plantdoc/backend/internal/repository"
	"github.com/plantdoc/backend/internal/storage"
)

type AskInput struct {
	Question      string
	Image         *ai.Image
	ImageFilename string
}

type AskResult struct {
	ID             string   `json:"id"`
	SpeciesName    string   `json:"species_name"`    // ชนิดพืช/สัตว์ที่ AI ระบุ
	ScientificName string   `json:"scientific_name"` // ชื่อวิทยาศาสตร์
	Answer         string   `json:"answer"`
	Keywords       []string `json:"keywords"`
	ImageURL       *string  `json:"image_url"`
	Cached         bool     `json:"cached"`
	Verified       bool     `json:"verified"`
}

type DiagnosisService struct {
	ai       *ai.Gemini
	repo     *repository.ConsultationRepo
	uploader storage.Uploader
}

func NewDiagnosisService(g *ai.Gemini, repo *repository.ConsultationRepo, up storage.Uploader) *DiagnosisService {
	return &DiagnosisService{ai: g, repo: repo, uploader: up}
}

func (s *DiagnosisService) Ask(ctx context.Context, in AskInput) (*AskResult, error) {
	// 1. Identify species + extract keywords with AI.
	ext, err := s.ai.ExtractKeywords(ctx, in.Question, in.Image)
	if err != nil {
		return nil, fmt.Errorf("extract keywords: %w", err)
	}
	species := normalizeOne(ext.Species)
	keywords := normalize(ext.Keywords)

	// 2. Look for an existing answer for the same species.
	if len(keywords) > 0 {
		match, err := s.repo.FindBestMatch(ctx, species, keywords, minOverlap(len(keywords)))
		if err != nil {
			slog.Warn("keyword lookup failed", "err", err)
		} else if match != nil {
			return &AskResult{
				ID:             match.ID,
				SpeciesName:    deref(match.SpeciesName),
				ScientificName: deref(match.ScientificName),
				Answer:         match.AIAnswer,
				Keywords:       match.Keywords,
				ImageURL:       match.ImageURL,
				Cached:         true,
				Verified:       match.IsVerified,
			}, nil
		}
	}

	// 3. Not found: ask AI to identify species, diagnose and suggest treatment.
	d, err := s.ai.Diagnose(ctx, in.Question, in.Image)
	if err != nil {
		return nil, fmt.Errorf("diagnose: %w", err)
	}

	// 4. Upload image (only for new questions, to save storage).
	var imageURL *string
	if in.Image != nil {
		url, err := s.uploader.Upload(ctx, in.ImageFilename, in.Image.Data, in.Image.MimeType)
		if err != nil {
			slog.Warn("image upload failed; saving without image", "err", err)
		} else {
			imageURL = &url
		}
	}

	// 5. Save to the knowledge base.
	c := &repository.Consultation{
		QuestionText:   in.Question,
		Keywords:       keywords,
		Species:        ptr(species),
		SpeciesName:    ptr(strings.TrimSpace(d.SpeciesName)),
		ScientificName: ptr(strings.TrimSpace(d.ScientificName)),
		ImageURL:       imageURL,
		AIAnswer:       d.Answer,
	}
	if err := s.repo.Create(ctx, c); err != nil {
		slog.Error("save consultation failed", "err", err)
	}

	return &AskResult{
		ID:             c.ID,
		SpeciesName:    deref(c.SpeciesName),
		ScientificName: deref(c.ScientificName),
		Answer:         d.Answer,
		Keywords:       keywords,
		ImageURL:       imageURL,
	}, nil
}

func (s *DiagnosisService) Recent(ctx context.Context, limit int) ([]repository.Consultation, error) {
	return s.repo.ListRecent(ctx, limit)
}

// minOverlap: species must already match; additionally require most symptom
// keywords to match so different problems on the same crop are not conflated.
func minOverlap(n int) int {
	switch {
	case n <= 1:
		return 1
	case n <= 3:
		return 2
	default:
		return 3
	}
}

func normalizeOne(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

// ptr returns nil for empty strings so they are stored as NULL.
func ptr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func normalize(in []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(in))
	for _, k := range in {
		k = strings.ToLower(strings.TrimSpace(k))
		if k == "" || seen[k] {
			continue
		}
		seen[k] = true
		out = append(out, k)
	}
	return out
}
