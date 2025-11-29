package transaction

import (
	"github.com/21strive/redifu"
	"github.com/faizauthar12/paystore/lib/balance"
)

type CommonTransaction interface {
	GetUUID() string
}

type Transaction struct {
	*redifu.Record
	TransactionType TransactionType `json:"transcationType"`
	RecordUUID      string          `json:"recordUUID"`
	BalanceUUID     string          `json:"balanceUUID"`
}

func (t *Transaction) SetType(transactionType TransactionType) {
	t.TransactionType = transactionType
}

func (t *Transaction) SetRecord(transaction CommonTransaction) {
	t.RecordUUID = transaction.GetUUID()
}

func (t *Transaction) SetBalance(balance *balance.Balance) {
	t.BalanceUUID = balance.UUID
}

func NewTransaction() *Transaction {
	transaction := &Transaction{}
	redifu.InitRecord(transaction)
	return transaction
}
