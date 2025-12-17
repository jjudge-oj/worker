CREATE TABLE IF NOT EXISTS testcase_groups (
    id BIGSERIAL PRIMARY KEY,
    problem_id BIGINT NOT NULL,
    order_id INTEGER NOT NULL,
    name VARCHAR(255) NOT NULL,
    points INTEGER NOT NULL DEFAULT 0,
    FOREIGN KEY (problem_id) REFERENCES problems(id) ON DELETE CASCADE,
    CONSTRAINT uq_testcase_groups_order UNIQUE (problem_id, order_id)
);

CREATE INDEX idx_testcase_groups_problem_id ON testcase_groups(problem_id);

CREATE TABLE IF NOT EXISTS testcases (
    id BIGSERIAL PRIMARY KEY,
    testcase_group_id BIGINT NOT NULL,
    order_id INTEGER NOT NULL,
    input TEXT NOT NULL,
    output TEXT NOT NULL,
    is_hidden BOOLEAN NOT NULL DEFAULT false,
    FOREIGN KEY (testcase_group_id) REFERENCES testcase_groups(id) ON DELETE CASCADE,
    CONSTRAINT uq_testcases_order UNIQUE (testcase_group_id, order_id)
);

CREATE INDEX idx_testcases_testcase_group_id ON testcases(testcase_group_id);
CREATE INDEX idx_testcases_is_hidden ON testcases(is_hidden);
