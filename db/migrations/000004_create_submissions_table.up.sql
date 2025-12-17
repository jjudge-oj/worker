CREATE TABLE IF NOT EXISTS submissions (
    id BIGSERIAL PRIMARY KEY,
    problem_id BIGINT NOT NULL,
    user_id BIGINT NOT NULL,
    code TEXT NOT NULL,
    language VARCHAR(50) NOT NULL,
    verdict VARCHAR(50) NOT NULL CHECK (verdict IN ('PENDING', 'JUDGING', 'AC', 'WA', 'TLE', 'MLE', 'RE', 'CE', 'SE', 'IE', 'SKIPPED')),
    score INTEGER NOT NULL DEFAULT 0,
    cpu_time BIGINT NOT NULL DEFAULT 0,
    memory BIGINT NOT NULL DEFAULT 0,
    message TEXT,
    tests_passed INTEGER NOT NULL DEFAULT 0,
    tests_total INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (problem_id) REFERENCES problems(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX idx_submissions_problem_id ON submissions(problem_id);
CREATE INDEX idx_submissions_user_id ON submissions(user_id);
CREATE INDEX idx_submissions_verdict ON submissions(verdict);
CREATE INDEX idx_submissions_created_at ON submissions(created_at DESC);

CREATE TABLE IF NOT EXISTS testcase_results (
    id BIGSERIAL PRIMARY KEY,
    submission_id BIGINT NOT NULL REFERENCES submissions(id) ON DELETE CASCADE,
    testcase_id BIGINT NOT NULL,
    verdict VARCHAR(50) NOT NULL CHECK (verdict IN ('PENDING', 'JUDGING', 'AC', 'WA', 'TLE', 'MLE', 'RE', 'CE', 'SE', 'IE', 'SKIPPED')),
    cpu_time BIGINT NOT NULL DEFAULT 0,
    memory BIGINT NOT NULL DEFAULT 0,
    input TEXT,
    expected_output TEXT,
    actual_output TEXT,
    error_message TEXT
);

CREATE INDEX idx_testcase_results_submission_id ON testcase_results(submission_id);
