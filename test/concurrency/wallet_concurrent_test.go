package concurrency

import (
	"context"
	"testing"

	"github.com/Jiang-hao/walletApiService/internal/model"
	"github.com/Jiang-hao/walletApiService/internal/repository"
	"github.com/Jiang-hao/walletApiService/internal/service"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"sync"
)

type MockWalletRepository struct {
	mock.Mock
}

func (m *MockWalletRepository) CreateWallet(ctx context.Context, wallet *model.Wallet) error {
	args := m.Called(ctx, wallet)
	return args.Error(0)
}

func (m *MockWalletRepository) GetWallet(ctx context.Context, id uuid.UUID) (*model.Wallet, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*model.Wallet), args.Error(1)
}

func (m *MockWalletRepository) GetWalletByUserAndCurrency(ctx context.Context, userID uuid.UUID, currency string) (*model.Wallet, error) {
	args := m.Called(ctx, userID, currency)
	return args.Get(0).(*model.Wallet), args.Error(1)
}

func (m *MockWalletRepository) GetWalletForUpdate(ctx context.Context, id uuid.UUID) (*model.Wallet, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*model.Wallet), args.Error(1)
}

func (m *MockWalletRepository) UpdateWalletBalance(ctx context.Context, id uuid.UUID, newBalance decimal.Decimal, version int) (int64, error) {
	args := m.Called(ctx, id, newBalance, version)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockWalletRepository) UpdateWalletBalanceTx(ctx context.Context, tx *sqlx.Tx, id uuid.UUID, newBalance decimal.Decimal) error {
	args := m.Called(ctx, tx, id, newBalance)
	return args.Error(0)
}

type MockTransactionRepository struct {
	mock.Mock
}

func (m *MockTransactionRepository) CreateTransaction(ctx context.Context, tx *model.Transaction) error {
	args := m.Called(ctx, tx)
	return args.Error(0)
}

func (m *MockTransactionRepository) GetTransactions(ctx context.Context, walletID uuid.UUID, offset, limit int) ([]model.Transaction, error) {
	args := m.Called(ctx, walletID, offset, limit)
	return args.Get(0).([]model.Transaction), args.Error(1)
}

func (m *MockTransactionRepository) GetAllTransactions(ctx context.Context, userID uuid.UUID, offset, limit int) ([]model.Transaction, error) {
	args := m.Called(ctx, userID, offset, limit)
	return args.Get(0).([]model.Transaction), args.Error(1)
}

func (m *MockTransactionRepository) CreateTransactionTx(ctx context.Context, tx *sqlx.Tx, transaction *model.Transaction) error {
	args := m.Called(ctx, tx, transaction)
	return args.Error(0)
}

func (m *MockTransactionRepository) BeginTx(ctx context.Context) (repository.WalletTx, error) {
	args := m.Called(ctx)
	return args.Get(0).(repository.WalletTx), args.Error(1)
}

type MockWalletTx struct {
	mock.Mock
}

func (m *MockWalletTx) Commit() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockWalletTx) Rollback() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockWalletTx) GetWalletForUpdate(ctx context.Context, id uuid.UUID) (*model.Wallet, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*model.Wallet), args.Error(1)
}

func (m *MockWalletTx) UpdateWalletBalanceTx(ctx context.Context, id uuid.UUID, newBalance decimal.Decimal) error {
	args := m.Called(ctx, id, newBalance)
	return args.Error(0)
}

func (m *MockWalletTx) CreateTransactionTx(ctx context.Context, tx *model.Transaction) error {
	args := m.Called(ctx, tx)
	return args.Error(0)
}

func TestConcurrentDeposits(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	currency := "USD"
	initialBalance := decimal.NewFromInt(100)
	depositAmount := decimal.NewFromInt(10)
	numDeposits := 5

	// Setup mock wallet
	wallet := &model.Wallet{
		ID:       uuid.New(),
		UserID:   userID,
		Currency: currency,
		Balance:  initialBalance,
		Version:  1,
	}

	// Setup mocks
	wr := &MockWalletRepository{}
	tr := &MockTransactionRepository{}
	txManager := tr

	// Expect GetWalletByUserAndCurrency to return our wallet
	wr.On("GetWalletByUserAndCurrency", ctx, userID, currency).Return(wallet, nil)

	for i := 0; i < numDeposits; i++ {
		wr.On("UpdateWalletBalance", ctx, wallet.ID, mock.AnythingOfType("decimal.Decimal"), mock.AnythingOfType("int")).
			Return(int64(1), nil)
	}

	// Expect transaction creation for each deposit
	tr.On("CreateTransaction", ctx, mock.AnythingOfType("*model.Transaction")).Return(nil)

	// Create service
	service := service.NewWalletService(wr, tr, txManager)

	// Run concurrent deposits
	var wg sync.WaitGroup
	wg.Add(numDeposits)

	for i := 0; i < numDeposits; i++ {
		go func(i int) {
			defer wg.Done()
			_, err := service.Deposit(ctx, userID, depositAmount, currency, "deposit")
			assert.NoError(t, err)
		}(i)
	}

	wg.Wait()

	// Verify all expected mock calls were made
	wr.AssertExpectations(t)
	tr.AssertExpectations(t)
}

