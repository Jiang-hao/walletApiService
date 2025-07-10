
CREATE TABLE users (
                       id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
                       username VARCHAR(50) UNIQUE NOT NULL,
                       email VARCHAR(50) UNIQUE NOT NULL,
                       created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE wallets (
                         id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
                         user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
                         currency VARCHAR(3) NOT NULL DEFAULT 'USD',
                         balance DECIMAL(19,4) NOT NULL DEFAULT 0 CHECK (balance >= 0),
                         version INTEGER NOT NULL DEFAULT 0,  -- 乐观锁版本号
                         created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
                         updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
                         UNIQUE(user_id, currency)
);

CREATE TABLE transactions (
                              id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
                              wallet_id UUID NOT NULL REFERENCES wallets(id) ON DELETE CASCADE,
                              user_id UUID NOT NULL,
                              amount DECIMAL(19,4) NOT NULL CHECK (amount != 0),
	currency VARCHAR(3),
    balance_before DECIMAL(19,4) NOT NULL,
    balance_after DECIMAL(19,4) NOT NULL,
    type VARCHAR(20) NOT NULL CHECK (type IN ('deposit', 'withdrawal', 'transfer')),
    related_tx_id UUID,
    reference TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- index
CREATE INDEX idx_transactions_user_wallet ON transactions (user_id, wallet_id);
CREATE INDEX idx_wallets_user ON wallets(user_id);
CREATE INDEX idx_tx_wallet ON transactions(wallet_id);
CREATE INDEX idx_tx_created ON transactions(created_at DESC);




-- Init data

INSERT INTO users (id, username, email) VALUES
                                            ('11111111-1111-1111-1111-111111111111', 'user1', 'user1@example.com'),
                                            ('22222222-2222-2222-2222-222222222222', 'user2', 'user2@example.com');


INSERT INTO wallets (id, user_id, balance, currency) VALUES
                                                         ('33333333-3333-3333-3333-333333333333', '11111111-1111-1111-1111-111111111111', 1000.00, 'USD'),
                                                         ('44444444-4444-4444-4444-444444444444', '22222222-2222-2222-2222-222222222222', 500.00, 'USD');