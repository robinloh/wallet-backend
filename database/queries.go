package database

const (
	INSERT_ACCOUNTS_QUERY     = `INSERT INTO accounts (id, balance) VALUES (@id, @balance)`
	GET_ACCOUNT_BALANCE_QUERY = `SELECT id, balance FROM accounts WHERE id = @id`

	DEPOSIT_QUERY                    = `UPDATE accounts SET balance = balance + $2 WHERE id = $1`
	DEPOSIT_INSERT_TRANSACTION_QUERY = `INSERT INTO transactions (id, account_id, amount, sendreceiveflag, sender_id, receiver_id, status) VALUES (@id, @account_id, @amount, @sendreceiveflag, @sender_id, @receiver_id, @status)`

	GET_ACCOUNT_TRANSACTIONS_QUERY = `SELECT id, account_id, amount, sendreceiveflag, sender_id, receiver_id, timestamp::timestamptz, status FROM transactions WHERE account_id = @account_id`
)
