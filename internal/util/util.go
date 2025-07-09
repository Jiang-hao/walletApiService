package util

import (
	"context"
	"time"

	"github.com/Jiang-hao/walletApiService/internal/errors"
	"github.com/Jiang-hao/walletApiService/internal/model"
	"github.com/Jiang-hao/walletApiService/internal/repository"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type WalletUtil struct {
	WalletRepo      repository.WalletRepository
	TransactionRepo repository.TransactionRepository
	TxManager       repository.TxManager
}

func NewWalletUtil(
	walletRepo repository.WalletRepository,
	transactionRepo repository.TransactionRepository,
	txManager repository.TxManager,
) *WalletUtil {
	return &WalletUtil{
		WalletRepo:      walletRepo,
		TransactionRepo: transactionRepo,
		TxManager:       txManager,
	}
}

func (u *WalletUtil) GetOrCreateWallet(ctx context.Context, userID uuid.UUID, currency string) (*model.Wallet, error) {
	const op = "utils.GetOrCreateWallet"

	wallet, err := u.WalletRepo.GetWalletByUserAndCurrency(ctx, userID, currency)
	if err == nil {
		return wallet, nil
	}

	if errors.IsNotFound(err) {
		wallet = &model.Wallet{
			ID:       uuid.New(),
			UserID:   userID,
			Currency: currency,
			Balance:  decimal.Zero,
		}
		if err := u.WalletRepo.CreateWallet(ctx, wallet); err != nil {
			return nil, errors.WrapInternal(op, err)
		}
		return wallet, nil
	}

	return nil, errors.WrapInternal(op, err)
}

func (u *WalletUtil) UpdateBalanceWithRetry(
	ctx context.Context,
	wallet *model.Wallet,
	amount decimal.Decimal,
	reference string,
	txType string,
	maxRetries int,
) (*model.WalletResponse, error) {
	const op = "utils.UpdateBalanceWithRetry"

	var newBalance decimal.Decimal
	newBalance = wallet.Balance.Add(amount)
	if newBalance.IsNegative() {
		return nil, errors.NewInsufficientBalance(op)
	}

	for i := 0; i < maxRetries; i++ {
		rows, err := u.WalletRepo.UpdateWalletBalance(ctx, wallet.ID, newBalance, wallet.Version)
		if err != nil {
			return nil, errors.WrapInternal(op, err)
		}

		if rows == 1 {
			tx := &model.Transaction{
				ID:            uuid.New(),
				UserID:        wallet.UserID,
				Currency:      wallet.Currency,
				WalletID:      wallet.ID,
				Amount:        amount,
				BalanceBefore: wallet.Balance,
				BalanceAfter:  newBalance,
				Type:          txType,
				Reference:     reference,
			}
			if err := u.TransactionRepo.CreateTransaction(ctx, tx); err != nil {
				return nil, errors.WrapInternal(op, err)
			}

			return &model.WalletResponse{
				ID:       wallet.ID,
				UserID:   wallet.UserID,
				Balance:  newBalance,
				Currency: wallet.Currency,
			}, nil
		}
		time.Sleep(100 * time.Millisecond)
	}

	return nil, errors.NewConflict(op, "optimistic lock conflict")
}

func (u *WalletUtil) ValidateTransfer(from, to *model.Wallet, amount decimal.Decimal) error {
	const op = "utils.ValidateTransfer"

	if from.Currency != to.Currency {
		return errors.NewCurrencyMismatch(op)
	}
	if from.Balance.LessThan(amount) {
		return errors.NewInsufficientBalance(op)
	}
	return nil
}

func (u *WalletUtil) CreateTransferTransactions(
	ctx context.Context,
	tx repository.WalletTx,
	from, to *model.Wallet,
	amount decimal.Decimal,
	reference string,
) error {
	const op = "utils.CreateTransferTransactions"

	txID := uuid.New()
	fromTx := &model.Transaction{
		ID:            txID,
		WalletID:      from.ID,
		UserID:        from.UserID,
		Currency:      from.Currency,
		Amount:        amount.Neg(),
		BalanceBefore: from.Balance,
		BalanceAfter:  from.Balance.Sub(amount),
		Type:          "transfer",
		Reference:     reference,
	}
	if err := tx.CreateTransactionTx(ctx, fromTx); err != nil {
		return errors.WrapInternal(op, err)
	}

	toTx := &model.Transaction{
		ID:            uuid.New(),
		WalletID:      to.ID,
		UserID:        to.UserID,
		Currency:      to.Currency,
		Amount:        amount,
		BalanceBefore: to.Balance,
		BalanceAfter:  to.Balance.Add(amount),
		Type:          "transfer",
		RelatedTxID:   &txID,
		Reference:     reference,
	}
	return errors.WrapInternal(op, tx.CreateTransactionTx(ctx, toTx))
}

func (u *WalletUtil) GetTransactions(
	ctx context.Context,
	walletID uuid.UUID,
	offset, limit int,
) ([]model.Transaction, error) {
	const op = "utils.GetTransactions"

	transactions, err := u.TransactionRepo.GetTransactions(ctx, walletID, offset, limit)
	return transactions, errors.WrapInternal(op, err)
}

func (u *WalletUtil) GetAllTransactions(
	ctx context.Context,
	userID uuid.UUID,
	offset, limit int,
) ([]model.Transaction, error) {
	const op = "utils.GetTransactions"

	transactions, err := u.TransactionRepo.GetAllTransactions(ctx, userID, offset, limit)
	return transactions, errors.WrapInternal(op, err)
}
