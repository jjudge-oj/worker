package worker

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"

	"github.com/jjudge-oj/api/types"
	"github.com/jjudge-oj/worker/internal/lime"
)

const (
	compilationTimeLimitUs = 30_000_000 // 30 seconds
	compilationMemoryLimit = 512 * 1024 * 1024
	compilationMaxProcs    = 32
	defaultMaxProcs        = 1
)

type publishFunc func(ctx context.Context, submission types.Submission) error

func (w *Worker) processJob(ctx context.Context, job types.SubmissionJob) error {
	return w.processJobWithPublisher(ctx, job, w.publishResult)
}

func (w *Worker) processContestJob(ctx context.Context, job types.ContestSubmissionJob) error {
	cs := job.ContestSubmission
	// Convert to Submission so the shared processing logic can run unchanged.
	syntheticJob := types.SubmissionJob{
		Submission: types.Submission{
			ID:        int(cs.ID),
			ProblemID: cs.ProblemID,
			UserID:    cs.UserID,
			Code:      cs.Code,
			Language:  cs.Language,
			Verdict:   cs.Verdict,
		},
		Problem: job.Problem,
	}
	return w.processJobWithPublisher(ctx, syntheticJob, func(ctx context.Context, result types.Submission) error {
		// Copy judging outcome back onto the original ContestSubmission.
		cs.Verdict = result.Verdict
		cs.Score = result.Score
		cs.CPUTime = result.CPUTime
		cs.Memory = result.Memory
		cs.Message = result.Message
		cs.TestsPassed = result.TestsPassed
		cs.TestsTotal = result.TestsTotal
		cs.TestcaseResults = result.TestcaseResults
		return w.publishContestResult(ctx, cs)
	})
}

func (w *Worker) processJobWithPublisher(ctx context.Context, job types.SubmissionJob, publish publishFunc) error {
	submission := job.Submission
	problem := job.Problem

	// Publish JUDGING status
	submission.Verdict = types.VerdictJudging
	if err := publish(ctx, submission); err != nil {
		log.Printf("worker: failed to publish JUDGING status for submission %d: %v", submission.ID, err)
	}

	// Create work directory
	workDir := filepath.Join(w.cfg.Judge.SubmissionsDir, fmt.Sprintf("%d", submission.ID))
	if err := os.MkdirAll(workDir, 0755); err != nil {
		return w.failWithSystemError(ctx, submission, fmt.Sprintf("failed to create work dir: %v", err), publish)
	}
	defer os.RemoveAll(workDir)

	// Write source code
	sourceFile, execArgs, err := w.writeSource(workDir, submission)
	if err != nil {
		return w.failWithSystemError(ctx, submission, fmt.Sprintf("failed to write source: %v", err), publish)
	}

	// Compile if needed
	if submission.Language == "cpp" {
		compiled, compileErr := w.compile(ctx, workDir, sourceFile, submission, publish)
		if compileErr != nil {
			return w.failWithSystemError(ctx, submission, fmt.Sprintf("compilation system error: %v", compileErr), publish)
		}
		if !compiled {
			return nil // CE already published
		}
		execArgs = []string{"/work/solution"}
	}

	// Sort testcase groups by ordinal
	groups := make([]types.TestcaseGroup, len(problem.TestcaseGroups))
	copy(groups, problem.TestcaseGroups)
	sort.Slice(groups, func(i, j int) bool {
		return groups[i].Ordinal < groups[j].Ordinal
	})

	// Execute per test case
	var (
		results      []types.TestcaseResult
		maxCPUTime   int64
		maxMemory    int64
		testsPassed  int
		testsTotal   int
		score        int
		worstVerdict types.Verdict = types.VerdictAccepted
	)

	for _, group := range groups {
		log.Println("worker: processing testcase group", group.ID)

		// Sort testcases within group by ordinal
		testcases := make([]types.Testcase, len(group.Testcases))
		copy(testcases, group.Testcases)
		sort.Slice(testcases, func(i, j int) bool {
			return testcases[i].Ordinal < testcases[j].Ordinal
		})

		groupAllPassed := true

		for _, tc := range testcases {
			log.Println("worker: processing testcase", tc.Ordinal)

			testsTotal++

			// Fetch test input and expected output
			inPath, err := w.tccache.GetOrFetch(ctx, tc.InKey)
			if err != nil {
				return w.failWithSystemError(ctx, submission, fmt.Sprintf("failed to fetch test input %s: %v", tc.InKey, err), publish)
			}
			outPath, err := w.tccache.GetOrFetch(ctx, tc.OutKey)
			if err != nil {
				return w.failWithSystemError(ctx, submission, fmt.Sprintf("failed to fetch test output %s: %v", tc.OutKey, err), publish)
			}

			inputContent, err := os.ReadFile(inPath)
			if err != nil {
				return w.failWithSystemError(ctx, submission, fmt.Sprintf("failed to read test input: %v", err), publish)
			}
			expectedOutput, err := os.ReadFile(outPath)
			if err != nil {
				return w.failWithSystemError(ctx, submission, fmt.Sprintf("failed to read expected output: %v", err), publish)
			}

			// Execute
			timeLimitUs := uint64(problem.TimeLimit) * 1000 // ms → μs
			memoryLimitBytes := uint64(problem.MemoryLimit)

			report, err := lime.Run(ctx, w.cfg, w.slotPool, workDir, "", execArgs, string(inputContent), timeLimitUs, memoryLimitBytes, defaultMaxProcs)
			if err != nil {
				return w.failWithSystemError(ctx, submission, fmt.Sprintf("execution error: %v", err), publish)
			}

			log.Printf("worker: testcase %d report: status=%d exitCode=%d cpuTime=%d memory=%d", tc.ID, report.Status, report.ExitCode, report.CPUTime, report.Memory)

			// Map report status to verdict
			tcVerdict := w.mapStatusToVerdict(ctx, report, string(expectedOutput))

			// Track results
			cpuTimeMs := int64(report.CPUTime / 1000) // μs → ms
			if cpuTimeMs > maxCPUTime {
				maxCPUTime = cpuTimeMs
			}
			memBytes := int64(report.Memory)
			if memBytes > maxMemory {
				maxMemory = memBytes
			}

			if tcVerdict == types.VerdictAccepted {
				testsPassed++
			} else {
				groupAllPassed = false
				if worstVerdict == types.VerdictAccepted {
					worstVerdict = tcVerdict
				}
			}

			result := types.TestcaseResult{
				SubmissionID: int64(submission.ID),
				TestcaseID:   tc.ID,
				Verdict:      tcVerdict,
				CPUTime:      cpuTimeMs,
				Memory:       memBytes,
			}
			if !tc.IsHidden {
				result.Input = truncate(string(inputContent), 200)
				result.ExpectedOutput = truncate(string(expectedOutput), 200)
				result.ActualOutput = truncate(report.Stdout, 200)
			}
			if tcVerdict == types.VerdictRuntimeError {
				result.ErrorMessage = truncate(report.Stderr, 200)
			}

			results = append(results, result)
		}

		if groupAllPassed {
			score += group.Points
		}
	}

	// Aggregate final verdict
	finalVerdict := types.VerdictAccepted
	if testsPassed < testsTotal {
		finalVerdict = worstVerdict
	}

	submission.Verdict = finalVerdict
	submission.Score = score
	submission.CPUTime = maxCPUTime
	submission.Memory = maxMemory
	submission.TestsPassed = testsPassed
	submission.TestsTotal = testsTotal
	submission.TestcaseResults = results

	return publish(ctx, submission)
}

