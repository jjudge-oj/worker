package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/joshjms/castletown/internal/config"
	"github.com/joshjms/castletown/internal/models"
	"github.com/lib/pq"
)

type ProblemsRepository struct {
	db *sql.DB
}

func NewProblemsRepository(cfg config.DatabaseConfig) (*ProblemsRepository, error) {
	db, err := sql.Open("postgres", cfg.DSN())
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}
	return &ProblemsRepository{db: db}, nil
}

func (r *ProblemsRepository) GetProblemDetails(ctx context.Context, problemID int) (*models.Problem, error) {
	const query = `
		SELECT
			p.id,
			p.time_limit,
			p.memory_limit,
			p.tags,
			p.created_at,
			p.updated_at,
			g.id,
			g.order_id,
			g.name,
			g.points,
			t.id,
			t.order_id,
			t.testcase_group_id,
			t.input,
			t.output,
			t.is_hidden
		FROM problems p
		LEFT JOIN testcase_groups g ON p.id = g.problem_id
		LEFT JOIN testcases t ON g.id = t.testcase_group_id
		WHERE p.id = $1
		ORDER BY g.order_id NULLS LAST, t.order_id NULLS LAST;
	`

	rows, err := r.db.QueryContext(ctx, query, problemID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var (
		p          *models.Problem
		groupByID  = map[int]*models.TestcaseGroup{}
		groupOrder []int
	)

	for rows.Next() {
		// Problem fields
		var (
			pID         int
			timeLimit   int64
			memoryLimit int64
			tags        []string
			createdAt   time.Time
			updatedAt   time.Time
		)

		// Group fields (nullable because LEFT JOIN)
		var (
			gID      sql.NullInt64
			gName    sql.NullString
			gPoints  sql.NullInt64
			gOrderID sql.NullInt64
		)

		// Testcase fields (nullable because LEFT JOIN)
		var (
			tID      sql.NullInt64
			tGroupID sql.NullInt64
			tOrderID sql.NullInt64
			tInput   sql.NullString
			tOutput  sql.NullString
			tHidden  sql.NullBool
		)

		if err := rows.Scan(
			&pID,
			&timeLimit,
			&memoryLimit,
			pq.Array(&tags),
			&createdAt,
			&updatedAt,
			&gID,
			&gOrderID,
			&gName,
			&gPoints,
			&tID,
			&tOrderID,
			&tGroupID,
			&tInput,
			&tOutput,
			&tHidden,
		); err != nil {
			return nil, err
		}

		// Create Problem once
		if p == nil {
			p = &models.Problem{
				ID:          pID,
				TimeLimit:   timeLimit,
				MemoryLimit: memoryLimit,
				Tags:        tags,
				CreatedAt:   createdAt,
				UpdatedAt:   updatedAt,
			}
		}

		// If there is a group row, ensure group exists in map
		var grp *models.TestcaseGroup
		if gID.Valid {
			id := int(gID.Int64)
			grp = groupByID[id]
			if grp == nil {
				grp = &models.TestcaseGroup{
					ID:        id,
					OrderID:   int(gOrderID.Int64),
					ProblemID: pID,
					Name:      gName.String,
					Points:    int(gPoints.Int64),
				}
				groupByID[id] = grp
				groupOrder = append(groupOrder, id)
			}
		}

		// If there is a testcase row, attach it
		if tID.Valid {
			tc := models.Testcase{
				ID:              int(tID.Int64),
				OrderID:         int(tOrderID.Int64),
				TestcaseGroupID: int(tGroupID.Int64),
				Input:           tInput.String,
				Output:          tOutput.String,
				IsHidden:        tHidden.Valid && tHidden.Bool,
			}

			if grp != nil {
				grp.Testcases = append(grp.Testcases, tc)
			}
		}
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}
	if p == nil {
		return nil, sql.ErrNoRows
	}

	// Move groups from map into slice, preserving order
	p.TestcaseGroups = make([]models.TestcaseGroup, 0, len(groupOrder))
	for _, id := range groupOrder {
		p.TestcaseGroups = append(p.TestcaseGroups, *groupByID[id])
	}

	return p, nil
}

func (r *ProblemsRepository) GetTestcases(ctx context.Context, problemID int) ([]models.Testcase, error) {
	const query = `
		SELECT
			t.id,
			t.order_id,
			t.testcase_group_id,
			t.input,
			t.output,
			t.is_hidden
		FROM testcases t
		JOIN testcase_groups g ON t.testcase_group_id = g.id
		WHERE g.problem_id = $1
		ORDER BY g.order_id, t.order_id;
	`

	rows, err := r.db.QueryContext(ctx, query, problemID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var testcases []models.Testcase
	for rows.Next() {
		var tc models.Testcase
		if err := rows.Scan(
			&tc.ID,
			&tc.OrderID,
			&tc.TestcaseGroupID,
			&tc.Input,
			&tc.Output,
			&tc.IsHidden,
		); err != nil {
			return nil, err
		}
		testcases = append(testcases, tc)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}
	return testcases, nil
}

func (r *ProblemsRepository) UpdateSubmissionResult(ctx context.Context, subID int, sub *models.Submission) error {
	_, err := r.db.ExecContext(ctx, "UPDATE submissions SET verdict = $1, score = $2, message = $3 WHERE id = $4", sub.Verdict, sub.Score, sub.Message, subID)
	return err
}

func (r *ProblemsRepository) Close() error {
	return r.db.Close()
}
