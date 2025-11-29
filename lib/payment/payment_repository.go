package payment

import (
	"database/sql"

	"github.com/21strive/redifu"
	"github.com/faizauthar12/paystore/config"
	"github.com/faizauthar12/paystore/lib/balance"
	"github.com/faizauthar12/paystore/lib/builder"
	"github.com/faizauthar12/paystore/lib/organization"
	"github.com/faizauthar12/paystore/lib/transaction"
	vendorModel "github.com/faizauthar12/paystore/user"
	"github.com/redis/go-redis/v9"
)

var firstPartSelectQuery = `SELECT p.uuid, p.randid, p.created_at, p.updated_at, p.amount, p.balance_before_payment, p.balance_after_payment, p.balance_uuid, p.organization_uuid, p.hash`
var findLatestPaymentQuery = firstPartSelectQuery + ` FROM payment p WHERE p.balance_uuid = $1 ORDER BY created_at DESC LIMIT 1;`
var findPaymentByUUIDQuery = firstPartSelectQuery + ` FROM payment p WHERE p.uuid = $1;`

type RepositoryClient interface {
	Create(tx *sql.Tx, payment *Payment, balance *balance.Balance, organization *organization.Organization) error
	Update(tx *sql.Tx, payment *Payment) error
	FindLatestPayment(balance *balance.Balance) (*Payment, error)
	FindByUUID(uuid string) (*Payment, error)
	SeedPartialByBalance(subtraction int64, lastRandId string, balance *balance.Balance) error
}

type Repository struct {
	readDB                  *sql.DB
	base                    *redifu.Base[*Payment]
	timelineByAccount       *redifu.Timeline[*Payment]
	timelineByAccountSeeder *redifu.TimelineSeeder[*Payment]
	AppConfig               *config.App
	findLatestPaymentStmt   *sql.Stmt
	findPaymentByUUIDStmt   *sql.Stmt
}

func (br *Repository) Create(tx *sql.Tx, payment *Payment, balance *balance.Balance, organization *organization.Organization) error {
	if payment.BalanceUUID != balance.UUID {
		return UnmatchBalance
	}

	createPaymentQuery := `
		INSERT INTO payment (
			uuid, randid, created_at, updated_at,
			amount, balance_before_payment, balance_after_payment,
			balance_uuid, organization_uuid, vendor_record_id, status, hash
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`
	_, err := tx.Exec(
		createPaymentQuery,
		payment.GetUUID(),
		payment.GetRandId(),
		payment.GetCreatedAt(),
		payment.GetUpdatedAt(),
		payment.Amount,
		payment.BalanceBeforePayment,
		payment.BalanceAfterPayment,
		payment.BalanceUUID,
		payment.OrganizationUUID,
		payment.VendorRecordID,
		payment.Status,
		payment.Hash,
	)
	if err != nil {
		return err
	}

	errSet := br.base.Set(payment)
	if errSet != nil {
		return errSet
	}

	errSet = br.timelineByAccount.AddItem(payment, []string{organization.GetRandId(), balance.GetRandId()})
	if errSet != nil {
		return errSet
	}

	return err
}

func (br *Repository) Update(tx *sql.Tx, payment *Payment) error {
	query := `UPDATE payment SET updated_at = $1, organization_uuid = $2,
                   vendor_record_id = $3, status = $4, hash = $5 WHERE uuid = $6`
	_, errExec := tx.Exec(query, payment.GetUpdatedAt(), payment.OrganizationUUID, payment.VendorRecordID,
		payment.Status, payment.Hash, payment.GetUUID())
	if errExec != nil {
		return errExec
	}

	br.base.Set(payment)
	return nil
}

func (br *Repository) FindLatestPayment(balance *balance.Balance) (*Payment, error) {
	payment, err := PaymentRowScanner(br.findLatestPaymentStmt.QueryRow(balance.GetUUID()))
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return payment, nil
}

