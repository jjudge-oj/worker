package grader

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/joshjms/castletown/internal/config"
	"github.com/joshjms/castletown/internal/container"
	"github.com/joshjms/castletown/internal/models"
	"github.com/joshjms/castletown/internal/utils"
)

func (g *Grader) handleCppSubmission(ctx context.Context, sub *models.Submission, problem *models.Problem, submissionDir string) error {
	compileDir := filepath.Join(submissionDir, "compile")
	if err := os.Mkdir(compileDir, 0700); err != nil {
		return fmt.Errorf("create compile dir: %w", err)
	}

	r, err := g.compileCpp(ctx, sub, compileDir)
	if err != nil {
		sub.Verdict = models.VerdictCompilationError
		return nil
	}
	if r.Status != container.STATUS_OK {
		sub.Verdict = models.VerdictCompilationError
		return nil
	}

	testcasesDir := filepath.Join(g.runtimeCfg.Judge.DiskCacheDir, fmt.Sprintf("%d", problem.ID))

	groups := append([]models.TestcaseGroup(nil), problem.TestcaseGroups...)
	sort.Slice(groups, func(i, j int) bool { return groups[i].OrderID < groups[j].OrderID })

	testcaseResults := make([]models.TestcaseResult, 0)
	var totalScore int = 0

	for _, grp := range groups {
		tcs := append([]models.Testcase(nil), grp.Testcases...)
		sort.Slice(tcs, func(i, j int) bool { return tcs[i].OrderID < tcs[j].OrderID })

		passGroup := true

		for _, tc := range tcs {
			var tcr models.TestcaseResult
			tcr.TestcaseID = tc.ID
			tcr.SubmissionID = sub.ID

			if !passGroup {
				tcr.Verdict = models.VerdictSkipped
				testcaseResults = append(testcaseResults, tcr)
				continue
			}

			base := fmt.Sprintf("%d_%d", grp.OrderID, tc.OrderID)
			inPath := filepath.Join(testcasesDir, base+".in")
			outPath := filepath.Join(testcasesDir, base+".out")

			if _, err := os.ReadFile(inPath); err != nil {
				return fmt.Errorf("read input %s: %w", inPath, err)
			}
			if _, err := os.ReadFile(outPath); err != nil {
				return fmt.Errorf("read output %s: %w", outPath, err)
			}

			execDir := filepath.Join(submissionDir, base)
			if err := os.Mkdir(execDir, 0700); err != nil {
				return fmt.Errorf("create exec dir: %w", err)
			}
			utils.FileCopy(filepath.Join(compileDir, "main"), filepath.Join(execDir, "main"))

			if _, err := os.Stat(filepath.Join(execDir, "main")); err != nil {
				return fmt.Errorf("stat executable: %w", err)
			}

			report, err := g.executeCpp(
				ctx,
				problem.TimeLimit*1000,
				problem.MemoryLimit*1024*1024,
				inPath,
				execDir,
			)
			if err != nil {
				return err
			}

			tcr.CPUTime = int64(report.CPUTime)
			tcr.Memory = int64(report.Memory)

			switch report.Status {
			case container.STATUS_TIME_LIMIT_EXCEEDED:
				tcr.Verdict = models.VerdictTimeLimitExceeded
			case container.STATUS_MEMORY_LIMIT_EXCEEDED:
				tcr.Verdict = models.VerdictMemoryLimitExceeded
			case container.STATUS_RUNTIME_ERROR:
				tcr.Verdict = models.VerdictRuntimeError
			case container.STATUS_OK:
				// continue to checking
			default:
				return fmt.Errorf("unknown container status: %s", report.Status)
			}

			if tcr.Verdict != "" {
				passGroup = false
				testcaseResults = append(testcaseResults, tcr)
				continue
			}

			expected, err := os.ReadFile(outPath)
			if err != nil {
				return fmt.Errorf("read expected: %w", err)
			}

			match, err := NewChecker(WithTokenComparison()).Check(report.Stdout, string(expected))
			if err != nil {
				return err
			}
			if !match {
				tcr.Verdict = models.VerdictWrongAnswer
				passGroup = false
				testcaseResults = append(testcaseResults, tcr)
				continue
			}

			tcr.Verdict = models.VerdictAccepted
			testcaseResults = append(testcaseResults, tcr)
		}

		if passGroup {
			totalScore += grp.Points
		}
	}

	for _, tcr := range testcaseResults {
		if tcr.Verdict != models.VerdictAccepted && tcr.Verdict != models.VerdictSkipped {
			sub.Verdict = tcr.Verdict
			break
		}
	}

	for _, tcr := range testcaseResults {
		sub.CPUTime = max(sub.CPUTime, tcr.CPUTime)
		sub.Memory = max(sub.Memory, tcr.Memory)
	}

	sub.TestcaseResults = testcaseResults
	sub.Score = totalScore

	if sub.Verdict == "" || sub.Verdict == models.VerdictJudging {
		sub.Verdict = models.VerdictAccepted // YAY!
	}

	return nil
}

func (g *Grader) compileCpp(ctx context.Context, sub *models.Submission, workDir string) (*container.Report, error) {
	sourcePath := filepath.Join(workDir, "main.cpp")
	if err := os.WriteFile(sourcePath, []byte(sub.Code), 0644); err != nil {
		return nil, fmt.Errorf("write source: %w", err)
	}

	imageDir, err := resolveRootfsImageDir(g.runtimeCfg, "gcc-15-bookworm")
	if err != nil {
		return nil, err
	}

	compileArgs := []string{"g++", "-O2", "-std=c++20", "-o", "main", "main.cpp"}
	return container.RunInContainer(
		ctx,
		g.runtimeCfg,
		g.slotPool,
		workDir,
		imageDir,
		compileArgs,
		nil,
		10_000_000,
		512*1024*1024,
		64,
	)
}

func (g *Grader) executeCpp(ctx context.Context, timeLimitUs, memoryLimitBytes int64, inputPath, submissionDir string) (*container.Report, error) {
	inFile, err := os.Open(inputPath)
	if err != nil {
		return nil, fmt.Errorf("open input: %w", err)
	}
	defer inFile.Close()

	imageDir, err := resolveRootfsImageDir(g.runtimeCfg, "gcc-15-bookworm")
	if err != nil {
		return nil, err
	}

	execArgs := []string{"./main"}
	report, err := container.RunInContainer(
		ctx,
		g.runtimeCfg,
		g.slotPool,
		submissionDir,
		imageDir,
		execArgs,
		inFile,
		timeLimitUs,
		memoryLimitBytes,
		1,
	)
	if err != nil {
		return nil, err
	}

	return report, nil
}

func resolveRootfsImageDir(cfg *config.Config, image string) (string, error) {
	defaultImagePath := filepath.Join(cfg.Judge.ImagesDir, image)
	if _, err := os.Stat(defaultImagePath); err == nil {
		return defaultImagePath, nil
	}

	entries, err := os.ReadDir(cfg.Judge.ImagesDir)
	if err != nil {
		return "", fmt.Errorf("failed to read images dir %s: %w", cfg.Judge.ImagesDir, err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			return filepath.Join(cfg.Judge.ImagesDir, entry.Name()), nil
		}
	}

	return "", fmt.Errorf("image not found")
}
