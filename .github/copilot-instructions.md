# Paystore Project Documentation

## 📋 Project Overview

**Paystore** is a Go-based payment and balance management microservice that handles user balances, payments, and withdrawals with vendor integration. It exposes both gRPC and REST API endpoints for managing financial transactions.

### Key Characteristics
- **Language**: Go 1.24.0
- **Architecture**: Microservice with gRPC + REST (Fiber)
- **Database**: PostgreSQL (read/write replicas)
- **Caching**: Redis
- **Communication**: gRPC with Protocol Buffers
- **Company**: 21strive

---

## 🏗️ Project Architecture

### High-Level Flow

```
External Client
      ↓
  gRPC/REST API
      ↓
  gRPC Handler / Operation Layer
      ↓
  Business Logic (PaystoreClient)
      ↓
  Repository Layer (Data Access)
      ↓
PostgreSQL (Write/Read) + Redis Cache
```

### Technology Stack

| Component | Technology | Purpose |
|-----------|-----------|---------|
| Framework | Fiber v2 | REST API server |
| RPC | gRPC | Microservice communication |
| Database | PostgreSQL | Main data store |
| Cache | Redis | Session & query caching |
| Serialization | Protocol Buffers | gRPC message format |
| Libraries | `redifu`, `item` | 21strive utilities |

---

## 📂 Project Structure

```
paystore/
├── main.go                 # Entry point - gRPC server setup
├── main_test.go           # Tests
├── migrate.go             # Database schema & initialization
├── go.mod / go.sum        # Dependency management
│
├── config/                # Application configuration
│   └── config.go          # DefaultConfig, vendor table setup
│
├── operation/             # Core business logic
│   ├── paystore.proto     # gRPC service definitions
│   ├── grpc_handler.go    # gRPC endpoint handlers
│   └── workflow.go        # PaystoreClient - main service logic
│
├── lib/                   # Domain models & repositories
│   ├── balance/           # Balance/account management
│   ├── payment/           # Payment transactions
│   ├── withdraw/          # Withdrawal transactions
│   ├── organization/      # Organization/tenant management
│   ├── transaction/       # Transaction ledger
│   ├── pin/               # PIN management (unused currently)
│   ├── builder/           # Query builders
│   └── helper/            # Utilities (DB connection, logging)
│
├── user/                  # Vendor models
│   ├── payment_vendor_model.go   # Payment provider data
│   └── withdraw_vendor_model.go  # Withdraw provider data
│
├── fetch/                 # External data fetching
│   ├── http_handler.go    # HTTP webhook handlers
│   └── workflow.go        # Fetch logic
│
├── protos/                # Generated Protocol Buffer files
│   ├── paystore.pb.go
│   └── paystore_grpc.pb.go
│
└── .github/               # GitHub configuration
    └── copilot-instructions.md  # This file
```

---

## 🔑 Core Concepts

### 1. **Balance Model**
Represents a user's account with financial data:
- `UUID`: Unique identifier
- `Balance`: Current balance amount (int64)
- `Currency`: Currency code (e.g., "IDR")
- `OrganizationUUID`: Tenant/organization reference
- `ExternalID`: Third-party system identifier
- `IncomeAccumulation`: Total income received
- `WithdrawAccumulation`: Total withdrawn
- `LastReceive`, `LastWithdraw`: Timestamp tracking

**Key Methods**:
- `Collect(amount)`: Add funds to balance
- `Withdraw(amount)`: Deduct funds from balance
- `Deactivate()`: Mark account as inactive

### 2. **Payment Model**
Represents incoming payment transactions:
- `Amount`: Payment amount (before fees)
- `Fees`: Transaction fees
- `BalanceBeforePayment`: Previous balance
- `BalanceAfterPayment`: Balance after transaction
- `Status`: PENDING → PAID / FAILED
- `Hash`: SHA256 hash for integrity (includes previous payment hash)
- `VendorRecordID`: External payment provider reference

**Key Methods**:
- `SetAmount()`: Calculate amount with organization fees
- `GenerateHash()`: Create cryptographic hash with chain verification
- `Verify()`: Validate payment integrity
- `SetPaid()` / `SetFailed()`: Update payment status

### 3. **Withdraw Model**
Represents outgoing withdrawal transactions:
- Similar structure to Payment
- `Status`: PENDING → SUCCESS / FAILED
- Checks `InsufficientFunds` before creating
- Updates balance only on success

