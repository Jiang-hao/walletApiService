package repository

import (
	"context"
	"database/sql"
	"github.com/shopspring/decimal"

	"github.com/Jiang-hao/walletApiService/internal/errors"
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
	const op = "wallet.Create"

	query := `INSERT INTO wallets (id, user_id, currency, balance) 
              VALUES (:id, :user_id, :currency, :balance)`

	if _, err := r.db.NamedExecContext(ctx, query, wallet); err != nil {
		return errors.NewInsufficientBalance(op)
	}
	return nil
}

func (r *walletRepo) GetWallet(ctx context.Context, id uuid.UUID) (*model.Wallet, error) {
	const op = "wallet.GetByID"
	var wallet model.Wallet

	err := r.db.GetContext(ctx, &wallet, `SELECT * FROM wallets WHERE id = $1`, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.NewNotFound(op, "wallet")
		}
		return nil, errors.NewInternal(op, err)
	}
	return &wallet, nil
}

func (r *walletRepo) GetWalletByUserAndCurrency(ctx context.Context, userID uuid.UUID, currency string) (*model.Wallet, error) {
	const op = "wallet.GetByUserAndCurrency"
	var wallet model.Wallet

	err := r.db.GetContext(ctx, &wallet,
		`SELECT * FROM wallets WHERE user_id = $1 AND currency = $2`,
		userID, currency)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.NewNotFound(op, "wallet")
		}
		return nil, errors.NewInternal(op, err)
	}
	return &wallet, nil
}

func (r *walletRepo) GetWalletForUpdate(ctx context.Context, id uuid.UUID) (*model.Wallet, error) {
	const op = "wallet.GetForUpdate"
	var wallet model.Wallet

	err := r.db.GetContext(ctx, &wallet,
		`SELECT * FROM wallets WHERE id = $1 FOR UPDATE`, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.NewNotFound(op, "wallet")
		}
		return nil, errors.NewInternal(op, err)
	}
	return &wallet, nil
}

func (r *walletRepo) UpdateWalletBalance(ctx context.Context, id uuid.UUID, newBalance decimal.Decimal, version int) (int64, error) {
	const op = "wallet.UpdateBalance"

	result, err := r.db.ExecContext(ctx,
		`UPDATE wallets SET balance = $1, version = version + 1 
         WHERE id = $2 AND version = $3`,
		newBalance, id, version)
	if err != nil {
		return 0, errors.NewInternal(op, err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return 0, errors.NewInternal(op, err)
	}
	return rows, nil
}

type TxWalletRepository interface {
	UpdateWalletBalanceTx(ctx context.Context, tx *sqlx.Tx, id uuid.UUID, newBalance decimal.Decimal) error
}

func (r *walletRepo) UpdateWalletBalanceTx(ctx context.Context, tx *sqlx.Tx, id uuid.UUID, newBalance decimal.Decimal) error {
	const op = "wallet.UpdateBalanceTx"

	if _, err := tx.ExecContext(ctx,
		`UPDATE wallets SET balance = $1 WHERE id = $2`,
		newBalance, id); err != nil {
		return errors.NewInternal(op, err)
	}
	return nil
}