func (w *Worker) writeSource(workDir string, submission types.Submission) (string, []string, error) {
	var filename string
	var args []string

	switch submission.Language {
	case "cpp":
		filename = "solution.cpp"
		args = []string{"/work/solution"} // will be set after compilation
	case "python":
		filename = "solution.py"
		args = []string{"python3", "/work/solution.py"}
	default:
		return "", nil, fmt.Errorf("unsupported language: %s", submission.Language)
	}

	filePath := filepath.Join(workDir, filename)
	if err := os.WriteFile(filePath, []byte(submission.Code), 0644); err != nil {
		return "", nil, err
	}

	return filename, args, nil
}

func (w *Worker) compile(ctx context.Context, workDir, sourceFile string, submission types.Submission, publish publishFunc) (bool, error) {
	args := []string{"/usr/local/bin/g++", "-std=c++20", "-O2", "-o", "/work/solution", "/work/" + sourceFile}

	report, err := lime.Run(ctx, w.cfg, w.slotPool, workDir, "", args, "", compilationTimeLimitUs, compilationMemoryLimit, compilationMaxProcs)
	if err != nil {
		return false, err
	}

	if report.Status != lime.STATUS_OK || report.ExitCode != 0 {
		submission.Verdict = types.VerdictCompilationError
		submission.Message = report.Stderr
		_ = publish(ctx, submission)
		return false, nil
	}

	return true, nil
}

func (w *Worker) mapStatusToVerdict(ctx context.Context, report *lime.Report, expectedOutput string) types.Verdict {
	switch report.Status {
	case lime.STATUS_TIME_LIMIT_EXCEEDED:
		return types.VerdictTimeLimitExceeded
	case lime.STATUS_MEMORY_LIMIT_EXCEEDED:
		return types.VerdictMemoryLimitExceeded
	case lime.STATUS_RUNTIME_ERROR:
		return types.VerdictRuntimeError
	case lime.STATUS_OK:
		ok, err := w.grader.Grade(ctx, report.Stdout, expectedOutput, "token")
		if err != nil {
			log.Printf("worker: grader error: %v", err)
			return types.VerdictSystemError
		}
		if ok {
			return types.VerdictAccepted
		}
		return types.VerdictWrongAnswer
	default:
		return types.VerdictSystemError
	}
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max]
}

func (w *Worker) failWithSystemError(ctx context.Context, submission types.Submission, message string, publish publishFunc) error {
	log.Printf("worker: system error for submission %d: %s", submission.ID, message)
	submission.Verdict = types.VerdictSystemError
	submission.Message = message
	return publish(ctx, submission)
}
