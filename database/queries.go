package database

const (
	INSERT_ACCOUNTS_QUERY     = `INSERT INTO accounts (id, balance) VALUES (@id, @balance)`
	GET_ACCOUNT_BALANCE_QUERY = `SELECT id, balance FROM accounts WHERE id = @id`

	DEPOSIT_QUERY = `
	WITH accs AS (
		UPDATE accounts SET balance = balance + $2 WHERE id = $1 AND $2 > 0.00
		RETURNING *
	), txns AS (
		INSERT INTO transactions (id, account_id, amount, txntype, sender_id, receiver_id, status) 
		VALUES (
			$3, 
			$1, 
			$2, 
			'deposit', 
			'', 
			'', 
			CASE WHEN (SELECT COUNT(*) FROM accs) = 0 THEN 'failed' ELSE 'completed' END
		)
	)
	SELECT COUNT(*) FROM accs`

	WITHDRAW_QUERY = `
	WITH accs AS (
		UPDATE accounts SET balance = balance - $2 WHERE id = $1 AND balance >= $2
		RETURNING *
	), txns AS (
		INSERT INTO transactions (id, account_id, amount, txntype, sender_id, receiver_id, status) 
		VALUES (
			$3, 
			$1, 
			$2, 
			'withdraw', 
			'', 
			'', 
			CASE WHEN (SELECT COUNT(*) FROM accs) = 0 THEN 'failed' ELSE 'completed' END
		)
	)
	SELECT COUNT(*) FROM accs 
	`

	GET_ACCOUNT_TRANSACTIONS_QUERY = `SELECT id, account_id, amount, txntype, sender_id, receiver_id, timestamp::timestamptz, status FROM transactions WHERE account_id = @account_id`
)
