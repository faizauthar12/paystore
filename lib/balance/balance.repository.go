package balance

import (
	"database/sql"

	"github.com/21strive/redifu"
	"github.com/faizauthar12/paystore/config"
	"github.com/faizauthar12/paystore/lib/organization"
	"github.com/redis/go-redis/v9"
)

var findByUUIDQuery = `SELECT * FROM balance WHERE uuid = $1;`
var findByExternalIDQuery = `SELECT * FROM balance WHERE external_id = $1;`
var createBalanceQuery = `INSERT INTO balance
    (
     uuid, randid, created_at, updated_at, balance,
     last_receive, last_withdraw, income_accumulation, withdraw_accumulation,
     currency, active, external_id, organization_uuid
	) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13);`

type RepositoryClient interface {
	Create(balance *Balance) error
	Update(tx *sql.Tx, balance *Balance) error
	FindByUUID(uuid string) (*Balance, error)
	FindByExternalID(externalID string) (*Balance, error)
	SeedPartial(subtraction int64, lastRandId string, organization organization.Organization) error
}

type Repository struct {
	base                 *redifu.Base[*Balance]
	timeline             *redifu.Timeline[*Balance]
	timelineSeeder       *redifu.TimelineSeeder[*Balance]
	createBalanceStmt    *sql.Stmt
	findByUUIDStmt       *sql.Stmt
	findByExternalIDStmt *sql.Stmt
}

func (br *Repository) Create(balance *Balance) (err error) {
	_, errExec := br.createBalanceStmt.Exec(balance.GetUUID(),
		balance.GetRandId(), balance.GetCreatedAt(), balance.GetUpdatedAt(), balance.Balance,
		balance.LastReceive, balance.LastWithdraw, balance.IncomeAccumulation, balance.WithdrawAccumulation,
		balance.Currency, balance.Active, balance.ExternalID, balance.OrganizationUUID)
	if errExec != nil {
		return errExec
	}

	errSet := br.base.Set(balance)
	if errSet != nil {
		return errSet
	}

	br.timeline.AddItem(balance, []string{balance.OrganizationUUID})

	return nil
}

func (br *Repository) Update(tx *sql.Tx, balance *Balance) (err error) {
	query := `UPDATE balance SET
		updated_at = $1, balance = $2, last_receive = $3, last_withdraw = $4, income_accumulation = $5,
		withdraw_accumulation = $6, currency = $7, active = $8, external_id = $9, organization_uuid = $10
		WHERE uuid = $11`

	_, errExec := tx.Exec(
		query, balance.GetUpdatedAt(), balance.Balance, balance.LastReceive, balance.LastWithdraw,
		balance.IncomeAccumulation, balance.WithdrawAccumulation, balance.Currency, balance.Active,
		balance.ExternalID, balance.OrganizationUUID, balance.GetUUID())
	if errExec != nil {
		return errExec
	}

	errSet := br.base.Set(balance)
	if errSet != nil {
		return errSet
	}

	return nil
}

func (br *Repository) FindByUUID(uuid string) (*Balance, error) {
	account, errFind := BalanceRowScanner(br.findByUUIDStmt.QueryRow(uuid))
	if errFind != nil {
		if errFind == sql.ErrNoRows {
			return nil, BalanceNotFound
		}
		return nil, errFind
	}

	return account, nil
}

func (br *Repository) FindByExternalID(externalID string) (*Balance, error) {
	account, errFind := BalanceRowScanner(br.findByExternalIDStmt.QueryRow(externalID))
	if errFind != nil {
		if errFind == sql.ErrNoRows {
			return nil, BalanceNotFound
		}
		return nil, errFind
	}

	return account, nil
}

func (br *Repository) SeedPartial(subtraction int64, lastRandId string, organization organization.Organization) error {
	baseQuery := `SELECT
    	uuid, randid, created_at, updated_at, balance, last_receive, last_withdraw, income_accumulation,
    	withdraw_accumulation, currency, active, external_id, organization_uuid FROM balance`

	rowQuery := baseQuery + ` WHERE randid = $1`
	firstPageQuery := baseQuery + ` WHERE organization_uuid = $1 ORDER BY created_at DESC`
	nextPageQuery := baseQuery + ` WHERE organization_uuid = $1 AND created_at < $2 ORDER BY created_at DESC`

	return br.timelineSeeder.SeedPartial(
		rowQuery, firstPageQuery, nextPageQuery, BalanceRowScanner, BalanceRowsScanner,
		[]interface{}{organization.GetUUID()}, subtraction, lastRandId, []string{organization.GetRandId()})
}

func BalanceRowScanner(row *sql.Row) (*Balance, error) {
	balance := NewBalance()
	err := row.Scan(balance.ScanDestinations()...)

	return balance, err
}

func BalanceRowsScanner(rows *sql.Rows) (*Balance, error) {
	balance := NewBalance()
	err := rows.Scan(balance.ScanDestinations()...)

	return balance, err
}

func NewRepository(writeDB *sql.DB, readDB *sql.DB, redis redis.UniversalClient, config *config.App) *Repository {
	base := redifu.NewBase[*Balance](redis, "balance:%s", config.RecordAge)
	timeline := redifu.NewTimeline[*Balance](redis, base, "balance:organization:%s", config.ItemPerPage, redifu.Descending, config.PaginationAge)
	timelineSeeder := redifu.NewTimelineSeeder[*Balance](readDB, base, timeline)

	findByUUIDStmt, err := readDB.Prepare(findByUUIDQuery)
	if err != nil {
		panic(err)
	}
	findByExternalIDStmt, err := readDB.Prepare(findByExternalIDQuery)
	if err != nil {
		panic(err)
	}
	createBalanceStmt, err := writeDB.Prepare(createBalanceQuery)
	if err != nil {
		panic(err)
	}

	return &Repository{
		base:                 base,
		timeline:             timeline,
		timelineSeeder:       timelineSeeder,
		findByUUIDStmt:       findByUUIDStmt,
		findByExternalIDStmt: findByExternalIDStmt,
		createBalanceStmt:    createBalanceStmt,
	}
}
