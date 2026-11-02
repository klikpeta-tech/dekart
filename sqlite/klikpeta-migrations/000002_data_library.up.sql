-- Data Library grants and map bindings (fork-only track, see docs/fork-branches.md).

-- Grants: append-only, the latest row per (workspace_id, item_id) wins.
-- seq orders rows written in the same second, which created_at cannot (ENG-2146).
CREATE TABLE library_grant_log (
  seq            INTEGER PRIMARY KEY AUTOINCREMENT,
  workspace_id   TEXT NOT NULL REFERENCES workspaces(id),
  item_id        TEXT NOT NULL,                 -- catalogue item id in the library database
  status         INTEGER NOT NULL,              -- 1 granted, 2 revoked
  ends_on        TEXT NOT NULL,                 -- 'YYYY-MM-DD'; access lasts until 23:59:59 Asia/Jakarta of that day
  category_ids   TEXT NOT NULL DEFAULT '[]',    -- JSON array of granted category ids; empty for items without a taxonomy
  authored_by    TEXT NOT NULL,
  created_at     DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_library_grant_log_ws_item ON library_grant_log (workspace_id, item_id, seq DESC);

-- One binding per library dataset. Kept after the dataset is removed, so restoring a map version re-attaches it.
CREATE TABLE library_dataset_bindings (
  dataset_id     TEXT PRIMARY KEY,              -- datasets.id
  report_id      TEXT NOT NULL,
  item_id        TEXT NOT NULL,
  area_code      TEXT NOT NULL DEFAULT '',      -- '' is the whole coverage, else a 2, 4 or 7 digit BPS code
  category_ids   TEXT NOT NULL DEFAULT '[]',
  brand_ids      TEXT NOT NULL DEFAULT '[]',
  version        INTEGER NOT NULL DEFAULT 1,    -- compare-and-set token; a counter, because timestamps collide within a second
  created_by     TEXT NOT NULL,
  created_at     DATETIME DEFAULT CURRENT_TIMESTAMP,
  updated_at     DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_library_dataset_bindings_report ON library_dataset_bindings (report_id);
