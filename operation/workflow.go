package operation

import (
	"database/sql"

	"github.com/faizauthar12/paystore/config"
	"github.com/faizauthar12/paystore/lib/balance"
	"github.com/faizauthar12/paystore/lib/organization"
	"github.com/faizauthar12/paystore/lib/payment"
	"github.com/faizauthar12/paystore/lib/transaction"
	"github.com/faizauthar12/paystore/lib/withdraw"

	"github.com/redis/go-redis/v9"
)

type OrganizationClient struct {
	organizationRepository organization.RepositoryClient
}

type PaystoreClient struct {
	writeDB                *sql.DB
	balanceRepository      balance.RepositoryClient
	paymentRepository      payment.RepositoryClient
	transactionRepository  transaction.RepositoryClient
	organizationRepository organization.RepositoryClient
	withdrawRepository     withdraw.RepositoryClient
}

func (ps *PaystoreClient) CreateOrganization(name, slug string) (*organization.Organization, error) {

	// find if organization with the same slug already exists
	// if exists, return it
	isOrganizationExist := true
	newOrganization, err := ps.organizationRepository.FindBySlug(slug)
	if err != nil {
		if err == organization.OrganizationNotFound {
			isOrganizationExist = false
		} else {
			return nil, err
		}
	}

	if isOrganizationExist {
		return newOrganization, nil
	}

	newOrganization = organization.NewOrganization()
	newOrganization.Name = name
	newOrganization.Slug = slug

	err = ps.organizationRepository.Create(newOrganization)
	if err != nil {
		return nil, err
	}

	return newOrganization, nil
}

func (ps *PaystoreClient) CreateBalance(externalID string,
	currency string, organizationSlug string) (*balance.Balance, error) {
	organizationFromDB, errFind := ps.organizationRepository.FindBySlug(organizationSlug)
	if errFind != nil {
		return nil, errFind
	}

	newBalance := balance.NewBalance()
	newBalance.OrganizationUUID = organizationFromDB.GetUUID()
	newBalance.Currency = currency
	newBalance.ExternalID = externalID
	newBalance.Active = true

	errCreate := ps.balanceRepository.Create(newBalance)
	if errCreate != nil {
		return nil, errCreate
	}

	return newBalance, nil
}

func (ps *PaystoreClient) CreatePayment(accountUUID string, amount int64) (*payment.Payment, error) {
	balanceFromDB, errFind := ps.balanceRepository.FindByUUID(accountUUID)
	if errFind != nil {
		return nil, errFind
	}

	organizationFromDB, errFind := ps.organizationRepository.FindByUUID(balanceFromDB.OrganizationUUID)
	if errFind != nil {
		return nil, errFind
	}

	previousPayment, errFind := ps.paymentRepository.FindLatestPayment(balanceFromDB)
	if errFind != nil {
		return nil, errFind
	}

	newPayment := payment.NewPayment()
	newPayment.SetBalance(balanceFromDB)
	newPayment.SetAmount(amount, balanceFromDB.Balance, organizationFromDB)
	newPayment.GenerateHash(previousPayment)

	newPayment.OrganizationUUID = organizationFromDB.GetUUID()

	newTranscation := transaction.NewTransaction()
	newTranscation.SetType(transaction.TypePayment)
	newTranscation.SetRecord(newPayment)
	newTranscation.SetBalance(balanceFromDB)

	tx, errInitTx := ps.writeDB.Begin()
	if errInitTx != nil {
		return nil, errInitTx
	}
	defer tx.Rollback()

	errCreatePayment := ps.paymentRepository.Create(tx, newPayment, balanceFromDB, organizationFromDB)
	if errCreatePayment != nil {
		return nil, errCreatePayment
	}

	errCreateTransaction := ps.transactionRepository.Create(tx, newTranscation)
	if errCreateTransaction != nil {
		return nil, errCreateTransaction
	}

	errCommit := tx.Commit()
	if errCommit != nil {
		return nil, errCommit
	}

	return newPayment, nil
}

func (ps *PaystoreClient) FinalizedPayment(accountUUID string,
	paymentUUID string, paymentStatus payment.PaymentStatus, vendorRecordID string) error {
	balanceFromDB, errFind := ps.balanceRepository.FindByUUID(accountUUID)
	if errFind != nil {
		return errFind
	}

	paymentFromDB, errFind := ps.paymentRepository.FindByUUID(paymentUUID)
	if errFind != nil {
		return errFind
	}

	updateBalance := false
	if paymentStatus == payment.PaymentStatusFailed {
		paymentFromDB.SetFailed()
	} else if paymentStatus == payment.PaymentStatusPaid {
		paymentFromDB.SetPaid()
		paymentFromDB.SetVendorRecord(vendorRecordID)
		balanceFromDB.LastReceive = paymentFromDB.GetCreatedAt()
		balanceFromDB.Collect(paymentFromDB.Amount)
		updateBalance = true
	}

	tx, errInitTx := ps.writeDB.Begin()
	if errInitTx != nil {
		return errInitTx
	}
	defer tx.Rollback()

	errUpdatePayment := ps.paymentRepository.Update(tx, paymentFromDB)
	if errUpdatePayment != nil {
		return errUpdatePayment
	}

	if updateBalance {
		errUpdateBalance := ps.balanceRepository.Update(tx, balanceFromDB)
		if errUpdateBalance != nil {
			return errUpdateBalance
		}
	}

	errCommit := tx.Commit()
	if errCommit != nil {
		return errCommit
	}

	return nil
}