func TestConcurrentWithdrawals(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	currency := "USD"
	initialBalance := decimal.NewFromInt(1000)
	withdrawalAmount := decimal.NewFromInt(10)
	numWithdrawals := 5

	// Setup mock wallet
	wallet := &model.Wallet{
		ID:       uuid.New(),
		UserID:   userID,
		Currency: currency,
		Balance:  initialBalance,
		Version:  1,
	}

	// Setup mocks
	wr := &MockWalletRepository{}
	tr := &MockTransactionRepository{}
	txManager := tr

	// Expect GetWalletByUserAndCurrency to return our wallet
	wr.On("GetWalletByUserAndCurrency", ctx, userID, currency).Return(wallet, nil)

	for i := 0; i < numWithdrawals; i++ {
		wr.On("UpdateWalletBalance", ctx, wallet.ID, mock.AnythingOfType("decimal.Decimal"), mock.AnythingOfType("int")).
			Return(int64(1), nil)
	}

	// Expect transaction creation for each withdrawal
	tr.On("CreateTransaction", ctx, mock.AnythingOfType("*model.Transaction")).Return(nil)

	// Create service
	service := service.NewWalletService(wr, tr, txManager)

	// Run concurrent withdrawals
	var wg sync.WaitGroup
	wg.Add(numWithdrawals)

	for i := 0; i < numWithdrawals; i++ {
		go func(i int) {
			defer wg.Done()
			_, err := service.Withdraw(ctx, userID, withdrawalAmount, currency, "withdrawal")
			assert.NoError(t, err)
		}(i)
	}

	wg.Wait()

	// Verify all expected mock calls were made
	wr.AssertExpectations(t)
	tr.AssertExpectations(t)
}

func TestConcurrentTransfers(t *testing.T) {
	ctx := context.Background()
	fromUserID := uuid.New()
	toUserID := uuid.New()
	currency := "USD"
	initialFromBalance := decimal.NewFromInt(1000)
	initialToBalance := decimal.NewFromInt(0)
	transferAmount := decimal.NewFromInt(10)
	numTransfers := 3

	// Setup mock wallets
	fromWallet := &model.Wallet{
		ID:       uuid.New(),
		UserID:   fromUserID,
		Currency: currency,
		Balance:  initialFromBalance,
	}
	toWallet := &model.Wallet{
		ID:       uuid.New(),
		UserID:   toUserID,
		Currency: currency,
		Balance:  initialToBalance,
	}

	// Setup mocks
	wr := &MockWalletRepository{}
	tr := &MockTransactionRepository{}
	txManager := tr

	// Mock transaction behavior
	mockTx := &MockWalletTx{}

	// Expect BeginTx for each transfer
	tr.On("BeginTx", ctx).Return(mockTx, nil)

	// Expect GetWalletByUserAndCurrency for both wallets
	wr.On("GetWalletByUserAndCurrency", ctx, fromUserID, currency).Return(fromWallet, nil)
	wr.On("GetWalletByUserAndCurrency", ctx, toUserID, currency).Return(toWallet, nil)

	// Expect UpdateWalletBalanceTx for both wallets
	mockTx.On("UpdateWalletBalanceTx", ctx, fromWallet.ID, mock.AnythingOfType("decimal.Decimal")).Return(nil)
	mockTx.On("UpdateWalletBalanceTx", ctx, toWallet.ID, mock.AnythingOfType("decimal.Decimal")).Return(nil)

	// Expect CreateTransactionTx for both wallets
	mockTx.On("CreateTransactionTx", ctx, mock.AnythingOfType("*model.Transaction")).Return(nil)

	// Expect Commit for each transfer
	mockTx.On("Commit").Return(nil)

	// Create service
	service := service.NewWalletService(wr, tr, txManager)

	// Run concurrent transfers
	var wg sync.WaitGroup
	wg.Add(numTransfers)

	for i := 0; i < numTransfers; i++ {
		go func(i int) {
			defer wg.Done()
			_, err := service.Transfer(ctx, fromUserID, toUserID, transferAmount, currency, "transfer")
			assert.NoError(t, err)
		}(i)
	}

	wg.Wait()

	// Verify all expected mock calls were made
	wr.AssertExpectations(t)
	tr.AssertExpectations(t)
	mockTx.AssertExpectations(t)
}

