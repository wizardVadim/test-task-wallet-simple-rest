CREATE TABLE exchange_rates (
    id INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    code VARCHAR(30) NOT NULL UNIQUE,
    units_per_usd REAL NOT NULL
    CONSTRAINT exchange_rates_units_per_usd_check
        CHECK (
            units_per_usd > 0
            AND units_per_usd < 'Infinity'::real
        ),
    CONSTRAINT exchange_rates_usd_rate_check
        CHECK (code <> 'USD' OR units_per_usd = 1)
);