### 4. **Organization Model**
Represents business tenants:
- `Name`, `Slug`: Organization identification
- `FeesConstant`: Fixed or percentage fee amount
- `FeesType`: FIXED or PERCENT fee calculation
- Used to determine payment fees

### 5. **Transaction Model**
Ledger entry for all balance changes:
- `TransactionType`: PAYMENT or WITHDRAW
- `RecordUUID`: Reference to Payment/Withdraw
- `BalanceUUID`: Associated balance
- Audit trail for all operations

---

## 🔄 Request Flow

### Creating a Payment

1. **Client calls**: `CreatePaymentRequest` (gRPC)
   - Provides: `AccountUUID`, `Amount`

2. **gRPC Handler** (`grpc_handler.go`)
   - Converts protobuf to Go types
   - Calls `PaystoreClient.CreatePayment()`

3. **PaystoreClient** (`workflow.go`)
   - Fetches balance & organization
   - Calculates fees based on organization settings
   - Creates Payment model with status PENDING
   - Generates hash for integrity
   - Creates Transaction record
   - Begins database transaction

4. **Repositories** (payment, transaction)
   - Insert payment & transaction records
   - Update balances (deferred)
   - Commit on success

5. **Response**: Returns `CreatedResponse` with payment UUID

### Finalizing Payment (Success Flow)

1. **Client calls**: `FinalizedPaymentRequest` with `PaymentStatus = PAID`

2. **PaystoreClient.FinalizedPayment()**
   - Fetches existing payment & balance
   - Sets payment status to PAID
   - Records vendor record ID
   - Updates balance:
     - `LastReceive` = payment timestamp
     - `Balance` += payment amount
     - `IncomeAccumulation` += amount
   - Database transaction commit

---

## 📊 Database Schema

### Tables

#### `balance`
```sql
uuid VARCHAR PRIMARY KEY
randid VARCHAR NOT NULL
balance BIGINT (current amount)
last_receive TIMESTAMP
last_withdraw TIMESTAMP
income_accumulation BIGINT
withdraw_accumulation BIGINT
currency VARCHAR(3)
active BOOL
external_id VARCHAR
organization_uuid UUID
```

#### `payment`
```sql
uuid VARCHAR PRIMARY KEY
amount BIGINT
fees BIGINT
balance_before_payment BIGINT
balance_after_payment BIGINT
balance_uuid VARCHAR
organization_uuid VARCHAR
vendor_record_id VARCHAR
status VARCHAR(20) [PENDING, PAID, FAILED]
hash VARCHAR (SHA256)
```

#### `withdraw`
```sql
uuid VARCHAR PRIMARY KEY
amount BIGINT
balance_before_withdraw BIGINT
balance_after_withdraw BIGINT
balance_uuid VARCHAR
organization_uuid VARCHAR
vendor_record_id VARCHAR
status VARCHAR(20) [PENDING, SUCCESS, FAILED]
hash VARCHAR
```

#### `transaction`
```sql
uuid VARCHAR PRIMARY KEY
transaction_type VARCHAR [PAYMENT, WITHDRAW]
record_uuid VARCHAR (payment/withdraw UUID)
balance_uuid VARCHAR
```

#### `organization`
```sql
uuid VARCHAR PRIMARY KEY
name VARCHAR
slug VARCHAR
fees_constant BIGINT
fees_type VARCHAR [FIXED, PERCENT]
```

---

## 🌐 gRPC API Reference

### Service: Paystore

```protobuf
service Paystore {
  rpc CreateBalance (CreateBalanceRequest) returns (CreatedResponse);
  rpc CreatePayment (CreatePaymentRequest) returns (CreatedResponse);
  rpc FinalizedPayment (FinalizedPaymentRequest) returns (EmptyResponse);
  rpc CreateWithdraw (CreateWithdrawRequest) returns (CreatedResponse);
  rpc FinalizedWithdraw (FinalizedWithdrawRequest) returns (EmptyResponse);
}
```

### Request/Response Messages

#### CreateBalance
```
Input:
  - ExternalID: string (external system ID)
  - OrganizationSlug: string (tenant identifier)
  - Currency: string (e.g., "IDR")

Output:
  - ID: string (balance UUID)
```