func (br *Repository) FindByUUID(uuid string) (*Payment, error) {
	payment, err := PaymentRowScanner(br.findPaymentByUUIDStmt.QueryRow(uuid))
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, PaymentNotFound
		}
		return nil, err
	}

	return payment, nil
}

func (br *Repository) SeedPartialByBalance(subtraction int64, lastRandId string, balance *balance.Balance) error {
	joinedQuery := builder.JoinBuilder(firstPartSelectQuery, transaction.TypePayment, br.AppConfig)

	rowQuery := joinedQuery + " WHERE p.randid = $1"
	firstPageQuery := joinedQuery + " WHERE p.balance_uuid = $1 ORDER BY created_at DESC"
	nextPageQuery := joinedQuery + " WHERE p.balance_uuid = $1 AND created_at < $2 ORDER BY created_at DESC"

	return br.timelineByAccountSeeder.SeedPartialWithRelation(rowQuery, firstPageQuery, nextPageQuery,
		PaymentRowScanner, PaymentRowsScanner, []interface{}{balance.GetUUID()},
		subtraction, lastRandId, []string{balance.GetRandId()})
}

func NewRepository(readDB *sql.DB, redis redis.UniversalClient, appConfig *config.App) (*Repository, error) {
	var err error

	if appConfig == nil {
		return nil, ConfigRequired
	}

	vendorRepo := NewVendorRepository(redis, appConfig)

	vendorRelation := redifu.NewRelation[*vendorModel.PaymentVendor](vendorRepo.GetBase(),
		"PaymentVendor", "PaymentVendorRandId")

	basePayment := redifu.NewBase[*Payment](redis, "payment:%s", appConfig.RecordAge)
	timelineByAccount := redifu.NewTimeline[*Payment](redis, basePayment,
		"payment:organization:%s:balance:%s", appConfig.ItemPerPage,
		redifu.Descending, appConfig.PaginationAge)
	timelineByAccount.AddRelation("vendor", vendorRelation)
	timelineByAccountSeeder := redifu.NewTimelineSeeder[*Payment](readDB, basePayment, timelineByAccount)

	findLatestPaymentStmt, err := readDB.Prepare(findLatestPaymentQuery)
	if err != nil {
		panic(err)
	}
	findPaymentByUUIDStmt, err := readDB.Prepare(findPaymentByUUIDQuery)
	if err != nil {
		panic(err)
	}

	return &Repository{
		base:                    basePayment,
		timelineByAccount:       timelineByAccount,
		timelineByAccountSeeder: timelineByAccountSeeder,
		AppConfig:               appConfig,
		findLatestPaymentStmt:   findLatestPaymentStmt,
		findPaymentByUUIDStmt:   findPaymentByUUIDStmt,
	}, nil
}

func PaymentRowScanner(row *sql.Row) (*Payment, error) {
	payment := NewPayment()
	err := row.Scan(payment.ScanDestinations()...)
	return payment, err
}

func PaymentRowsScanner(rows *sql.Rows, relation map[string]redifu.Relation) (*Payment, error) {
	payment := NewPayment()
	paymentVendor := vendorModel.NewPaymentVendor()

	var scanDestinations []interface{}
	scanDestinations = append(scanDestinations, payment.ScanDestinations()...)
	scanDestinations = append(scanDestinations, paymentVendor.ScanDestinations()...)

	err := rows.Scan(scanDestinations)

	if paymentVendor.UUID != "" {
		errSet := relation["vendor"].SetItem(paymentVendor)
		if errSet != nil {
			return nil, errSet
		}
	}

	return payment, err
}

type VendorRepository struct {
	base *redifu.Base[*vendorModel.PaymentVendor]
}

func (r *VendorRepository) GetBase() *redifu.Base[*vendorModel.PaymentVendor] {
	return r.base
}

func NewVendorRepository(redis redis.UniversalClient, config *config.App) *VendorRepository {
	base := redifu.NewBase[*vendorModel.PaymentVendor](redis, "vendor-item:%s", config.RecordAge)
	return &VendorRepository{
		base: base,
	}
}