func (ps *PaystoreClient) CreateWithdraw(accountUUID string, amount int64) (*withdraw.Withdraw, error) {
	balanceFromDB, errFind := ps.balanceRepository.FindByUUID(accountUUID)
	if errFind != nil {
		return nil, errFind
	}

	organizationFromDB, errFind := ps.organizationRepository.FindByUUID(balanceFromDB.OrganizationUUID)
	if errFind != nil {
		return nil, errFind
	}

	if balanceFromDB.Balance < amount {
		return nil, balance.InsufficientFunds
	}

	newWithdraw := withdraw.NewWithdraw()
	newWithdraw.SetBalance(balanceFromDB)
	newWithdraw.SetAmount(amount, balanceFromDB.Balance)
	newWithdraw.SetOrganization(organizationFromDB)

	newTransaction := transaction.NewTransaction()
	newTransaction.SetType(transaction.TypeWithdraw)
	newTransaction.SetRecord(newWithdraw)
	newTransaction.SetBalance(balanceFromDB)

	tx, errInitTx := ps.writeDB.Begin()
	if errInitTx != nil {
		return nil, errInitTx
	}
	defer tx.Rollback()

	errCreate := ps.withdrawRepository.Create(tx, newWithdraw, balanceFromDB, organizationFromDB)
	if errCreate != nil {
		return nil, errCreate
	}

	errCreate = ps.transactionRepository.Create(tx, newTransaction)
	if errCreate != nil {
		return nil, errCreate
	}

	errCommit := tx.Commit()
	if errCommit != nil {
		return nil, errCommit
	}

	return newWithdraw, nil
}

func (ps *PaystoreClient) FinalizedWithdraw(accountUUID string,
	withdrawUUID string, withdrawStatus withdraw.WithdrawStatus, vendorRecordID string) error {
	balanceFromDB, errFind := ps.balanceRepository.FindByUUID(accountUUID)
	if errFind != nil {
		return errFind
	}

	withdrawFromDB, errFind := ps.withdrawRepository.FindByUUID(withdrawUUID)
	if errFind != nil {
		return errFind
	}

	updateBalance := false
	if withdrawStatus == withdraw.StatusFailed {
		withdrawFromDB.SetFailed()
	} else {
		withdrawFromDB.SetSuccess()
		withdrawFromDB.SetVendorRecord(vendorRecordID)
		balanceFromDB.LastWithdraw = withdrawFromDB.GetCreatedAt()
		balanceFromDB.Withdraw(withdrawFromDB.Amount)
		updateBalance = true
	}

	tx, errInitTx := ps.writeDB.Begin()
	if errInitTx != nil {
		return errInitTx
	}
	defer tx.Rollback()

	errUpdateWithdraw := ps.withdrawRepository.Update(tx, withdrawFromDB)
	if errUpdateWithdraw != nil {
		return errUpdateWithdraw
	}

	if updateBalance {
		errUpdateBalance := ps.balanceRepository.Update(tx, balanceFromDB)
		if errUpdateBalance != nil {
			return errUpdateBalance
		}
	}

	errCommit := tx.Commit()
	if errCommit != nil {
		return errCommit
	}

	return nil
}

type PaymentSeeder struct {
	ps *PaystoreClient
}

func (psr *PaymentSeeder) ByBalance(subtraction int64, lastRandId string, balanceUUID string) error {
	balanceFromDB, errFind := psr.ps.balanceRepository.FindByUUID(balanceUUID)
	if errFind != nil {
		return errFind
	}
	if balanceFromDB == nil {
		return balance.BalanceNotFound
	}

	return psr.ps.paymentRepository.SeedPartialByBalance(subtraction, lastRandId, balanceFromDB)
}

func (ps *PaystoreClient) SeedPayment() *PaymentSeeder {
	return &PaymentSeeder{ps: ps}
}

type WithdrawSeeder struct {
	ps *PaystoreClient
}

func (psr *WithdrawSeeder) ByBalance(subtraction int64, lastRandId string, balanceUUID string) error {
	balanceFromDB, errFind := psr.ps.balanceRepository.FindByUUID(balanceUUID)
	if errFind != nil {
		return errFind
	}
	if balanceFromDB == nil {
		return balance.BalanceNotFound
	}

	return psr.ps.withdrawRepository.SeedPartialByBalance(subtraction, lastRandId, balanceFromDB)
}

func (ps *PaystoreClient) SeedWithdraw() *WithdrawSeeder {
	return &WithdrawSeeder{ps: ps}
}

func New(writeDB *sql.DB, readDB *sql.DB, redis redis.UniversalClient,
	config *config.App) *PaystoreClient {
	var errInit error

	balanceRepo := balance.NewRepository(writeDB, readDB, redis, config)
	transactionRepo := transaction.NewRepository(writeDB, readDB, redis, config)
	paymentRepo, errInit := payment.NewRepository(readDB, redis, config)
	if errInit != nil {
		panic(errInit)
	}
	withdrawRepo := withdraw.NewRepository(readDB, redis, config)
	organizationRepo := organization.NewRepository(writeDB, readDB, redis, config)

	return Client(writeDB, balanceRepo, paymentRepo, transactionRepo, withdrawRepo, organizationRepo)
}

func Client(writeDB *sql.DB, balanceRepository balance.RepositoryClient,
	paymentRepository payment.RepositoryClient, transactionRepository transaction.RepositoryClient,
	withdrawRepository withdraw.RepositoryClient, organizationRepo organization.RepositoryClient) *PaystoreClient {
	return &PaystoreClient{
		writeDB:                writeDB,
		balanceRepository:      balanceRepository,
		paymentRepository:      paymentRepository,
		transactionRepository:  transactionRepository,
		withdrawRepository:     withdrawRepository,
		organizationRepository: organizationRepo,
	}
}
