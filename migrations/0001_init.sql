CREATE TABLE IF NOT EXISTS branches (
    id         BIGSERIAL PRIMARY KEY,
    code       TEXT NOT NULL UNIQUE,
    name       TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS form_sessions (
    id               BIGSERIAL PRIMARY KEY,
    form_id          TEXT NOT NULL UNIQUE,
    test_id          BIGINT NOT NULL,
    test_title       TEXT NOT NULL DEFAULT '',
    duration_minutes INT NOT NULL CHECK (duration_minutes > 0),
    form_url         TEXT NOT NULL,
    respondent_url   TEXT NOT NULL,
    edit_url         TEXT NOT NULL DEFAULT '',
    item_map         JSONB NOT NULL DEFAULT '{}'::jsonb,
    question_count   INT NOT NULL DEFAULT 0,
    status           TEXT NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'closed')),
    closed_at        TIMESTAMPTZ,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_form_sessions_test_id ON form_sessions (test_id);
CREATE INDEX IF NOT EXISTS idx_form_sessions_created_at ON form_sessions (created_at DESC);

CREATE TABLE IF NOT EXISTS interns (
    id          BIGSERIAL PRIMARY KEY,
    phone       TEXT NOT NULL UNIQUE,
    last_name   TEXT NOT NULL DEFAULT '',
    first_name  TEXT NOT NULL DEFAULT '',
    middle_name TEXT NOT NULL DEFAULT '',
    branch_id   BIGINT REFERENCES branches (id),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_interns_branch_id ON interns (branch_id);

CREATE TABLE IF NOT EXISTS attempts (
    id              BIGSERIAL PRIMARY KEY,
    response_id     TEXT NOT NULL UNIQUE,
    form_session_id BIGINT NOT NULL REFERENCES form_sessions (id),
    intern_id       BIGINT NOT NULL REFERENCES interns (id),
    test_id         BIGINT NOT NULL,
    branch_id       BIGINT REFERENCES branches (id),
    total_questions INT NOT NULL DEFAULT 0,
    auto_submitted  BOOLEAN NOT NULL DEFAULT FALSE,
    submitted_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_attempts_intern_id ON attempts (intern_id, submitted_at DESC);
CREATE INDEX IF NOT EXISTS idx_attempts_form_session_id ON attempts (form_session_id);
CREATE INDEX IF NOT EXISTS idx_attempts_test_id ON attempts (test_id);

CREATE TABLE IF NOT EXISTS answers (
    id            BIGSERIAL PRIMARY KEY,
    attempt_id    BIGINT NOT NULL REFERENCES attempts (id) ON DELETE CASCADE,
    question_id   BIGINT NOT NULL,
    question_type SMALLINT NOT NULL,
    item_id       TEXT NOT NULL DEFAULT '',
    answer_text   TEXT NOT NULL DEFAULT '',
    score         SMALLINT CHECK (score IN (0, 1)),
    graded_by     TEXT CHECK (graded_by IN ('auto', 'manual')),
    graded_at     TIMESTAMPTZ,
    position      INT NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_answers_attempt_id ON answers (attempt_id, position);
CREATE INDEX IF NOT EXISTS idx_answers_pending ON answers (attempt_id) WHERE score IS NULL;

CREATE TABLE IF NOT EXISTS grading_log (
    id         BIGSERIAL PRIMARY KEY,
    answer_id  BIGINT NOT NULL REFERENCES answers (id) ON DELETE CASCADE,
    attempt_id BIGINT NOT NULL,
    old_score  SMALLINT,
    new_score  SMALLINT,
    source     TEXT NOT NULL CHECK (source IN ('auto', 'manual')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_grading_log_answer_id ON grading_log (answer_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_grading_log_attempt_id ON grading_log (attempt_id, created_at DESC);
