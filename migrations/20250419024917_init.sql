-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS accounts
(
    id      VARCHAR(36) PRIMARY KEY,
    balance FLOAT8
);

CREATE TYPE txntype AS ENUM ('deposit', 'withdraw', 'sender', 'receiver');

CREATE TABLE IF NOT EXISTS transactions
(
    id     VARCHAR(36),
    account_id      VARCHAR(36),
    amount          FLOAT8,
    txntype         txntype,
    sender_id       VARCHAR(36),
    receiver_id     VARCHAR(36),
    timestamp TIMESTAMP DEFAULT (NOW() AT TIME ZONE 'cct'),
    status TEXT,
    PRIMARY KEY (id, txntype)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE accounts;
-- +goose StatementEnd