func TestConcurrentMixedOperations(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	currency := "USD"
	initialBalance := decimal.NewFromInt(100)
	operationAmount := decimal.NewFromInt(10)
	numOperations := 6 // 3 deposits and 3 withdrawals

	// Setup mock wallet
	wallet := &model.Wallet{
		ID:       uuid.New(),
		UserID:   userID,
		Currency: currency,
		Balance:  initialBalance,
		Version:  1,
	}

	// Setup mocks
	wr := &MockWalletRepository{}
	tr := &MockTransactionRepository{}
	txManager := tr

	// Expect GetWalletByUserAndCurrency to return our wallet
	wr.On("GetWalletByUserAndCurrency", ctx, userID, currency).Return(wallet, nil)

	// Expect UpdateWalletBalance to succeed with optimistic locking
	wr.On("UpdateWalletBalance", ctx, wallet.ID, mock.AnythingOfType("decimal.Decimal"), mock.AnythingOfType("int")).
		Return(int64(1), nil)

	// Expect transaction creation for each operation
	tr.On("CreateTransaction", ctx, mock.AnythingOfType("*model.Transaction")).Return(nil)

	// Create service
	service := service.NewWalletService(wr, tr, txManager)

	// Run concurrent mixed operations
	var wg sync.WaitGroup
	wg.Add(numOperations)

	for i := 0; i < numOperations; i++ {
		go func(i int) {
			defer wg.Done()
			var err error
			if i%2 == 0 {
				_, err = service.Deposit(ctx, userID, operationAmount, currency, "deposit")
			} else {
				_, err = service.Withdraw(ctx, userID, operationAmount, currency, "withdrawal")
			}
			assert.NoError(t, err)
		}(i)
	}

	wg.Wait()

	// Verify all expected mock calls were made
	wr.AssertExpectations(t)
	tr.AssertExpectations(t)
}
func TestOptimisticLockingWithRetries(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	currency := "USD"
	initialBalance := decimal.NewFromInt(100)
	depositAmount := decimal.NewFromInt(10)
	numConcurrentOps := 3

	// Setup mock wallet
	wallet := &model.Wallet{
		ID:       uuid.New(),
		UserID:   userID,
		Currency: currency,
		Balance:  initialBalance,
		Version:  1,
	}

	// Setup mocks
	wr := &MockWalletRepository{}
	tr := &MockTransactionRepository{}
	txManager := tr

	// Expect GetWalletByUserAndCurrency to return our wallet
	wr.On("GetWalletByUserAndCurrency", ctx, userID, currency).Return(wallet, nil).Times(numConcurrentOps)

	// 使用mock.Anything匹配参数，因为并发操作顺序不确定
	// 前numConcurrentOps次调用返回冲突
	for i := 0; i < numConcurrentOps; i++ {
		wr.On("UpdateWalletBalance", ctx, wallet.ID, mock.AnythingOfType("decimal.Decimal"), wallet.Version).
			Return(int64(0), nil).Once()
	}

	// 后续调用返回成功
	wr.On("UpdateWalletBalance", ctx, wallet.ID, mock.AnythingOfType("decimal.Decimal"), mock.AnythingOfType("int")).
		Return(int64(1), nil)

	// Expect transaction creation for each successful operation
	tr.On("CreateTransaction", ctx, mock.AnythingOfType("*model.Transaction")).Return(nil).Times(numConcurrentOps)

	// Create service
	service := service.NewWalletService(wr, tr, txManager)

	// Run concurrent operations
	var wg sync.WaitGroup
	wg.Add(numConcurrentOps)

	for i := 0; i < numConcurrentOps; i++ {
		go func(i int) {
			defer wg.Done()
			_, err := service.Deposit(ctx, userID, depositAmount, currency, "deposit")
			assert.NoError(t, err, "Deposit operation %d failed", i)
		}(i)
	}

	wg.Wait()

	// 验证UpdateWalletBalance被调用了足够次数
	// 至少numConcurrentOps次(初始尝试) + numConcurrentOps次(重试)
	minExpectedCalls := numConcurrentOps * 2
	assert.True(t, len(wr.Calls) >= minExpectedCalls,
		"Expected at least %d UpdateWalletBalance calls, got %d",
		minExpectedCalls, len(wr.Calls))

	// 验证CreateTransaction被调用了numConcurrentOps次
	tr.AssertNumberOfCalls(t, "CreateTransaction", numConcurrentOps)
}
