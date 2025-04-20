-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS accounts
(
    id      VARCHAR(36) PRIMARY KEY,
    balance FLOAT8
);

CREATE TYPE txntype AS ENUM ('sender', 'receiver');

CREATE TABLE IF NOT EXISTS transactions
(
    id              VARCHAR(36) PRIMARY KEY,
    account_id      VARCHAR(36),
    amount          FLOAT8,
    sendreceiveflag txntype,
    sender_id       VARCHAR(36),
    receiver_id     VARCHAR(36),
    timestamp       TIMESTAMP DEFAULT NOW(),
    status          TEXT
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE accounts;
-- +goose StatementEnd
