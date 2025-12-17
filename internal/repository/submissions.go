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

type TestcaseResultsWriter interface {
	InsertTestcaseResults(ctx context.Context, subID int64, results []models.TestcaseResult) error
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
	_, err := r.db.ExecContext(ctx, `
		UPDATE submissions
		SET verdict = $1,
			score = $2,
			message = $3,
			cpu_time = $4,
			memory = $5,
			tests_passed = $6,
			tests_total = $7,
			updated_at = NOW()
		WHERE id = $8
	`, sub.Verdict, sub.Score, sub.Message, sub.CPUTime, sub.Memory, sub.TestsPassed, sub.TestsTotal, subID)
	return err
}

func (r *SubmissionsRepository) InsertTestcaseResults(ctx context.Context, subID int64, results []models.TestcaseResult) error {
	if len(results) == 0 {
		return nil
	}

	const outputLimit = 200

	stmt, err := r.db.PrepareContext(ctx, `
		INSERT INTO testcase_results (
			submission_id, testcase_id, verdict, cpu_time, memory, input, expected_output, actual_output, error_message
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, res := range results {
		expected := truncateString(res.ExpectedOutput, outputLimit)
		actual := truncateString(res.ActualOutput, outputLimit)

		if _, err := stmt.ExecContext(
			ctx,
			subID,
			res.TestcaseID,
			res.Verdict,
			res.CPUTime,
			res.Memory,
			res.Input,
			expected,
			actual,
			res.ErrorMessage,
		); err != nil {
			return err
		}
	}
	return nil
}

func (r *SubmissionsRepository) Close() error {
	return r.db.Close()
}

func truncateString(s string, limit int) string {
	if limit <= 0 {
		return ""
	}

	runes := []rune(s)
	if len(runes) <= limit {
		return s
	}

	return string(runes[:limit])
}
