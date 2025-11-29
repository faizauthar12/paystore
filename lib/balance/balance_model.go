package balance

import (
	"time"

	"github.com/21strive/redifu"
	"github.com/faizauthar12/paystore/lib/organization"
)

type Balance struct {
	*redifu.Record
	Balance              int64
	LastReceive          time.Time
	LastWithdraw         time.Time
	IncomeAccumulation   int64
	WithdrawAccumulation int64
	Currency             string
	Active               bool
	ExternalID           string
	OrganizationUUID     string
}

func (ac *Balance) SetCurrency(currency string) {
	ac.Currency = currency
}

func (ac *Balance) SetExternalID(externalID string) {
	ac.ExternalID = externalID
}

func (ac *Balance) SetOrganization(organization organization.Organization) {
	ac.OrganizationUUID = organization.GetUUID()
}

func (ac *Balance) Deactivate() {
	ac.Active = false
}

func (ac *Balance) Collect(amount int64) {
	if amount > 0 {
		ac.Balance += amount
		ac.IncomeAccumulation += amount
	}
}

func (ac *Balance) Withdraw(amount int64) error {
	if amount > ac.Balance {
		return InsufficientFunds
	}

	ac.Balance -= amount
	ac.WithdrawAccumulation += amount
	return nil
}

func (ac *Balance) ScanDestinations() []interface{} {
	return []interface{}{
		&ac.UUID,
		&ac.RandId,
		&ac.CreatedAt,
		&ac.UpdatedAt,
		&ac.Balance,
		&ac.LastReceive,
		&ac.LastWithdraw,
		&ac.IncomeAccumulation,
		&ac.WithdrawAccumulation,
		&ac.Currency,
		&ac.Active,
		&ac.ExternalID,
		&ac.OrganizationUUID,
	}
}

func NewBalance() *Balance {
	account := &Balance{}
	redifu.InitRecord(account)

	account.Active = true
	return account
}
