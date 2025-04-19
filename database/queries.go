package database

const (
	INSERT_ACCOUNTS_QUERY = `INSERT INTO accounts (accountid, balance) VALUES (@accountid, @balance)`
)
