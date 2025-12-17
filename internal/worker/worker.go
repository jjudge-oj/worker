package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/joshjms/castletown/internal/cache"
	"github.com/joshjms/castletown/internal/config"
	"github.com/joshjms/castletown/internal/grader"
	"github.com/joshjms/castletown/internal/models"
	"github.com/joshjms/castletown/internal/mq"
	"github.com/joshjms/castletown/internal/repository"
	"github.com/joshjms/castletown/internal/store"
	"github.com/joshjms/castletown/internal/telemetry"
	"github.com/rs/zerolog"
)

type SubmissionGrader interface {
	Handle(ctx context.Context, sub *models.Submission) error
}

type Worker struct {
	log     zerolog.Logger
	metrics *telemetry.Metrics

	g SubmissionGrader

	queueConsumer mq.Consumer
}

func NewWorker(cfg *config.Config) (*Worker, error) {
	problemsRepo, err := repository.NewProblemsRepository(cfg.Database)
	if err != nil {
		return nil, fmt.Errorf("create problems repository: %w", err)
	}

	submissionsRepo, err := repository.NewSubmissionsRepository(cfg.Database)
	if err != nil {
		return nil, fmt.Errorf("create submissions repository: %w", err)
	}

	testcaseStore, err := store.NewTestcaseStore(cfg.Minio)
	if err != nil {
		return nil, fmt.Errorf("create testcase store: %w", err)
	}

	if err := os.MkdirAll(cfg.Judge.DiskCacheDir, 0755); err != nil {
		return nil, fmt.Errorf("create disk cache dir: %w", err)
	}

	problemCache := cache.NewProblemCache(problemsRepo, testcaseStore, 256, cfg.Judge.DiskCacheDir)

	log := zerolog.New(os.Stdout).With().Timestamp().Logger()

	w := &Worker{
		log:     log,
		metrics: telemetry.NewMetricsRegistry(),
		g:       grader.NewGrader(log, cfg, submissionsRepo, problemCache),
	}

	w.queueConsumer = mq.NewConsumer(cfg.RabbitMQ, log, cfg.Judge.MaxConcurrency)

	return w, nil
}

func (w *Worker) Run(ctx context.Context) error {
	if w.queueConsumer == nil {
		<-ctx.Done()
		return ctx.Err()
	}

	w.queueConsumer.Run(ctx, w.handleQueueMessage)
	return ctx.Err()
}

func (w *Worker) handle(ctx context.Context, sub *models.Submission) error {
	if w.g == nil {
		return nil
	}
	return w.g.Handle(ctx, sub)
}

func (w *Worker) handleQueueMessage(ctx context.Context, body []byte) error {
	var sub models.Submission
	if err := json.Unmarshal(body, &sub); err != nil {
		return fmt.Errorf("invalid submission payload: %w", err)
	}
	w.log.Info().Int64("submission_id", sub.ID).Msg("Processing submission")
	if err := w.handle(ctx, &sub); err != nil {
		w.log.Error().Err(err).Int64("submission_id", sub.ID).Msg("Failed to process submission")
		return fmt.Errorf("handle submission: %w", err)
	}
	w.log.Info().Int64("submission_id", sub.ID).Msg("Finished processing submission")
	w.log.Info().Any("submission", sub).Msg("Submission result")
	return nil
}
