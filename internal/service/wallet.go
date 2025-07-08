package service

import (
	"context"
	"errors"
	"fmt"
	"github.com/shopspring/decimal"
	"time"

	"github.com/Jiang-hao/walletApiService/internal/model"
	"github.com/Jiang-hao/walletApiService/internal/repository"
	"github.com/google/uuid"
)

type WalletService interface {
	Deposit(ctx context.Context, userID uuid.UUID, amount decimal.Decimal, currency, reference string) (*model.WalletResponse, error)
	Withdraw(ctx context.Context, userID uuid.UUID, amount decimal.Decimal, currency, reference string) (*model.WalletResponse, error)
	Transfer(ctx context.Context, fromUserID, toWalletID uuid.UUID, amount decimal.Decimal, currency, reference string) (*model.WalletResponse, error)
	GetBalance(ctx context.Context, userID uuid.UUID, currency string) (decimal.Decimal, error)
	GetTransactionHistory(ctx context.Context, userID uuid.UUID, currency string, page, pageSize int) ([]model.Transaction, error)
}

type walletService struct {
	walletRepo      repository.WalletRepository
	transactionRepo repository.TransactionRepository
	txManager       repository.TxManager
}

func NewWalletService(
	walletRepo repository.WalletRepository,
	transactionRepo repository.TransactionRepository,
	txManager repository.TxManager,
) WalletService {
	return &walletService{
		walletRepo:      walletRepo,
		transactionRepo: transactionRepo,
		txManager:       txManager,
	}
}

func (s *walletService) getOrCreateWallet(ctx context.Context, userID uuid.UUID, currency string) (*model.Wallet, error) {
	wallet, err := s.walletRepo.GetWalletByUserAndCurrency(ctx, userID, currency)
	if err == nil {
		return wallet, nil
	}

	if err.Error() == "wallet not found" {
		newWallet := &model.Wallet{
			ID:       uuid.New(),
			UserID:   userID,
			Currency: currency,
			Balance:  decimal.Zero,
		}
		err = s.walletRepo.CreateWallet(ctx, newWallet)
		if err != nil {
			return nil, fmt.Errorf("failed to create wallet: %w", err)
		}
		return newWallet, nil
	}

	return nil, fmt.Errorf("failed to get wallet: %w", err)
}

