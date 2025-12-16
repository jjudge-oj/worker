package models

import "time"

type Verdict string

const (
	VerdictPending             Verdict = "PENDING"
	VerdictJudging             Verdict = "JUDGING"
	VerdictAccepted            Verdict = "AC"
	VerdictWrongAnswer         Verdict = "WA"
	VerdictTimeLimitExceeded   Verdict = "TLE"
	VerdictMemoryLimitExceeded Verdict = "MLE"
	VerdictRuntimeError        Verdict = "RE"
	VerdictCompilationError    Verdict = "CE"
	VerdictSystemError         Verdict = "SE"
	VerdictInternalError       Verdict = "IE"
)

// Submission represents a user's submission to a problem
type Submission struct {
	ID            int64     `json:"id" db:"id"`
	ProblemID     int64     `json:"problem_id" db:"problem_id"`
	UserID        int64     `json:"user_id" db:"user_id"`
	Code          string    `json:"code" db:"code"`
	Language      string    `json:"language" db:"language"`
	Verdict       Verdict   `json:"verdict" db:"verdict"`
	Score         float64   `json:"score" db:"score"`
	ExecutionTime int64     `json:"execution_time" db:"execution_time"`
	MemoryUsed    int64     `json:"memory_used" db:"memory_used"`
	Message       string    `json:"message" db:"message"`
	TestsPassed   int       `json:"tests_passed" db:"tests_passed"`
	TestsTotal    int       `json:"tests_total" db:"tests_total"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time `json:"updated_at" db:"updated_at"`
}

// TestcaseResult represents the result of running a single test case
type TestcaseResult struct {
	TestcaseID     int     `json:"testcase_id"`
	Verdict        Verdict `json:"verdict"`
	ExecutionTime  int64   `json:"execution_time"`
	MemoryUsed     int64   `json:"memory_used"`
	Input          string  `json:"input,omitempty"`
	ExpectedOutput string  `json:"expected_output,omitempty"`
	ActualOutput   string  `json:"actual_output,omitempty"`
	ErrorMessage   string  `json:"error_message,omitempty"`
}

// ExecutionResult represents the result of code execution
type ExecutionResult struct {
	Verdict       Verdict          `json:"verdict"`
	TestResults   []TestcaseResult `json:"test_results"`
	Score         float64          `json:"score"`
	ExecutionTime int64            `json:"execution_time"`
	MemoryUsed    int64            `json:"memory_used"`
	Message       string           `json:"message,omitempty"`
}

// Language represents a supported programming language
type Language struct {
	ID               string  `json:"id"`
	Name             string  `json:"name"`
	Extension        string  `json:"extension"`
	CompileCommand   string  `json:"compile_command"`
	ExecuteCommand   string  `json:"execute_command"`
	Version          string  `json:"version"`
	TimeMultiplier   float64 `json:"time_multiplier"`
	MemoryMultiplier float64 `json:"memory_multiplier"`
}
