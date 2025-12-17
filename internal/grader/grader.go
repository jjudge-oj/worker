package grader

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/joshjms/castletown/internal/cache"
	"github.com/joshjms/castletown/internal/config"
	"github.com/joshjms/castletown/internal/container"
	"github.com/joshjms/castletown/internal/models"
	"github.com/joshjms/castletown/internal/repository"
	"github.com/rs/zerolog"
)

var handlerMap = map[string]func(*Grader, context.Context, *models.Submission, *models.Problem, string) error{
	"cpp": (*Grader).handleCppSubmission,
}

type Grader struct {
	log                   zerolog.Logger
	runtimeCfg            *config.Config
	submissionsRepository *repository.SubmissionsRepository
	problemCache          *cache.ProblemCache
	slotPool              *container.SlotPool
}

func NewGrader(log zerolog.Logger, runtimeCfg *config.Config, submissionsRepository *repository.SubmissionsRepository, store *cache.ProblemCache) *Grader {
	return &Grader{
		log:                   log,
		runtimeCfg:            runtimeCfg,
		submissionsRepository: submissionsRepository,
		problemCache:          store,
		slotPool:              container.NewSlotPool(container.WithMaxConcurrency(runtimeCfg.Judge.MaxConcurrency)),
	}
}

func (g *Grader) Handle(ctx context.Context, sub *models.Submission) error {
	if sub == nil {
		return errors.New("submission is nil")
	}

	var (
		problem *models.Problem
		err     error
	)

	if err := g.markSubmissionAsGrading(ctx, sub); err != nil {
		g.log.Error().Err(err).Int64("submission_id", sub.ID).Msg("Failed to mark submission as grading")
	}

	var release func()
	problem, release, err = g.problemCache.GetProblemWithLease(ctx, int(sub.ProblemID))
	if release != nil {
		defer release()
	}
	if err != nil {
		_ = g.markSubmissionWithVerdict(ctx, sub, models.VerdictInternalError, 0, "failed to prepare submission for judging")
		return fmt.Errorf("failed to get problem with lease: %w", err)
	}

	submissionDir := filepath.Join(g.runtimeCfg.Judge.WorkRoot, fmt.Sprintf("submission_%d", sub.ID))
	if err := os.Mkdir(submissionDir, 0700); err != nil {
		return err
	}
	defer os.RemoveAll(submissionDir)

	handler, ok := handlerMap[sub.Language]
	if !ok {
		return errors.New("no handler for language: " + sub.Language)
	}

	if err := handler(g, ctx, sub, problem, submissionDir); err != nil {
		_ = g.markSubmissionWithVerdict(ctx, sub, models.VerdictSystemError, 0, "failed to judge submission")
		return err
	}

	if err := g.persistSubmission(ctx, sub); err != nil {
		g.log.Error().Err(err).Int64("submission_id", sub.ID).Msg("Failed to persist submission result")
		return err
	}

	return nil
}
