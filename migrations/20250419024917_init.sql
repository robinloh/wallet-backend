-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS accounts
(
    id        SERIAL PRIMARY KEY,
    accountID VARCHAR(36),
    balance   NUMERIC(2)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DELETE TABLE accounts;
-- +goose StatementEnd
