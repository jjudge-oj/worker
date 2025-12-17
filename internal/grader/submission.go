package grader

import (
	"context"
	"fmt"

	"github.com/joshjms/castletown/internal/models"
)

func (g *Grader) markSubmissionAsGrading(ctx context.Context, sub *models.Submission) error {
	sub.Verdict = models.VerdictJudging
	sub.Score = 0
	sub.Message = "Grading in progress"

	return g.submissionsRepository.UpdateSubmissionResult(ctx, int(sub.ID), sub)
}

func (g *Grader) markSubmissionWithVerdict(ctx context.Context, sub *models.Submission, verdict models.Verdict, score int, message string) error {
	sub.Verdict = verdict
	sub.Score = score
	sub.Message = message

	return g.submissionsRepository.UpdateSubmissionResult(ctx, int(sub.ID), sub)
}

func (g *Grader) persistSubmission(ctx context.Context, sub *models.Submission) error {
	if sub.Message == "Grading in progress" {
		sub.Message = ""
	}

	sub.TestsTotal = len(sub.TestcaseResults)
	sub.TestsPassed = 0

	for _, result := range sub.TestcaseResults {
		if result.Verdict == models.VerdictAccepted {
			sub.TestsPassed++
		}
	}

	if err := g.submissionsRepository.UpdateSubmissionResult(ctx, int(sub.ID), sub); err != nil {
		return fmt.Errorf("update submission result: %w", err)
	}

	if err := g.submissionsRepository.InsertTestcaseResults(ctx, sub.ID, sub.TestcaseResults); err != nil {
		return fmt.Errorf("insert testcase results: %w", err)
	}

	return nil
}
