CREATE TABLE wallets (
    id UUID PRIMARY KEY,
    balance BIGINT NOT NULL DEFAULT 0,
    CONSTRAINT wallets_balance_non_negative CHECK (balance >= 0)
);