package model

import (
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type Transaction struct {
	ID            uuid.UUID       `db:"id"`
	UserID        uuid.UUID       `db:"user_id"`
	WalletID      uuid.UUID       `db:"wallet_id"`
	Amount        decimal.Decimal `db:"amount"`
	Currency      string          `db:"currency"`
	BalanceBefore decimal.Decimal `db:"balance_before"`
	BalanceAfter  decimal.Decimal `db:"balance_after"`
	Type          string          `db:"type"`
	RelatedTxID   *uuid.UUID      `db:"related_tx_id"`
	Reference     string          `db:"reference"`
	CreatedAt     string          `db:"created_at"`
}

type DepositRequest struct {
	Amount    decimal.Decimal `json:"amount"`
	Currency  string          `json:"currency" binding:"required,len=3"`
	Reference string          `json:"reference"`
}

type WithdrawalRequest struct {
	Amount    decimal.Decimal `json:"amount"`
	Currency  string          `json:"currency" binding:"required,len=3"`
	Reference string          `json:"reference"`
}

type TransferRequest struct {
	ToUserId  uuid.UUID       `json:"to_user_id" binding:"required"`
	Amount    decimal.Decimal `json:"amount"`
	Currency  string          `json:"currency" binding:"required,len=3"`
	Reference string          `json:"reference"`
}

type BalanceResponse struct {
	Balance  decimal.Decimal `json:"balance"`
	Currency string          `json:"currency"`
}

type TransactionResponse struct {
	ID            uuid.UUID       `json:"id"`
	Amount        decimal.Decimal `json:"amount"`
	BalanceBefore decimal.Decimal `json:"balance_before"`
	BalanceAfter  decimal.Decimal `json:"balance_after"`
	Currency      string          `jon:"currency"`
	Type          string          `json:"type"`
	Reference     string          `json:"reference"`
	CreatedAt     string          `json:"created_at"`
}
