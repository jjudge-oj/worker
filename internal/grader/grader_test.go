package grader_test

import (
	"testing"

	"github.com/joshjms/castletown/internal/cache"
	"github.com/joshjms/castletown/internal/config"
	"github.com/joshjms/castletown/internal/container"
	"github.com/joshjms/castletown/internal/grader"
	"github.com/joshjms/castletown/internal/models"
	"github.com/joshjms/castletown/internal/repository"
	"github.com/joshjms/castletown/internal/store"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGrader_HandleSubmissionAccepted(t *testing.T) {
	container.Init()

	cfg := config.Load()

	pr, err := repository.NewProblemsRepository(cfg.Database)
	require.NoError(t, err)

	tcs, err := store.NewTestcaseStore(cfg.Minio)
	require.NoError(t, err)

	store := cache.NewProblemCache(pr, tcs, 20, cfg.Judge.DiskCacheDir)

	log := zerolog.New(nil).With().Timestamp().Logger()

	sr, err := repository.NewSubmissionsRepository(cfg.Database)
	require.NoError(t, err)

	g := grader.NewGrader(log, cfg, sr, store)

	sub := &models.Submission{
		ID:        1,
		ProblemID: 1,
		Language:  "cpp",
		Code: `
#include <bits/stdc++.h>
using namespace std;

int main() {
	long long a, b;
	cin >> a >> b;
	cout << a + b << endl;
	return 0;
}
		`,
	}

	assert.NoError(t, g.Handle(t.Context(), sub))
	assert.Equal(t, models.VerdictAccepted, sub.Verdict)
}

func TestGrader_HandleSubmissionWrongAnswer(t *testing.T) {
	container.Init()

	cfg := config.Load()

	pr, err := repository.NewProblemsRepository(cfg.Database)
	require.NoError(t, err)

	tcs, err := store.NewTestcaseStore(cfg.Minio)
	require.NoError(t, err)

	store := cache.NewProblemCache(pr, tcs, 20, cfg.Judge.DiskCacheDir)

	log := zerolog.New(nil).With().Timestamp().Logger()

	sr, err := repository.NewSubmissionsRepository(cfg.Database)
	require.NoError(t, err)

	g := grader.NewGrader(log, cfg, sr, store)

	sub := &models.Submission{
		ID:        1,
		ProblemID: 1,
		Language:  "cpp",
		Code: `
#include <bits/stdc++.h>
using namespace std;

int main() {
	int a, b;
	cin >> a >> b;
	cout << a + b << endl;
	return 0;
}
		`,
	}

	assert.NoError(t, g.Handle(t.Context(), sub))
	assert.Equal(t, models.VerdictWrongAnswer, sub.Verdict)
}
