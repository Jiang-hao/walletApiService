package model

import (
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type Wallet struct {
	ID        uuid.UUID       `db:"id"`
	UserID    uuid.UUID       `db:"user_id"`
	Currency  string          `db:"currency"`
	Balance   decimal.Decimal `db:"balance"`
	Version   int             `db:"version"`
	CreatedAt string          `db:"created_at"`
	UpdatedAt string          `db:"updated_at"`
}

type WalletResponse struct {
	ID       uuid.UUID       `json:"id"`
	UserID   uuid.UUID       `json:"user_id"`
	Balance  decimal.Decimal `json:"balance"`
	Currency string          `json:"currency"`
}
