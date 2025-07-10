# Wallet API Service - README

## Table of Contents
1. [Overview](#overview)
2. [Features](#features)
3. [Technical Decisions](#technical-decisions)
4. [Setup Instructions](#setup-instructions)
5. [API Documentation](#api-documentation)
6. [Testing](#testing)
7. [Code Review Guide](#code-review-guide)
8. [Areas for Improvement](#areas-for-improvement)
9. [Time Spent](#time-spent)
10. [Omitted Features](#omitted-features)

## Overview

This is a Go-based backend service for a wallet application that provides basic wallet operations including deposits, withdrawals, transfers between users, balance checks, and transaction history.

## Features

- User wallet management
- Money deposit/withdrawal
- Inter-user transfers
- Balance inquiry
- Transaction history
- Optimistic concurrency control
- Transactional operations

## Technical Decisions

### Architecture
- **Layered Architecture**: Clear separation between API, service, repository, and model layers
- **Domain Models**: Wallet and Transaction as core domain entities
- **Error Handling**: Custom error types with clear classification

### Concurrency Control
- **Optimistic Locking**: Implemented via versioning to handle concurrent updates
- **Retry Mechanism**: Automatic retries for failed updates due to conflicts
- **No Database Locks**: Avoids blocking reads while maintaining consistency

### Data Management
- **Decimal Precision**: Uses `shopspring/decimal` for accurate monetary calculations
- **Transaction Records**: Full audit trail of all wallet operations
- **Idempotency**: Reference IDs prevent duplicate operations

## Setup Instructions

### Prerequisites
- Go 1.20+
- PostgreSQL 14+
- (Optional) Redis for caching (not implemented in current version)

### Installation
1. Clone the repository:
   ```bash
   git clone https://github.com/yourusername/wallet-api-service.git
   cd wallet-api-service
   ```

2. Set up PostgreSQL database:
   ```sql
   CREATE DATABASE walletapi;
   ```

3. Run migrations:
   ```bash
   psql -U postgres -d wallet_service -f migrations/000001_init_schema.up.sql
   ```

4. Configure environment variables:
   ```bash
   export DB_HOST=localhost
   export DB_PORT=5432
   export DB_USER= {{your_user}}
   export DB_PASSWORD= {{your_pwd}}
   export DB_NAME=walletapi
   ```

5. Run the service:
   ```bash
   go run cmd/server/main.go
   ```

## API Documentation

### Endpoints

#### 1. Deposit Money
```
POST /wallet/deposit?user_id=<uuid>
```
Request Body:
```json
{
  "amount": "100.50",
  "currency": "USD",
  "reference": "deposit-ref-123"
}
```

#### 2. Withdraw Money
```
POST /wallet/withdraw?user_id=<uuid>
```
Request Body:
```json
{
  "amount": "50.00",
  "currency": "USD",
  "reference": "withdrawal-ref-456"
}
```

#### 3. Transfer Money
```
POST /wallet/transfer?user_id=<sender_uuid>
```
Request Body:
```json
{
  "to_user_id": "<recipient_uuid>",
  "amount": "25.00",
  "currency": "USD",
  "reference": "transfer-ref-789"
}
```

#### 4. Get Balance
```
GET /wallet/balance?user_id=<uuid>&currency=USD
```

#### 5. Get Transaction History
```
GET /wallet/transactions?user_id=<uuid>&currency=USD&page=1&page_size=10
```

### Assumptions
1. Currency codes are 3-letter ISO codes
2. All amounts are positive and in the smallest currency unit (e.g., cents)
3. User IDs are valid UUIDs
4. References are optional but recommended for tracking
5. System is eventually consistent for balance updates

## Testing

### Unit Tests

#### Test Cases

1. **WalletService_Deposit**
    - Successful deposit to existing wallet
    - Negative amount should fail
    - Zero amount should fail
    - Optimistic lock conflict with retry
    - Max retries exceeded

2. **WalletService_Withdraw**
    - Successful withdrawal
    - Insufficient balance
    - Negative amount should fail

3. **WalletService_Transfer**
    - Successful transfer
    - Insufficient balance
    - Transaction rollback on error

4. **WalletService_GetBalance**
    - Existing wallet
    - Non-existent wallet (creates new)

5. **WalletService_GetTransactionHistory**
    - Wallet-specific transactions
    - All user transactions

### Concurrent Tests

1. **ConcurrentDeposits**
    - Multiple simultaneous deposits
    - Verifies balance consistency

2. **ConcurrentWithdrawals**
    - Multiple simultaneous withdrawals
    - Verifies balance consistency

3. **ConcurrentTransfers**
    - Multiple simultaneous transfers
    - Verifies sender/receiver balances

4. **ConcurrentMixedOperations**
    - Mixed deposits and withdrawals
    - Verifies final balance consistency

5. **OptimisticLockingWithRetries**
    - Simulates lock conflicts
    - Verifies retry mechanism

## Code Review Guide

### Key Areas to Review
1. **Concurrency Control**
    - `internal/util/wallet.go` - Optimistic locking implementation
    - `internal/service/wallet.go` - Transaction handling

2. **Error Handling**
    - `internal/errors/errors.go` - Custom error types
    - Error wrapping throughout service layer

3. **Data Integrity**
    - `internal/repository/` - Database operations
    - Transaction handling in transfers

4. **API Design**
    - `internal/api/wallet.go` - Endpoint handlers
    - Request/response models

### Recommended Review Approach
1. Start with service layer tests to understand business logic
2. Review repository implementations for data access patterns
3. Examine API layer for request validation and error handling
4. Focus on concurrent test cases to verify thread safety

## Areas for Improvement

1. **Pagination**: More sophisticated pagination for transaction history
2. **Caching**: Redis integration for frequently accessed wallets
3. **Batch Operations**: Support for batch deposits/withdrawals
4. **Currency Conversion**: Multi-currency support with exchange rates
5. **Performance Metrics**: Add instrumentation for monitoring
6. **Enhanced Validation**: More robust input validation
7. **API Documentation**: Swagger/OpenAPI documentation


## Omitted Features (can consider as later system optimization)

1. **Authentication/Authorization**: Left for API gateway layer
2. **Rate Limiting**: Would be handled at infrastructure level
3. **WebSockets**: Real-time notifications not implemented
4. **Admin Endpoints**: Wallet administration functions
5. **Reporting**: Advanced financial reporting
6. **Event Sourcing**: Full audit history via events
7. **Saga Pattern**: For distributed transactions across services

The focus was kept on core wallet operations with robust concurrency control, as these were the most critical requirements for the initial implementation.