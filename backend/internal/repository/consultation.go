// Package repository holds the PostgreSQL (Supabase) data access layer.
package repository

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Consultation maps to public.consultations (see migrations/001_init.sql).
// question_embedding is not loaded here; it is reserved for future semantic search.
type Consultation struct {
	ID             string    `json:"id"`
	UserID         *string   `json:"user_id"`
	QuestionText   string    `json:"question"`
	ImageURL       *string   `json:"image_url"`
	Keywords       []string  `json:"keywords"`
	Species        *string   `json:"species"`         // general name used for matching, e.g. "ข้าว"
	SpeciesName    *string   `json:"species_name"`    // AI-identified name, e.g. "ข้าวหอมมะลิ"
	ScientificName *string   `json:"scientific_name"` // e.g. "Oryza sativa"
	AIAnswer       string    `json:"answer"`
	IsVerified     bool      `json:"is_verified"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type ConsultationRepo struct {
	db *pgxpool.Pool
}

func NewConsultationRepo(db *pgxpool.Pool) *ConsultationRepo {
	return &ConsultationRepo{db: db}
}

const selectCols = `id::text, user_id::text, question_text, image_url, COALESCE(keywords, '{}'),
	species, species_name, scientific_name, ai_answer, COALESCE(is_verified, false), created_at, updated_at`

func scan(row pgx.Row) (*Consultation, error) {
	var c Consultation
	err := row.Scan(&c.ID, &c.UserID, &c.QuestionText, &c.ImageURL, &c.Keywords,
		&c.Species, &c.SpeciesName, &c.ScientificName, &c.AIAnswer, &c.IsVerified, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// FindBestMatch returns the consultation for the same species sharing the most
// keywords, provided at least minOverlap keywords match. Admin-verified answers
// win ties. An empty species only matches rows whose species is also unknown.
// Returns (nil, nil) when nothing matches.
func (r *ConsultationRepo) FindBestMatch(ctx context.Context, species string, keywords []string, minOverlap int) (*Consultation, error) {
	row := r.db.QueryRow(ctx, `
		SELECT `+selectCols+`
		FROM (
			SELECT *, cardinality(ARRAY(
				SELECT unnest(keywords) INTERSECT SELECT unnest($1::text[])
			)) AS overlap
			FROM public.consultations
			WHERE keywords && $1::text[]
			  AND COALESCE(species, '') = $3
		) c
		WHERE overlap >= $2
		ORDER BY overlap DESC, is_verified DESC, created_at DESC
		LIMIT 1`, keywords, minOverlap, species)

	c, err := scan(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return c, err
}

func (r *ConsultationRepo) Create(ctx context.Context, c *Consultation) error {
	return r.db.QueryRow(ctx, `
		INSERT INTO public.consultations
			(user_id, question_text, image_url, keywords, species, species_name, scientific_name, ai_answer)
		VALUES ($1::uuid, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id::text, COALESCE(is_verified, false), created_at, updated_at`,
		c.UserID, c.QuestionText, c.ImageURL, c.Keywords, c.Species, c.SpeciesName, c.ScientificName, c.AIAnswer,
	).Scan(&c.ID, &c.IsVerified, &c.CreatedAt, &c.UpdatedAt)
}

func (r *ConsultationRepo) ListRecent(ctx context.Context, limit int) ([]Consultation, error) {
	rows, err := r.db.Query(ctx, `SELECT `+selectCols+` FROM public.consultations ORDER BY created_at DESC LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := []Consultation{}
	for rows.Next() {
		c, err := scan(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, *c)
	}
	return list, rows.Err()
}
