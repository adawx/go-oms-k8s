-- +goose Up
CREATE TABLE orders (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    side       TEXT NOT NULL CHECK (side IN ('BUY', 'SELL')),
    type       TEXT NOT NULL CHECK (type IN ('LIMIT', 'MARKET')),
    price      NUMERIC(20, 8) NOT NULL,
    quantity   NUMERIC(20, 8) NOT NULL,
    remaining  NUMERIC(20, 8) NOT NULL,
    timestamp  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE orders;