#### CreatePayment
```
Input:
  - AccountUUID: string (balance UUID)
  - Amount: int64 (payment amount)

Output:
  - ID: string (payment UUID)
```

#### FinalizedPayment
```
Input:
  - AccountUUID: string
  - PaymentUUID: string
  - VendorRecordId: string (payment provider ref)
  - PaymentStatus: enum [PENDING, PAID, FAILED]

Output: Empty
```

#### CreateWithdraw
```
Input:
  - AccountUUID: string
  - Amount: int64

Output:
  - ID: string (withdraw UUID)
```

#### FinalizedWithdraw
```
Input:
  - WithdrawUUID: string
  - VendorRecordId: string
  - WithdrawStatus: enum [PENDING, SUCCESS, FAILED]

Output: Empty
```

---

## ⚙️ Configuration

### Environment Variables

| Variable | Purpose | Example |
|----------|---------|---------|
| `DB_WRITE_HOST` | Write database host | localhost |
| `DB_WRITE_PORT` | Write DB port | 5432 |
| `DB_WRITE_USER` | Write DB user | postgres |
| `DB_WRITE_PASSWORD` | Write DB password | - |
| `DB_WRITE_NAME` | Write database name | paystore |
| `DB_WRITE_SSLMODE` | SSL mode | disable |
| `DB_READ_HOST` | Read replica host | localhost |
| `DB_READ_PORT` | Read replica port | 5432 |
| `DB_READ_USER` | Read DB user | postgres |
| `DB_READ_PASSWORD` | Read DB password | - |
| `DB_READ_NAME` | Read database name | paystore |
| `DB_READ_SSLMODE` | SSL mode | disable |
| `REDIS_HOST` | Redis server address | localhost:6379 |
| `REDIS_USER` | Redis username | - |
| `REDIS_PASS` | Redis password | - |
| `GRPC_PORT` | gRPC server port | 50051 |
| `PORT` | REST API port | 3000 |
| `PAYMENT_VENDOR_TABLE_NAME` | Payment vendor table | payments_vendor |
| `WITHDRAW_VENDOR_TABLE_NAME` | Withdraw vendor table | withdraws_vendor |

### Config Object (DefaultConfig)

Located in `config/config.go`:
- `ItemPerPage`: 50 records per page
- `RecordAge`: 12 hours (cache TTL)
- `PaginationAge`: 24 hours (pagination cache TTL)
- Auto-generates table aliases for queries

---

## 🔐 Data Integrity

### Hash Chain Verification

Payments use SHA256 hashing to maintain data integrity:

```go
HashPayload {
  UUID, RandID, CreatedAt
  Amount, Fees
  BalanceBeforePayment, BalanceAfterPayment
  BalanceUUID, OrganizationUUID
  VendorRecordID, Status
  PreviousPaymentHash ← Chain linking
}
```

**Purpose**: 
- Detect tampering
- Maintain immutable transaction chain
- Audit trail verification

---

## 📦 Vendor Models

### PaymentVendor
Maps webhook data from payment providers (e.g., Xendit):
- Invoice ID & external references
- Payment status & amounts
- Customer information (name, email, phone)
- Payment method & channel details
- Currency & fee information

### WithdrawVendor
Maps withdrawal provider data (structure similar to PaymentVendor)

**Customization**: Edit `/user/*.go` files to match your payment provider's webhook format.

---

## 🛠️ Key Libraries & Dependencies

### 21strive Internal
- `github.com/21strive/redifu` - Redis utilities with Record base type
- `github.com/21strive/item` - Utilities (RandID generation, etc.)

### External
- `github.com/gofiber/fiber/v2` - Web framework
- `google.golang.org/grpc` - RPC framework
- `google.golang.org/protobuf` - Protocol buffers
- `github.com/lib/pq` - PostgreSQL driver
- `github.com/redis/go-redis/v9` - Redis client
- `golang.org/x/crypto` - Cryptography utilities

---

## 🧪 Testing

- Test file: `main_test.go` (currently empty)
- Use repository pattern for testability
- Mock database & redis for unit tests

---

## 🚀 Running the Application

### Prerequisites
- Go 1.24.0+
- PostgreSQL (write & read replicas)
- Redis
- Network access between services

### Startup
```bash
# Load environment variables
export DB_WRITE_HOST=...
export REDIS_HOST=...
# ... (see Configuration section)

# Run application
go run main.go
```

