-- +goose Up
CREATE TABLE IF NOT EXISTS downloads (
    gid              TEXT PRIMARY KEY,
    status           TEXT    NULL,
    dir              TEXT    NULL,
    category         TEXT    NULL,
    total_length     INTEGER NULL,
    completed_length INTEGER NULL,
    download_speed   INTEGER NULL,
    connections      INTEGER NULL,
    num_pieces       INTEGER NULL,
    piece_length     INTEGER NULL,
    verified_length  INTEGER NULL,
    error_code       INTEGER NULL,
    error_message    TEXT    NULL,
    resolved_url     TEXT    NULL,
    created_at       DATETIME NOT NULL,
    updated_at       DATETIME NOT NULL
);

CREATE TABLE IF NOT EXISTS download_files (
    id               INTEGER PRIMARY KEY AUTOINCREMENT,
    gid              TEXT    NOT NULL REFERENCES downloads (gid) ON DELETE CASCADE,
    "index"          INTEGER NOT NULL,
    path             TEXT    NULL,
    length           INTEGER NULL,
    completed_length INTEGER NULL,
    selected         BOOLEAN NULL,
    UNIQUE (gid, "index")
);

CREATE TABLE IF NOT EXISTS file_uris (
    id      INTEGER PRIMARY KEY AUTOINCREMENT,
    file_id INTEGER NOT NULL REFERENCES download_files (id) ON DELETE CASCADE,
    uri     TEXT    NULL,
    status  TEXT    NULL
);

CREATE INDEX IF NOT EXISTS ix_file_uris_file_id ON file_uris (file_id);

-- WITHOUT ROWID matches the Python table options; (gid, ts) is already unique.
CREATE TABLE IF NOT EXISTS speed_samples (
    gid   TEXT    NOT NULL REFERENCES downloads (gid) ON DELETE CASCADE,
    ts    INTEGER NOT NULL,
    speed INTEGER NOT NULL,
    PRIMARY KEY (gid, ts)
) WITHOUT ROWID;

CREATE INDEX IF NOT EXISTS ix_speed_samples_ts ON speed_samples (ts);

-- +goose Down
DROP INDEX IF EXISTS ix_speed_samples_ts;
DROP TABLE IF EXISTS speed_samples;
DROP INDEX IF EXISTS ix_file_uris_file_id;
DROP TABLE IF EXISTS file_uris;
DROP TABLE IF EXISTS download_files;
DROP TABLE IF EXISTS downloads;
