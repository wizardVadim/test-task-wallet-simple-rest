package repository

import (
	"context"
	"exchanger-app/internal/core/domain"
	"fmt"

	"github.com/jackc/pgx/v5"
)

type DBTX interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

type PostgresRepository struct {
	conn DBTX
}

func NewPostgresRepository(conn DBTX) *PostgresRepository {
	return &PostgresRepository{
		conn: conn,
	}
}

func (repository *PostgresRepository) GetAll(ctx context.Context) ([]domain.ExchangeRate, error) {
	if err := ctx.Err(); err != nil {
		return []domain.ExchangeRate{}, err
	}

	queryRow := `SELECT code, units_per_usd FROM exchange_rates;`

	exchangeRates := make([]domain.ExchangeRate, 0)

	rows, err := repository.conn.Query(ctx, queryRow)
	if err != nil {
		return []domain.ExchangeRate{}, fmt.Errorf("get currencies: %w", err)
	}

	defer rows.Close()

	for rows.Next() {
		dao := ExchangeRateDAO{}

		if err := rows.Scan(
			&dao.Code,
			&dao.UnitsPerUSD,
		); err != nil {
			return []domain.ExchangeRate{}, err
		}

		fromCurrency, err := domain.NewCurrency(domain.CurrencyUSD)
		if err != nil {
			return []domain.ExchangeRate{}, err
		}

		toCurrency, err := domain.NewCurrency(domain.CurrencyType(dao.Code))
		if err != nil {
			return []domain.ExchangeRate{}, err
		}

		rate, err := domain.NewRate(dao.UnitsPerUSD)
		if err != nil {
			return []domain.ExchangeRate{}, err
		}

		exchangeRate, err := domain.NewExchangeRate(rate, fromCurrency, toCurrency)
		if err != nil {
			return []domain.ExchangeRate{}, err
		}

		exchangeRates = append(exchangeRates, exchangeRate)
	}

	if err := rows.Err(); err != nil {
		return []domain.ExchangeRate{}, err
	}

	return exchangeRates, nil
}
