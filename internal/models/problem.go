package models

import "time"

type Problem struct {
	ID             int             `json:"id" db:"id"`
	Title          string          `json:"title" db:"title"`
	Description    string          `json:"description" db:"description"`
	Difficulty     int             `json:"difficulty" db:"difficulty"`
	TimeLimit      int64           `json:"time_limit" db:"time_limit"`
	MemoryLimit    int64           `json:"memory_limit" db:"memory_limit"`
	TestcaseGroups []TestcaseGroup `json:"testcase_groups" db:"testcase_groups"`
	Tags           []string        `json:"tags" db:"tags"`
	CreatedAt      time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at" db:"updated_at"`
}

type TestcaseGroup struct {
	ID        int        `json:"id" db:"id"`
	OrderID   int        `json:"order_id" db:"order_id"`
	ProblemID int        `json:"problem_id" db:"problem_id"`
	Name      string     `json:"name" db:"name"`
	Testcases []Testcase `json:"testcases" db:"testcases"`
	Points    int        `json:"points" db:"points"`
}

type Testcase struct {
	ID              int    `json:"id" db:"id"`
	OrderID         int    `json:"order_id" db:"order_id"`
	TestcaseGroupID int    `json:"testcase_group_id" db:"testcase_group_id"`
	Input           string `json:"input" db:"input"`
	Output          string `json:"output" db:"output"`
	IsHidden        bool   `json:"is_hidden" db:"is_hidden"`
}
