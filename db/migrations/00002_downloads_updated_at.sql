-- +goose Up
-- +goose StatementBegin
CREATE TRIGGER trg_downloads_updated_at
AFTER UPDATE ON downloads
FOR EACH ROW
BEGIN
    UPDATE downloads
    SET updated_at = CURRENT_TIMESTAMP
    WHERE gid = NEW.gid;
END;
-- +goose StatementEnd

-- +goose Down
DROP TRIGGER IF EXISTS trg_downloads_updated_at;
