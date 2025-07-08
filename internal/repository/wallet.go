package repository

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/shopspring/decimal"

	"github.com/Jiang-hao/walletApiService/internal/model"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type WalletRepository interface {
	CreateWallet(ctx context.Context, wallet *model.Wallet) error
	GetWallet(ctx context.Context, id uuid.UUID) (*model.Wallet, error)
	GetWalletByUserAndCurrency(ctx context.Context, userID uuid.UUID, currency string) (*model.Wallet, error)
	GetWalletForUpdate(ctx context.Context, id uuid.UUID) (*model.Wallet, error)
	UpdateWalletBalance(ctx context.Context, id uuid.UUID, newBalance decimal.Decimal, version int) (int64, error)
	TxWalletRepository
}

type walletRepo struct {
	db *sqlx.DB
}

func NewWalletRepository(db *sqlx.DB) WalletRepository {
	return &walletRepo{db: db}
}

func (r *walletRepo) CreateWallet(ctx context.Context, wallet *model.Wallet) error {
	query := `INSERT INTO wallets (id, user_id, currency, balance) 
	          VALUES (:id, :user_id, :currency, :balance)`
	_, err := r.db.NamedExecContext(ctx, query, wallet)
	return err
}

func (r *walletRepo) GetWallet(ctx context.Context, id uuid.UUID) (*model.Wallet, error) {
	var wallet model.Wallet
	query := `SELECT * FROM wallets WHERE id = $1`
	err := r.db.GetContext(ctx, &wallet, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("wallet not found")
		}
		return nil, fmt.Errorf("failed to get wallet: %w", err)
	}
	return &wallet, nil
}

func (r *walletRepo) GetWalletByUserAndCurrency(ctx context.Context, userID uuid.UUID, currency string) (*model.Wallet, error) {
	var wallet model.Wallet
	query := `SELECT * FROM wallets WHERE user_id = $1 AND currency = $2`
	err := r.db.GetContext(ctx, &wallet, query, userID, currency)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("wallet not found")
		}
		return nil, fmt.Errorf("failed to get wallet: %w", err)
	}
	return &wallet, nil
}

func (r *walletRepo) GetWalletForUpdate(ctx context.Context, id uuid.UUID) (*model.Wallet, error) {
	var wallet model.Wallet
	query := `SELECT * FROM wallets WHERE id = $1 FOR UPDATE`
	err := r.db.GetContext(ctx, &wallet, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("wallet not found")
		}
		return nil, fmt.Errorf("failed to get wallet for update: %w", err)
	}
	return &wallet, nil
}

func (r *walletRepo) UpdateWalletBalance(ctx context.Context, id uuid.UUID, newBalance decimal.Decimal, version int) (int64, error) {
	query := `UPDATE wallets SET balance = $1, version = version + 1, updated_at = NOW() 
	          WHERE id = $2 AND version = $3`
	result, err := r.db.ExecContext(ctx, query, newBalance, id, version)
	if err != nil {
		return 0, fmt.Errorf("failed to update wallet balance: %w", err)
	}
	return result.RowsAffected()
}

type TxWalletRepository interface {
	UpdateWalletBalanceTx(ctx context.Context, tx *sqlx.Tx, id uuid.UUID, newBalance decimal.Decimal) error
}

func (r *walletRepo) UpdateWalletBalanceTx(ctx context.Context, tx *sqlx.Tx, id uuid.UUID, newBalance decimal.Decimal) error {
	query := `UPDATE wallets SET balance = $1, updated_at = NOW() WHERE id = $2`
	_, err := tx.ExecContext(ctx, query, newBalance, id)
	return err
}
