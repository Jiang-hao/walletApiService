package unit

import (
	"context"
	"errors"
	"testing"

	"github.com/Jiang-hao/walletApiService/internal/model"
	"github.com/Jiang-hao/walletApiService/internal/repository"
	"github.com/Jiang-hao/walletApiService/internal/service"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockWalletRepository implements WalletRepository interface
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

// MockTransactionRepository implements TransactionRepository interface
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

// MockWalletTx implements WalletTx interface
type MockWalletTx struct {
	mock.Mock
	*sqlx.Tx
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

func TestWalletService_Deposit(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	amount := decimal.NewFromFloat(100.50)
	currency := "USD"
	reference := "test deposit"

	tests := []struct {
		name        string
		setup       func(*MockWalletRepository, *MockTransactionRepository, *MockWalletTx)
		amount      decimal.Decimal
		expectError bool
		errorMsg    string
	}{
		{
			name: "successful deposit to existing wallet",
			setup: func(wr *MockWalletRepository, tr *MockTransactionRepository, wt *MockWalletTx) {
				wallet := &model.Wallet{
					ID:       uuid.New(),
					UserID:   userID,
					Currency: currency,
					Balance:  decimal.NewFromFloat(50.00),
					Version:  1,
				}
				wr.On("GetWalletByUserAndCurrency", ctx, userID, currency).Return(wallet, nil)
				wr.On("UpdateWalletBalance", ctx, wallet.ID, wallet.Balance.Add(amount), wallet.Version).Return(int64(1), nil)
				tr.On("CreateTransaction", ctx, mock.AnythingOfType("*model.Transaction")).Return(nil)
			},
			amount: amount,
		},
		{
			name: "negative amount should fail",
			setup: func(wr *MockWalletRepository, tr *MockTransactionRepository, wt *MockWalletTx) {
				// No mocks needed as validation happens first
			},
			amount:      decimal.NewFromFloat(-50),
			expectError: true,
		},
		{
			name: "zero amount should fail",
			setup: func(wr *MockWalletRepository, tr *MockTransactionRepository, wt *MockWalletTx) {
				// No mocks needed as validation happens first
			},
			amount:      decimal.Zero,
			expectError: true,
		},
		{
			name: "optimistic lock conflict with retry",
			setup: func(wr *MockWalletRepository, tr *MockTransactionRepository, wt *MockWalletTx) {
				wallet := &model.Wallet{
					ID:       uuid.New(),
					UserID:   userID,
					Currency: currency,
					Balance:  decimal.NewFromFloat(50.00),
					Version:  1,
				}
				wr.On("GetWalletByUserAndCurrency", ctx, userID, currency).Return(wallet, nil)

				// First update fails
				wr.On("UpdateWalletBalance", ctx, wallet.ID, wallet.Balance.Add(amount), wallet.Version).
					Return(int64(0), nil).
					Once()

				// Second update succeeds
				wr.On("UpdateWalletBalance", ctx, wallet.ID, wallet.Balance.Add(amount), wallet.Version).
					Return(int64(1), nil).
					Once()

				tr.On("CreateTransaction", ctx, mock.AnythingOfType("*model.Transaction")).Return(nil)
			},
			amount: amount,
		},
		{
			name: "max retries exceeded",
			setup: func(wr *MockWalletRepository, tr *MockTransactionRepository, wt *MockWalletTx) {
				wallet := &model.Wallet{
					ID:       uuid.New(),
					UserID:   userID,
					Currency: currency,
					Balance:  decimal.NewFromFloat(50.00),
					Version:  1,
				}
				wr.On("GetWalletByUserAndCurrency", ctx, userID, currency).Return(wallet, nil)
				wr.On("UpdateWalletBalance", ctx, wallet.ID, wallet.Balance.Add(amount), wallet.Version).
					Return(int64(0), nil).
					Times(3) // Will retry 3 times
			},
			amount:      amount,
			expectError: true,
			errorMsg:    "optimistic lock conflict",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wr := &MockWalletRepository{}
			tr := &MockTransactionRepository{}
			wt := &MockWalletTx{}

			if tt.setup != nil {
				tt.setup(wr, tr, wt)
			}

			//util := util.NewWalletUtil(wr, tr, tr) // tr implements TxManager
			service := service.NewWalletService(wr, tr, tr)

			_, err := service.Deposit(ctx, userID, tt.amount, currency, reference)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}

			wr.AssertExpectations(t)
			tr.AssertExpectations(t)
		})
	}
}

