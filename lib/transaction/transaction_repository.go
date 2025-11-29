package transaction

import (
	"database/sql"

	"github.com/21strive/redifu"
	"github.com/faizauthar12/paystore/config"
	"github.com/redis/go-redis/v9"
)

type RepositoryClient interface {
	Create(tx *sql.Tx, transaction *Transaction) error
}

type Repository struct {
	base                    *redifu.Base[*Transaction]
	timelineByBalance       *redifu.Timeline[*Transaction]
	timelineSeederByBalance *redifu.TimelineSeeder[*Transaction]
}

func (r *Repository) Create(tx *sql.Tx, transaction *Transaction) error {
	query := `INSERT INTO transaction (uuid, randid, created_at, updated_at, transaction_type, record_uuid, balance_uuid) VALUES ($1, $2, $3, $4, $5, $6, $7)`

	_, errExec := tx.Exec(query, transaction.GetUUID(), transaction.GetRandId(), transaction.GetCreatedAt(),
		transaction.GetUpdatedAt(), transaction.TransactionType, transaction.RecordUUID, transaction.BalanceUUID)
	if errExec != nil {
		return errExec
	}

	errSet := r.base.Set(transaction)
	if errSet != nil {
		return errSet
	}

	r.timelineByBalance.AddItem(transaction, []string{transaction.BalanceUUID})
	return nil
}

func NewRepository(writeDB *sql.DB, readDB *sql.DB, redis redis.UniversalClient, config *config.App) *Repository {
	base := redifu.NewBase[*Transaction](redis, "transaction:%s", config.RecordAge)
	timelineByBalance := redifu.NewTimeline[*Transaction](redis, base, "transaction:balance:%s", config.ItemPerPage, redifu.Descending, config.PaginationAge)
	timelineSeederByBalance := redifu.NewTimelineSeeder[*Transaction](readDB, base, timelineByBalance)

	return &Repository{
		base:                    base,
		timelineByBalance:       timelineByBalance,
		timelineSeederByBalance: timelineSeederByBalance,
	}
}
