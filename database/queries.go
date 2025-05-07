package database

type TxnType string

const (
	TxnTypeDeposit  TxnType = "deposit"
	TxnTypeWithdraw TxnType = "withdraw"
	TxnTypeSender   TxnType = "sender"
	TxnTypeReceiver TxnType = "receiver"
)

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
			$4, 
			CASE WHEN CAST ($4 AS txntype) = 'receiver' THEN $5 ELSE '' END, 
			CASE WHEN CAST ($4 AS txntype) = 'receiver' THEN $1 ELSE '' END, 
			CASE WHEN (SELECT COUNT(*) FROM accs) = 0 THEN 'failed' ELSE 'completed' END
		)
	), txns_transfer_failed AS (
		UPDATE transactions 
		SET (id, account_id, amount, txntype, sender_id, receiver_id, status) = 
		($3, $5, $2, 'sender', $5, $1, 'failed')
		WHERE $4 = 'receiver' AND (SELECT COUNT(*) FROM accs) = 0
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
			$4, 
			CASE WHEN CAST ($4 AS txntype) = 'sender' THEN $1 ELSE '' END, 
			CASE WHEN CAST ($4 AS txntype) = 'sender' THEN $5 ELSE '' END, 
			CASE WHEN (SELECT COUNT(*) FROM accs) = 0 THEN 'failed' ELSE 'completed' END
		)
	), txns_transfer_failed AS (
		INSERT INTO transactions 
		VALUES $3, $5, $2, 'receiver', $1, $5, 'failed'
		WHERE $4 = 'sender' AND (SELECT COUNT(*) FROM accs) = 0
	)
	SELECT COUNT(*) FROM accs 
	`

	GET_ACCOUNT_TRANSACTIONS_QUERY = `SELECT id, account_id, amount, txntype, sender_id, receiver_id, timestamp::timestamptz, status FROM transactions WHERE account_id = @account_id`
)