**Servers started**:
- gRPC: `localhost:50051` (or configured GRPC_PORT)
- REST API: `localhost:3000` (or configured PORT)

### Database Migration
```bash
# SQL migration defined in migrate.go
# Execute createTableBalance, createTableQuery, etc.
```

---

## 💡 Important Implementation Notes

### 1. Database Transactions
- Critical operations use database transactions
- Rollback on any error
- Commit only on success
- Ensures ACID compliance

### 2. Read/Write Separation
- Write operations → `writeDB`
- Read operations → `readDB` (replica)
- Improves performance & scalability

### 3. Status Management
Payment Status:
- `PENDING`: Initial state
- `PAID`: Successfully completed
- `FAILED`: Transaction failed

Withdraw Status:
- `PENDING`: Initial state
- `SUCCESS`: Successfully completed
- `FAILED`: Transaction failed

### 4. Fee Calculation
Based on organization settings:
- **FIXED**: `Amount = GrossAmount - FixedFee`, `Fee = FixedFee`
- **PERCENT**: `Amount = GrossAmount`, `Fee = GrossAmount * PercentRate / 100`

### 5. Repository Pattern
Each domain has repository interface:
- `balance.RepositoryClient`
- `payment.RepositoryClient`
- `withdraw.RepositoryClient`
- `organization.RepositoryClient`
- `transaction.RepositoryClient`

Enables:
- Easy mocking for tests
- Loose coupling
- Testability

---

## 🔍 Common Workflows

### Workflow 1: Receive Payment
```
1. CreatePayment() → Payment with status PENDING
2. Payment provider processes payment
3. FinalizedPayment(PAID) → Update balance, record transaction
```

### Workflow 2: Process Withdrawal
```
1. CreateWithdraw() → Validate funds available
2. Withdraw() with status PENDING
3. FinalizedWithdraw(SUCCESS) → Deduct from balance
4. If FinalizedWithdraw(FAILED) → Retry or notify user
```

### Workflow 3: Multi-tenant Operations
```
1. CreateBalance(organizationSlug) → Look up org fees
2. Apply org-specific fee configuration
3. All subsequent payments use those fee rules
```

---

## 🐛 Debugging Tips

### Check Service Status
```go
// In main.go, gRPC server runs in goroutine
// REST API on main thread
// Both must be listening for full service
```

### Database Issues
- Check connection strings in environment variables
- Verify SSL mode matches your PostgreSQL setup
- Ensure write/read database credentials are correct

### Redis Issues
- Verify Redis is running: `redis-cli ping`
- Check REDIS_HOST includes port (`:6379`)
- Verify authentication if enabled

### Transaction Rollbacks
- All payment/withdraw creations use transactions
- Check database logs if commits fail
- Verify table schemas match `migrate.go`

---

## 📝 Common Errors & Solutions

| Error | Cause | Solution |
|-------|-------|----------|
| `OrganizationNotFound` | Organization slug doesn't exist | Create organization in DB first |
| `BalanceNotFound` | Invalid balance/account UUID | Use valid balance UUID |
| `InsufficientFunds` | Withdraw amount > current balance | Check balance before withdraw |
| `FinalAmountLessThanZero` | FIXED fees exceed payment amount | Increase payment or reduce fees |
| Database connection fail | Wrong credentials/host | Verify environment variables |
| Redis connection fail | Redis not running | Start Redis service |

---

## 🎯 Next Steps for New Developers

1. **Read this document** - You're doing it! ✓
2. **Explore the code**:
   - Start with `main.go` entry point
   - Review `operation/workflow.go` for business logic
   - Check specific repository implementations
3. **Understand the models**:
   - Balance, Payment, Withdraw, Organization
   - Relationships and constraints
4. **Review gRPC definitions**: `operation/paystore.proto`
5. **Set up local environment**:
   - PostgreSQL with test database
   - Redis instance
   - Environment variables
6. **Run tests** and familiarize with workflows
7. **Check vendor models** if integrating payment provider

---

## 📞 Support Resources

- **gRPC Debugging**: Use `grpcurl` for testing
- **PostgreSQL**: Connect with `psql` for manual queries
- **Redis**: Use `redis-cli` for debugging
- **Logs**: Check JSON-formatted logs from `helper/helper.go`

---

**Last Updated**: 2024
**Project**: Paystore v1.0
**Maintained By**: 21strive Team