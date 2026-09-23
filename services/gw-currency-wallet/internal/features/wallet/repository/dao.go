package repository

type BalanceDAO struct {
	UserID   string `db:"user_id"`
	Currency string `db:"currency"`
	Amount   int64  `db:"amount"`
}
