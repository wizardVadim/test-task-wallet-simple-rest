DROP TABLE wallets;

CREATE TABLE currencies (
    code VARCHAR(3) PRIMARY KEY
);

INSERT INTO currencies (code)
VALUES ('USD'), ('RUB'), ('EUR');

CREATE TABLE balances (
    user_id UUID NOT NULL,
    currency VARCHAR(3) NOT NULL,
    amount BIGINT NOT NULL DEFAULT 0,

    CONSTRAINT user_currency_pkey
        PRIMARY KEY (user_id, currency),

    CONSTRAINT balances_user_fk
        FOREIGN KEY (user_id)
        REFERENCES users(id)
        ON DELETE CASCADE,

    CONSTRAINT balances_currency_fk
        FOREIGN KEY (currency)
        REFERENCES currencies(code),

    CONSTRAINT balances_amount_check
        CHECK (amount >= 0)
);