-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS accounts
(
    id      VARCHAR(36) PRIMARY KEY,
    balance NUMERIC(2)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE accounts;
-- +goose StatementEnd
