package repository

import (
	"context"
	"database/sql"
	"github.com/Jiang-hao/walletApiService/internal/errors"
	"github.com/Jiang-hao/walletApiService/internal/model"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/shopspring/decimal"
)

type TransactionRepository interface {
	CreateTransaction(ctx context.Context, tx *model.Transaction) error
	GetTransactions(ctx context.Context, walletID uuid.UUID, offset, limit int) ([]model.Transaction, error)
	GetAllTransactions(ctx context.Context, userID uuid.UUID, offset, limit int) ([]model.Transaction, error)
	TxTransactionRepository
}

type transactionRepo struct {
	db *sqlx.DB
}

func NewTransactionRepository(db *sqlx.DB) TransactionRepository {
	return &transactionRepo{db: db}
}

func (r *transactionRepo) CreateTransaction(ctx context.Context, tx *model.Transaction) error {
	const op = "transaction.Create"

	_, err := r.db.NamedExecContext(ctx, `
        INSERT INTO transactions 
        (id, wallet_id, amount, balance_before, balance_after, type, related_tx_id, reference, currency, user_id)
        VALUES (:id, :wallet_id, :amount, :balance_before, :balance_after, :type, :related_tx_id, :reference, :currency, :user_id)`,
		tx)
	return errors.IfInternalError(op, err)
}

func (r *transactionRepo) GetTransactions(ctx context.Context, walletID uuid.UUID, offset, limit int) ([]model.Transaction, error) {
	const op = "transaction.GetByWallet"
	var txs []model.Transaction

	err := r.db.SelectContext(ctx, &txs, `
        SELECT * FROM transactions 
        WHERE wallet_id = $1 
        ORDER BY created_at DESC 
        LIMIT $2 OFFSET $3`,
		walletID, limit, offset)
	if err != nil && err != sql.ErrNoRows {
		return nil, errors.NewInternal(op, err)
	}
	return txs, nil
}

func (r *transactionRepo) GetAllTransactions(ctx context.Context, userId uuid.UUID, offset, limit int) ([]model.Transaction, error) {
	const op = "transaction.GetByWallet"
	var txs []model.Transaction

	err := r.db.SelectContext(ctx, &txs, `
        SELECT * FROM transactions 
        WHERE user_id = $1 
        ORDER BY created_at DESC 
        LIMIT $2 OFFSET $3`,
		userId, limit, offset)
	if err != nil && err != sql.ErrNoRows {
		return nil, errors.NewInternal(op, err)
	}
	return txs, nil
}

type TxTransactionRepository interface {
	CreateTransactionTx(ctx context.Context, tx *sqlx.Tx, transaction *model.Transaction) error
}

func (r *transactionRepo) CreateTransactionTx(ctx context.Context, dbTx *sqlx.Tx, tx *model.Transaction) error {
	const op = "transaction.CreateTx"

	_, err := dbTx.NamedExecContext(ctx, `
        INSERT INTO transactions 
        (id, wallet_id, amount, balance_before, balance_after, type, related_tx_id, reference, currency, user_id)
        VALUES (:id, :wallet_id, :amount, :balance_before, :balance_after, :type, :related_tx_id, :reference, :currency, :user_id)`,
		tx)
	return errors.IfInternalError(op, err)
}

type TxManager interface {
	BeginTx(ctx context.Context) (WalletTx, error)
}

type WalletTx interface {
	Commit() error
	Rollback() error
	GetWalletForUpdate(ctx context.Context, id uuid.UUID) (*model.Wallet, error)
	UpdateWalletBalanceTx(ctx context.Context, id uuid.UUID, newBalance decimal.Decimal) error
	CreateTransactionTx(ctx context.Context, tx *model.Transaction) error
}

type walletTx struct {
	*sqlx.Tx
	walletRepo      TxWalletRepository
	transactionRepo TxTransactionRepository
}

func (r *transactionRepo) BeginTx(ctx context.Context) (WalletTx, error) {
	const op = "transaction.BeginTx"

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, errors.NewInternal(op, err)
	}

	return &walletTx{
		Tx:              tx,
		walletRepo:      NewWalletRepository(r.db).(TxWalletRepository),
		transactionRepo: r,
	}, nil
}

func (wt *walletTx) GetWalletForUpdate(ctx context.Context, id uuid.UUID) (*model.Wallet, error) {
	const op = "walletTx.GetForUpdate"
	var wallet model.Wallet

	err := wt.Tx.GetContext(ctx, &wallet,
		`SELECT * FROM wallets WHERE id = $1 FOR UPDATE`, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.NewNotFound(op, "wallet")
		}
		return nil, errors.NewInternal(op, err)
	}
	return &wallet, nil
}

func (wt *walletTx) UpdateWalletBalanceTx(ctx context.Context, id uuid.UUID, newBalance decimal.Decimal) error {
	const op = "walletTx.UpdateBalance"

	if err := wt.walletRepo.UpdateWalletBalanceTx(ctx, wt.Tx, id, newBalance); err != nil {
		return errors.NewInternal(op, err)
	}
	return nil
}

func (wt *walletTx) CreateTransactionTx(ctx context.Context, tx *model.Transaction) error {
	const op = "walletTx.CreateTransaction"

	if err := wt.transactionRepo.CreateTransactionTx(ctx, wt.Tx, tx); err != nil {
		return errors.NewInternal(op, err)
	}
	return nil
}

func (wt *walletTx) Commit() error {
	const op = "walletTx.Commit"
	return errors.IfInternalError(op, wt.Tx.Commit())
}

func (wt *walletTx) Rollback() error {
	const op = "walletTx.Rollback"
	return errors.IfInternalError(op, wt.Tx.Rollback())
}
