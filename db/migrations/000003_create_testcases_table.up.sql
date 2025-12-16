CREATE TABLE IF NOT EXISTS testcase_groups (
    id BIGSERIAL PRIMARY KEY,
    problem_id BIGINT NOT NULL,
    name VARCHAR(255) NOT NULL,
    points INTEGER NOT NULL DEFAULT 0,
    FOREIGN KEY (problem_id) REFERENCES problems(id) ON DELETE CASCADE
);

CREATE INDEX idx_testcase_groups_problem_id ON testcase_groups(problem_id);

CREATE TABLE IF NOT EXISTS testcases (
    id BIGSERIAL PRIMARY KEY,
    problem_id BIGINT NOT NULL,
    input TEXT NOT NULL,
    output TEXT NOT NULL,
    is_hidden BOOLEAN NOT NULL DEFAULT false,
    FOREIGN KEY (problem_id) REFERENCES problems(id) ON DELETE CASCADE
);

CREATE INDEX idx_testcases_problem_id ON testcases(problem_id);
CREATE INDEX idx_testcases_is_hidden ON testcases(is_hidden);
