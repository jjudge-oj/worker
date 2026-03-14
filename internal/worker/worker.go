package worker

import (
	"context"
	"encoding/json"
	"log"

	"github.com/jjudge-oj/api/types"
	"github.com/jjudge-oj/worker/config"
	"github.com/jjudge-oj/worker/internal/blob"
	"github.com/jjudge-oj/worker/internal/grader"
	"github.com/jjudge-oj/worker/internal/lime"
	"github.com/jjudge-oj/worker/internal/mq"
	"github.com/jjudge-oj/worker/internal/tccache"
)

const (
	resultQueue              = "submission-results"
	contestSubmissionQueue   = "contest-submissions"
	contestResultQueue       = "contest-submission-results"
)

// Worker consumes submission jobs from the queue, judges them, and publishes results.
type Worker struct {
	cfg      *config.Config
	mq       *mq.MQ
	grader   *grader.Client
	blob     *blob.Storage
	tccache  *tccache.TestcaseCache
	slotPool *lime.SlotPool
}

// New constructs a Worker with all required dependencies.
func New(cfg *config.Config, mqClient *mq.MQ, graderClient *grader.Client, blobStorage *blob.Storage, tc *tccache.TestcaseCache, sp *lime.SlotPool) *Worker {
	return &Worker{
		cfg:      cfg,
		mq:       mqClient,
		grader:   graderClient,
		blob:     blobStorage,
		tccache:  tc,
		slotPool: sp,
	}
}

// Start subscribes to the configured queue and processes jobs until ctx is cancelled.
func (w *Worker) Start(ctx context.Context) error {
	queue := w.cfg.RabbitMQ.Queue
	log.Printf("worker: subscribing to %q queue", queue)

	if queue == contestSubmissionQueue {
		return w.mq.Subscribe(ctx, queue, func(ctx context.Context, msg mq.Message) error {
			var job types.ContestSubmissionJob
			if err := json.Unmarshal(msg.Data, &job); err != nil {
				log.Printf("worker: failed to unmarshal contest job: %v", err)
				return nil // ack bad messages
			}
			log.Printf("worker: processing contest submission %d for problem %d", job.ContestSubmission.ID, job.Problem.ID)
			if err := w.processContestJob(ctx, job); err != nil {
				log.Printf("worker: failed to process contest submission %d: %v", job.ContestSubmission.ID, err)
				return err
			}
			log.Printf("worker: finished contest submission %d", job.ContestSubmission.ID)
			return nil
		})
	}

	return w.mq.Subscribe(ctx, queue, func(ctx context.Context, msg mq.Message) error {
		var job types.SubmissionJob
		if err := json.Unmarshal(msg.Data, &job); err != nil {
			log.Printf("worker: failed to unmarshal job: %v", err)
			return nil // ack bad messages
		}
		log.Printf("worker: processing submission %d for problem %d", job.Submission.ID, job.Problem.ID)
		if err := w.processJob(ctx, job); err != nil {
			log.Printf("worker: failed to process submission %d: %v", job.Submission.ID, err)
			return err
		}
		log.Printf("worker: finished submission %d", job.Submission.ID)
		return nil
	})
}

// publishResult publishes a submission update to the results queue.
func (w *Worker) publishResult(ctx context.Context, submission types.Submission) error {
	data, err := json.Marshal(submission)
	if err != nil {
		return err
	}
	_, err = w.mq.Publish(ctx, resultQueue, data, nil)
	return err
}

// publishContestResult publishes a contest submission update to the contest results queue.
func (w *Worker) publishContestResult(ctx context.Context, cs types.ContestSubmission) error {
	data, err := json.Marshal(cs)
	if err != nil {
		return err
	}
	_, err = w.mq.Publish(ctx, contestResultQueue, data, nil)
	return err
}
