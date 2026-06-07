-- +goose Up
-- +goose StatementBegin
ALTER TABLE messages ADD COLUMN thread_id TEXT NOT NULL DEFAULT '';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE messages DROP COLUMN thread_id;
-- +goose StatementEnd
