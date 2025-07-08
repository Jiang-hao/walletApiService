package repository

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/Jiang-hao/walletApiService/internal/model"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/shopspring/decimal"
)

type TransactionRepository interface {
	CreateTransaction(ctx context.Context, transaction *model.Transaction) error
	GetTransactions(ctx context.Context, walletID uuid.UUID, offset, limit int) ([]model.Transaction, error)
	TxTransactionRepository
}

type transactionRepo struct {
	db *sqlx.DB
}

func NewTransactionRepository(db *sqlx.DB) TransactionRepository {
	return &transactionRepo{db: db}
}

func (r *transactionRepo) CreateTransaction(ctx context.Context, transaction *model.Transaction) error {
	query := `INSERT INTO transactions 
		(id, wallet_id, amount, balance_before, balance_after, type, related_tx_id, reference)
		VALUES (:id, :wallet_id, :amount, :balance_before, :balance_after, :type, :related_tx_id, :reference)`

	_, err := r.db.NamedExecContext(ctx, query, transaction)
	return err
}

func (r *transactionRepo) GetTransactions(ctx context.Context, walletID uuid.UUID, offset, limit int) ([]model.Transaction, error) {
	var transactions []model.Transaction
	query := `SELECT * FROM transactions 
	          WHERE wallet_id = $1 
	          ORDER BY created_at DESC 
	          LIMIT $2 OFFSET $3`

	err := r.db.SelectContext(ctx, &transactions, query, walletID, limit, offset)
	if err != nil {
		if err == sql.ErrNoRows {
			return []model.Transaction{}, nil
		}
		return nil, fmt.Errorf("failed to get transactions: %w", err)
	}
	return transactions, nil
}

type TxTransactionRepository interface {
	CreateTransactionTx(ctx context.Context, tx *sqlx.Tx, transaction *model.Transaction) error
}

func (r *transactionRepo) CreateTransactionTx(ctx context.Context, tx *sqlx.Tx, transaction *model.Transaction) error {
	query := `INSERT INTO transactions 
		(id, wallet_id, amount, balance_before, balance_after, type, related_tx_id, reference)
		VALUES (:id, :wallet_id, :amount, :balance_before, :balance_after, :type, :related_tx_id, :reference)`

	_, err := tx.NamedExecContext(ctx, query, transaction)
	return err
}

type TxManager interface {
	BeginTx(ctx context.Context) (WalletTx, error)
}

type WalletTx interface {
	Commit() error
	Rollback() error
	GetWalletForUpdate(ctx context.Context, id uuid.UUID) (*model.Wallet, error)
	UpdateWalletBalanceTx(ctx context.Context, id uuid.UUID, newBalance decimal.Decimal) error
	CreateTransactionTx(ctx context.Context, transaction *model.Transaction) error
}

type walletTx struct {
	*sqlx.Tx
	walletRepo      TxWalletRepository
	transactionRepo TxTransactionRepository
}

func (r *transactionRepo) BeginTx(ctx context.Context) (WalletTx, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}

	return &walletTx{
		Tx:              tx,
		walletRepo:      NewWalletRepository(r.db).(TxWalletRepository),
		transactionRepo: r,
	}, nil
}

func (wt *walletTx) GetWalletForUpdate(ctx context.Context, id uuid.UUID) (*model.Wallet, error) {
	var wallet model.Wallet
	query := `SELECT * FROM wallets WHERE id = $1 FOR UPDATE`
	err := wt.Tx.GetContext(ctx, &wallet, query, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get wallet for update: %w", err)
	}
	return &wallet, nil
}

func (wt *walletTx) UpdateWalletBalanceTx(ctx context.Context, id uuid.UUID, newBalance decimal.Decimal) error {
	return wt.walletRepo.UpdateWalletBalanceTx(ctx, wt.Tx, id, newBalance)
}

func (wt *walletTx) CreateTransactionTx(ctx context.Context, transaction *model.Transaction) error {
	return wt.transactionRepo.CreateTransactionTx(ctx, wt.Tx, transaction)
}

func (wt *walletTx) Commit() error {
	return wt.Tx.Commit()
}

func (wt *walletTx) Rollback() error {
	return wt.Tx.Rollback()
}