func TestWalletService_Withdraw(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	amount := decimal.NewFromFloat(50.00)
	currency := "USD"
	reference := "test withdrawal"

	tests := []struct {
		name        string
		setup       func(*MockWalletRepository, *MockTransactionRepository, *MockWalletTx)
		amount      decimal.Decimal
		expectError bool
		errorMsg    string
	}{
		{
			name: "successful withdrawal",
			setup: func(wr *MockWalletRepository, tr *MockTransactionRepository, wt *MockWalletTx) {
				wallet := &model.Wallet{
					ID:       uuid.New(),
					UserID:   userID,
					Currency: currency,
					Balance:  decimal.NewFromFloat(100.00),
					Version:  1,
				}
				wr.On("GetWalletByUserAndCurrency", ctx, userID, currency).Return(wallet, nil)
				wr.On("UpdateWalletBalance", ctx, wallet.ID, wallet.Balance.Sub(amount), wallet.Version).Return(int64(1), nil)
				tr.On("CreateTransaction", ctx, mock.AnythingOfType("*model.Transaction")).Return(nil)
			},
			amount: amount,
		},
		{
			name: "insufficient balance",
			setup: func(wr *MockWalletRepository, tr *MockTransactionRepository, wt *MockWalletTx) {
				wallet := &model.Wallet{
					ID:       uuid.New(),
					UserID:   userID,
					Currency: currency,
					Balance:  decimal.NewFromFloat(30.00),
					Version:  1,
				}
				wr.On("GetWalletByUserAndCurrency", ctx, userID, currency).Return(wallet, nil)
			},
			amount:      amount,
			expectError: true,
			errorMsg:    "insufficient balance",
		},
		{
			name: "negative amount should fail",
			setup: func(wr *MockWalletRepository, tr *MockTransactionRepository, wt *MockWalletTx) {
				// No mocks needed as validation happens first
			},
			amount:      decimal.NewFromFloat(-50),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wr := &MockWalletRepository{}
			tr := &MockTransactionRepository{}
			wt := &MockWalletTx{}

			if tt.setup != nil {
				tt.setup(wr, tr, wt)
			}

			//util := util.NewWalletUtil(wr, tr, tr) // tr implements TxManager
			//service := &service.walletService{utils: util}

			service := service.NewWalletService(wr, tr, tr)

			_, err := service.Withdraw(ctx, userID, tt.amount, currency, reference)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}

			wr.AssertExpectations(t)
			tr.AssertExpectations(t)
		})
	}
}

func TestWalletService_Transfer(t *testing.T) {
	ctx := context.Background()
	fromUserID := uuid.New()
	toUserID := uuid.New()
	amount := decimal.NewFromFloat(50.00)
	currency := "USD"
	reference := "test transfer"

	tests := []struct {
		name        string
		setup       func(*MockWalletRepository, *MockTransactionRepository, *MockWalletTx)
		expectError bool
		errorMsg    string
	}{
		{
			name: "successful transfer",
			setup: func(wr *MockWalletRepository, tr *MockTransactionRepository, wt *MockWalletTx) {
				fromWallet := &model.Wallet{
					ID:       uuid.New(),
					UserID:   fromUserID,
					Currency: currency,
					Balance:  decimal.NewFromFloat(100.00),
				}
				toWallet := &model.Wallet{
					ID:       uuid.New(),
					UserID:   toUserID,
					Currency: currency,
					Balance:  decimal.NewFromFloat(20.00),
				}

				tr.On("BeginTx", ctx).Return(wt, nil)

				wr.On("GetWalletByUserAndCurrency", ctx, fromUserID, currency).
					Return(fromWallet, nil)
				wr.On("GetWalletByUserAndCurrency", ctx, toUserID, currency).
					Return(toWallet, nil)

				wt.On("UpdateWalletBalanceTx", ctx, fromWallet.ID, fromWallet.Balance.Sub(amount)).
					Return(nil)
				wt.On("UpdateWalletBalanceTx", ctx, toWallet.ID, toWallet.Balance.Add(amount)).
					Return(nil)

				wt.On("CreateTransactionTx", ctx, mock.AnythingOfType("*model.Transaction")).
					Return(nil).
					Twice()

				wt.On("Commit").Return(nil)
			},
		},
		{
			name: "insufficient balance",
			setup: func(wr *MockWalletRepository, tr *MockTransactionRepository, wt *MockWalletTx) {
				fromWallet := &model.Wallet{
					ID:       uuid.New(),
					UserID:   fromUserID,
					Currency: currency,
					Balance:  decimal.NewFromFloat(0.00),
				}
				toWallet := &model.Wallet{
					ID:       uuid.New(),
					UserID:   toUserID,
					Currency: currency,
					Balance:  decimal.NewFromFloat(20.00),
				}

				tr.On("BeginTx", ctx).Return(wt, nil)

				wr.On("GetWalletByUserAndCurrency", ctx, fromUserID, currency).
					Return(fromWallet, nil)
				wr.On("GetWalletByUserAndCurrency", ctx, toUserID, currency).
					Return(toWallet, nil)

				wt.On("Rollback").Return(nil)
			},
			expectError: true,
			errorMsg:    "insufficient balance",
		},
		{
			name: "transaction rollback on error",
			setup: func(wr *MockWalletRepository, tr *MockTransactionRepository, wt *MockWalletTx) {
				fromWallet := &model.Wallet{
					ID:       uuid.New(),
					UserID:   fromUserID,
					Currency: currency,
					Balance:  decimal.NewFromFloat(100.00),
				}
				toWallet := &model.Wallet{
					ID:       uuid.New(),
					UserID:   toUserID,
					Currency: currency,
					Balance:  decimal.NewFromFloat(20.00),
				}

				tr.On("BeginTx", ctx).Return(wt, nil)

				wr.On("GetWalletByUserAndCurrency", ctx, fromUserID, currency).Return(fromWallet, nil)
				wr.On("GetWalletByUserAndCurrency", ctx, toUserID, currency).Return(toWallet, nil)

				wt.On("UpdateWalletBalanceTx", ctx, fromWallet.ID, fromWallet.Balance.Sub(amount)).Return(nil)
				wt.On("UpdateWalletBalanceTx", ctx, toWallet.ID, toWallet.Balance.Add(amount)).
					Return(errors.New("database error"))

				wt.On("Rollback").Return(nil)
			},
			expectError: true,
			errorMsg:    "database error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wr := &MockWalletRepository{}
			tr := &MockTransactionRepository{}
			wt := &MockWalletTx{}

			if tt.setup != nil {
				tt.setup(wr, tr, wt)
			}
			
			service := service.NewWalletService(wr, tr, tr)
			_, err := service.Transfer(ctx, fromUserID, toUserID, amount, currency, reference)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}

			wr.AssertExpectations(t)
			tr.AssertExpectations(t)
			wt.AssertExpectations(t)
		})
	}
}

