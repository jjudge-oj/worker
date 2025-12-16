package repository

import (
	"context"
	"database/sql"

	"github.com/joshjms/castletown/internal/config"
	"github.com/joshjms/castletown/internal/models"
)

type SubmissionsRepository struct {
	db *sql.DB
}

func NewSubmissionsRepository(cfg config.DatabaseConfig) (*SubmissionsRepository, error) {
	db, err := sql.Open("postgres", cfg.DSN())
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}
	return &SubmissionsRepository{db: db}, nil
}

func (r *SubmissionsRepository) UpdateSubmissionResult(ctx context.Context, subID int, sub *models.Submission) error {
	_, err := r.db.ExecContext(ctx, "UPDATE submissions SET verdict = $1, score = $2, message = $3 WHERE id = $4", sub.Verdict, sub.Score, sub.Message, subID)
	return err
}

func (r *SubmissionsRepository) Close() error {
	return r.db.Close()
}
