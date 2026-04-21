CREATE TABLE IF NOT EXISTS projects (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE,
    budget INTEGER NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME
);

CREATE TABLE IF NOT EXISTS usage_events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    project_id INTEGER NOT NULL REFERENCES projects(id),
    model TEXT NOT NULL,
    tokens_in INTEGER NOT NULL,
    tokens_out INTEGER NOT NULL,
    cost_cents INTEGER NOT NULL,
    latency_ms INTEGER NOT NULL,
    tag TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_usage_project_time 
ON usage_events(project_id, created_at);


CREATE TABLE IF NOT EXISTS models (
  id                      INTEGER PRIMARY KEY AUTOINCREMENT,
  name                    TEXT NOT NULL UNIQUE,
  input_per_million_cents  INTEGER NOT NULL,  -- cost per 1M input tokens in cents
  output_per_million_cents INTEGER NOT NULL,  -- cost per 1M output tokens in cents
  created_at              DATETIME DEFAULT CURRENT_TIMESTAMP
);