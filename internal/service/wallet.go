package service

import (
	"context"

	"github.com/Jiang-hao/walletApiService/internal/errors"
	"github.com/Jiang-hao/walletApiService/internal/model"
	"github.com/Jiang-hao/walletApiService/internal/repository"
	"github.com/Jiang-hao/walletApiService/internal/util"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"log"
	"time"
)

type WalletService interface {
	Deposit(ctx context.Context, userID uuid.UUID, amount decimal.Decimal, currency, reference string) (*model.WalletResponse, error)
	Withdraw(ctx context.Context, userID uuid.UUID, amount decimal.Decimal, currency, reference string) (*model.WalletResponse, error)
	Transfer(ctx context.Context, fromUserID, toUserID uuid.UUID, amount decimal.Decimal, currency, reference string) (*model.WalletResponse, error)
	GetBalance(ctx context.Context, userID uuid.UUID, currency string) (decimal.Decimal, error)
	GetTransactionHistory(ctx context.Context, userID uuid.UUID, currency string, page, pageSize int) ([]model.Transaction, error)
}

type walletService struct {
	utils *util.WalletUtil
}

func NewWalletService(
	walletRepo repository.WalletRepository,
	transactionRepo repository.TransactionRepository,
	txManager repository.TxManager,
) WalletService {
	return &walletService{
		utils: util.NewWalletUtil(walletRepo, transactionRepo, txManager),
	}
}

func (s *walletService) Deposit(ctx context.Context, userID uuid.UUID, amount decimal.Decimal, currency, reference string) (*model.WalletResponse, error) {
	const op = "service.Deposit"
	start := time.Now()
	defer func() {
		log.Printf("[%s] completed in %v", op, time.Since(start))
	}()

	if amount.LessThanOrEqual(decimal.Zero) {
		return nil, errors.NewInvalidInput(op, "amount", amount)
	}

	wallet, err := s.utils.GetOrCreateWallet(ctx, userID, currency)
	if err != nil {
		return nil, errors.WrapInternal(op, err)
	}

	return s.utils.UpdateBalanceWithRetry(ctx, wallet, amount, reference, "deposit", 3)
}

func (s *walletService) Withdraw(ctx context.Context, userID uuid.UUID, amount decimal.Decimal, currency, reference string) (*model.WalletResponse, error) {
	const op = "service.Withdraw"
	start := time.Now()
	defer func() {
		log.Printf("[%s] completed in %v", op, time.Since(start))
	}()

	if amount.LessThanOrEqual(decimal.Zero) {
		return nil, errors.NewInvalidInput(op, "amount", amount)
	}

	wallet, err := s.utils.GetOrCreateWallet(ctx, userID, currency)
	if err != nil {
		return nil, errors.WrapInternal(op, err)
	}

	return s.utils.UpdateBalanceWithRetry(ctx, wallet, amount.Neg(), reference, "withdrawal", 3)
}

func (s *walletService) Transfer(
	ctx context.Context,
	fromUserID, toUserID uuid.UUID,
	amount decimal.Decimal,
	currency, reference string,
) (*model.WalletResponse, error) {
	const op = "service.Transfer"
	start := time.Now()
	defer func() {
		log.Printf("[%s] completed in %v", op, time.Since(start))
	}()

	if amount.LessThanOrEqual(decimal.Zero) {
		return nil, errors.NewInvalidInput(op, "amount", amount)
	}

	tx, err := s.utils.TxManager.BeginTx(ctx)
	if err != nil {
		return nil, errors.WrapInternal(op, err)
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	fromWallet, err := s.utils.GetOrCreateWallet(ctx, fromUserID, currency)
	if err != nil {
		return nil, errors.WrapInternal(op, err)
	}

	toWallet, err := s.utils.GetOrCreateWallet(ctx, toUserID, currency)
	if err != nil {
		return nil, errors.WrapInternal(op, err)
	}

	if err := s.utils.ValidateTransfer(fromWallet, toWallet, amount); err != nil {
		return nil, err
	}

	// update FROM wallet
	newFromBalance := fromWallet.Balance.Sub(amount)
	if err := tx.UpdateWalletBalanceTx(ctx, fromWallet.ID, newFromBalance); err != nil {
		return nil, errors.WrapInternal(op, err)
	}

	// update TO wallet
	newToBalance := toWallet.Balance.Add(amount)
	if err := tx.UpdateWalletBalanceTx(ctx, toWallet.ID, newToBalance); err != nil {
		return nil, errors.WrapInternal(op, err)
	}

	// update FROM - TO transaction (2 directions)
	if err := s.utils.CreateTransferTransactions(ctx, tx, fromWallet, toWallet, amount, reference); err != nil {
		return nil, errors.WrapInternal(op, err)
	}

	if err := tx.Commit(); err != nil {
		return nil, errors.WrapInternal(op, err)
	}

	return &model.WalletResponse{
		ID:       fromWallet.ID,
		UserID:   fromWallet.UserID,
		Balance:  newFromBalance,
		Currency: fromWallet.Currency,
	}, nil
}

func (s *walletService) GetBalance(ctx context.Context, userID uuid.UUID, currency string) (decimal.Decimal, error) {
	const op = "service.GetBalance"

	wallet, err := s.utils.GetOrCreateWallet(ctx, userID, currency)
	if err != nil {
		return decimal.Zero, errors.WrapInternal(op, err)
	}
	return wallet.Balance, nil
}

func (s *walletService) GetTransactionHistory(
	ctx context.Context,
	userID uuid.UUID,
	currency string,
	page, pageSize int,
) ([]model.Transaction, error) {
	const op = "service.GetTransactionHistory"

	if page < 1 {
		return nil, errors.NewInvalidInput(op, "page", page)
	}
	if pageSize < 1 || pageSize > 100 {
		return nil, errors.NewInvalidInput(op, "pageSize", pageSize)
	}
	offset := (page - 1) * pageSize

	if currency == "" {
		return s.utils.GetAllTransactions(ctx, userID, offset, pageSize)
	} else {
		wallet, err := s.utils.GetOrCreateWallet(ctx, userID, currency)
		if err != nil {
			return nil, errors.WrapInternal(op, err)
		}
		return s.utils.GetTransactions(ctx, wallet.ID, offset, pageSize)
	}

}
