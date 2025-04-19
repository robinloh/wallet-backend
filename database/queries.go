package database

const (
	INSERT_ACCOUNTS_QUERY     = `INSERT INTO accounts (id, balance) VALUES (@id, @balance)`
	GET_ACCOUNT_BALANCE_QUERY = `SELECT id, balance FROM accounts WHERE id = @id`
)