func (s *walletService) Deposit(ctx context.Context, userID uuid.UUID, amount decimal.Decimal, currency, reference string) (*model.WalletResponse, error) {
	if amount.LessThanOrEqual(decimal.Zero) {
		return nil, errors.New("amount must be positive")
	}

	for i := 0; i < 3; i++ {
		wallet, err := s.getOrCreateWallet(ctx, userID, currency)
		if err != nil {
			return nil, fmt.Errorf("get wallet failed: %w", err)
		}

		newBalance := wallet.Balance.Add(amount)
		rowsAffected, err := s.walletRepo.UpdateWalletBalance(
			ctx,
			wallet.ID,
			newBalance,
			wallet.Version,
		)
		if err != nil {
			return nil, fmt.Errorf("update balance failed: %w", err)
		}

		if rowsAffected == 1 {
			tx := model.Transaction{
				ID:            uuid.New(),
				WalletID:      wallet.ID,
				Amount:        amount,
				BalanceBefore: wallet.Balance,
				BalanceAfter:  newBalance,
				Type:          "deposit",
				Reference:     reference,
			}
			if err := s.transactionRepo.CreateTransaction(ctx, &tx); err != nil {
				return nil, fmt.Errorf("create transaction failed: %w", err)
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

	return nil, errors.New("operation conflicted after retries")
}

func (s *walletService) Withdraw(ctx context.Context, userID uuid.UUID, amount decimal.Decimal, currency, reference string) (*model.WalletResponse, error) {
	if amount.LessThanOrEqual(decimal.Zero) {
		return nil, errors.New("amount must be positive")
	}

	for i := 0; i < 3; i++ {
		wallet, err := s.getOrCreateWallet(ctx, userID, currency)
		if err != nil {
			return nil, fmt.Errorf("get wallet failed: %w", err)
		}

		newBalance := wallet.Balance.Sub(amount)
		if newBalance.LessThan(decimal.Zero) {
			return nil, errors.New("insufficient balance")
		}

		rowsAffected, err := s.walletRepo.UpdateWalletBalance(
			ctx,
			wallet.ID,
			newBalance,
			wallet.Version,
		)
		if err != nil {
			return nil, fmt.Errorf("update balance failed: %w", err)
		}

		if rowsAffected == 1 {
			tx := model.Transaction{
				ID:            uuid.New(),
				WalletID:      wallet.ID,
				Amount:        amount.Neg(),
				BalanceBefore: wallet.Balance,
				BalanceAfter:  newBalance,
				Type:          "withdrawal",
				Reference:     reference,
			}
			if err := s.transactionRepo.CreateTransaction(ctx, &tx); err != nil {
				return nil, fmt.Errorf("create transaction failed: %w", err)
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

	return nil, errors.New("operation conflicted after retries")
}

func (s *walletService) Transfer(
	ctx context.Context,
	fromUserID, toWalletID uuid.UUID,
	amount decimal.Decimal,
	currency, reference string,
) (*model.WalletResponse, error) {
	if amount.LessThanOrEqual(decimal.Zero) {
		return nil, errors.New("amount must be positive")
	}

	tx, err := s.txManager.BeginTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin transaction failed: %w", err)
	}
	defer tx.Rollback()

	fromWallet, err := s.getOrCreateWallet(ctx, fromUserID, currency)
	if err != nil {
		return nil, fmt.Errorf("get from wallet failed: %w", err)
	}

	toWallet, err := tx.GetWalletForUpdate(ctx, toWalletID)
	if err != nil {
		return nil, fmt.Errorf("get to wallet failed: %w", err)
	}

	if fromWallet.Currency != toWallet.Currency {
		return nil, errors.New("currency mismatch")
	}

	if fromWallet.Balance.LessThan(amount) {
		return nil, errors.New("insufficient balance")
	}

	newFromBalance := fromWallet.Balance.Sub(amount)
	if err := tx.UpdateWalletBalanceTx(ctx, fromWallet.ID, newFromBalance); err != nil {
		return nil, fmt.Errorf("update from wallet failed: %w", err)
	}

	newToBalance := toWallet.Balance.Add(amount)
	if err := tx.UpdateWalletBalanceTx(ctx, toWallet.ID, newToBalance); err != nil {
		return nil, fmt.Errorf("update to wallet failed: %w", err)
	}

	txID := uuid.New()
	fromTx := model.Transaction{
		ID:            txID,
		WalletID:      fromWallet.ID,
		Amount:        amount.Neg(),
		BalanceBefore: fromWallet.Balance,
		BalanceAfter:  newFromBalance,
		Type:          "transfer",
		Reference:     reference,
	}
	if err := tx.CreateTransactionTx(ctx, &fromTx); err != nil {
		return nil, fmt.Errorf("create from transaction failed: %w", err)
	}

	toTx := model.Transaction{
		ID:            uuid.New(),
		WalletID:      toWallet.ID,
		Amount:        amount,
		BalanceBefore: toWallet.Balance,
		BalanceAfter:  newToBalance,
		Type:          "transfer",
		RelatedTxID:   &txID,
		Reference:     reference,
	}
	if err := tx.CreateTransactionTx(ctx, &toTx); err != nil {
		return nil, fmt.Errorf("create to transaction failed: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit transaction failed: %w", err)
	}

	return &model.WalletResponse{
		ID:       fromWallet.ID,
		UserID:   fromWallet.UserID,
		Balance:  newFromBalance,
		Currency: fromWallet.Currency,
	}, nil
}

func (s *walletService) GetBalance(ctx context.Context, userID uuid.UUID, currency string) (decimal.Decimal, error) {
	wallet, err := s.getOrCreateWallet(ctx, userID, currency)
	if err != nil {
		return decimal.Zero, fmt.Errorf("get wallet failed: %w", err)
	}
	return wallet.Balance, nil
}

func (s *walletService) GetTransactionHistory(
	ctx context.Context,
	userID uuid.UUID,
	currency string,
	page, pageSize int,
) ([]model.Transaction, error) {
	wallet, err := s.getOrCreateWallet(ctx, userID, currency)
	if err != nil {
		return nil, fmt.Errorf("get wallet failed: %w", err)
	}

	offset := (page - 1) * pageSize
	transactions, err := s.transactionRepo.GetTransactions(ctx, wallet.ID, offset, pageSize)
	if err != nil {
		return nil, fmt.Errorf("get transactions failed: %w", err)
	}
	return transactions, nil
}