func TestWalletService_GetBalance(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	currency := "USD"

	tests := []struct {
		name        string
		setup       func(*MockWalletRepository)
		expectError bool
	}{
		{
			name: "existing wallet",
			setup: func(wr *MockWalletRepository) {
				wallet := &model.Wallet{
					ID:       uuid.New(),
					UserID:   userID,
					Currency: currency,
					Balance:  decimal.NewFromFloat(100.50),
				}
				wr.On("GetWalletByUserAndCurrency", ctx, userID, currency).Return(wallet, nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wr := &MockWalletRepository{}
			tr := &MockTransactionRepository{}

			if tt.setup != nil {
				tt.setup(wr)
			}

			service := service.NewWalletService(wr, tr, tr)

			balance, err := service.GetBalance(ctx, userID, currency)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.True(t, balance.GreaterThanOrEqual(decimal.Zero))
			}

			wr.AssertExpectations(t)
		})
	}
}

func TestWalletService_GetTransactionHistory(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	currency := "USD"
	page := 1
	pageSize := 10

	tests := []struct {
		name        string
		setup       func(*MockWalletRepository, *MockTransactionRepository)
		currency    string
		expectError bool
	}{
		{
			name: "wallet specific transactions",
			setup: func(wr *MockWalletRepository, tr *MockTransactionRepository) {
				wallet := &model.Wallet{
					ID:       uuid.New(),
					UserID:   userID,
					Currency: currency,
					Balance:  decimal.NewFromFloat(100.00),
				}
				wr.On("GetWalletByUserAndCurrency", ctx, userID, currency).Return(wallet, nil)

				transactions := []model.Transaction{
					{
						ID:            uuid.New(),
						WalletID:      wallet.ID,
						Amount:        decimal.NewFromFloat(50.00),
						BalanceBefore: decimal.NewFromFloat(50.00),
						BalanceAfter:  decimal.NewFromFloat(100.00),
						Type:          "deposit",
					},
				}
				tr.On("GetTransactions", ctx, wallet.ID, 0, pageSize).Return(transactions, nil)
			},
			currency: currency,
		},
		{
			name: "all user transactions",
			setup: func(wr *MockWalletRepository, tr *MockTransactionRepository) {
				transactions := []model.Transaction{
					{
						ID:            uuid.New(),
						UserID:        userID,
						Amount:        decimal.NewFromFloat(50.00),
						BalanceBefore: decimal.NewFromFloat(50.00),
						BalanceAfter:  decimal.NewFromFloat(100.00),
						Type:          "deposit",
						Currency:      "USD",
					},
					{
						ID:            uuid.New(),
						UserID:        userID,
						Amount:        decimal.NewFromFloat(30.00),
						BalanceBefore: decimal.NewFromFloat(100.00),
						BalanceAfter:  decimal.NewFromFloat(70.00),
						Type:          "withdrawal",
						Currency:      "EUR",
					},
				}
				tr.On("GetAllTransactions", ctx, userID, 0, pageSize).Return(transactions, nil)
			},
			currency: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wr := &MockWalletRepository{}
			tr := &MockTransactionRepository{}

			if tt.setup != nil {
				tt.setup(wr, tr)
			}

			service := service.NewWalletService(wr, tr, tr)

			_, err := service.GetTransactionHistory(ctx, userID, tt.currency, page, pageSize)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			wr.AssertExpectations(t)
			tr.AssertExpectations(t)
		})
	}
}
