package user

import (
	"time"

	"github.com/21strive/redifu"
	"github.com/faizauthar12/paystore/lib/helper"
)

type WithdrawVendor struct {
	// TODO: Fill attributes of your withdraw vendor attributes here
	// This model maps webhook data from your payment provider to your database schema
	// Customize these fields to match your specific withdraw vendor's webhook attributes
	*redifu.Record
	ID                   string    `json:"id,omitempty" db:"id"`
	Amount               float64   `json:"amount" db:"amount"`
	ChannelCode          string    `json:"channelCode" db:"channel_code"`
	Currency             string    `json:"currency" db:"currency"`
	Description          string    `json:"description,omitempty" db:"description"`
	ReferenceID          string    `json:"referenceId" db:"reference_id"`
	Status               string    `json:"status" db:"status"`
	Updated              time.Time `json:"updated" db:"updated"`
	Created              time.Time `json:"created" db:"created"`
	EstimatedArrivalTime time.Time `json:"estimatedDisbursementArrival,omitempty" db:"estimated_arrival_time"`
	FailureCode          string    `json:"failureCode" db:"failure_code"`
	BusinessID           string    `json:"businessId" db:"business_id"`
	AccountHolderName    string    `json:"accountHolderName" db:"account_holder_name"`
	AccountNumber        string    `json:"accountNumber" db:"account_number"`
	AccountType          string    `json:"accountType,omitempty" db:"account_type"`
	EmailTo              []string  `json:"emailTo" db:"email_to"`
	EmailCC              []string  `json:"emailCc" db:"email_cc"`
	EmailBCC             []string  `json:"emailBcc" db:"email_bcc"`
}

func (w *WithdrawVendor) ScanDestionations() []interface{} {
	// TODO: Fill scan destinations of your WithdrawVendo model here
	// This function returns pointers to struct fields for database row scanning
	// The order must exactly match the field order in your WithdrawVendor struct definition

	return []interface{}{
		&w.UUID,
		&w.RandId,
		&w.CreatedAt,
		&w.UpdatedAt,
		&w.ID,
		&w.Amount,
		&w.ChannelCode,
		&w.Currency,
		&w.Description,
		&w.ReferenceID,
		&w.Status,
		&w.Updated,
		&w.Created,
		&w.EstimatedArrivalTime,
		&w.FailureCode,
		&w.BusinessID,
		&w.AccountHolderName,
		&w.AccountNumber,
		&w.AccountType,
		&w.EmailTo,
		&w.EmailCC,
		&w.EmailBCC,
	}
}

func (w *WithdrawVendor) GetFields() []string {
	return helper.FetchColumns(w)
}

func NewWithdrawVendor() *WithdrawVendor {
	vendor := &WithdrawVendor{}
	redifu.InitRecord(vendor)
	return vendor
}
