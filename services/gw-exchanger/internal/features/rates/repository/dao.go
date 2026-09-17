package repository

type ExchangeRateDAO struct {
	Code        string  `db:"code"`
	UnitsPerUSD float32 `db:"units_per_usd"`
}
