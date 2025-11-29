package user

import (
	"time"

	"github.com/21strive/redifu"
	"github.com/faizauthar12/paystore/lib/helper"
)

type PaymentVendor struct {
	// TODO: Fill attributes of your payment vendor here
	// This model maps webhook data from your payment provider to your database schema
	// Customize these fields to match your specific payment vendor's webhook format

	*redifu.Record
	ID                     string    `json:"id" db:"id"`                  // Xendit Invoice ID
	ExternalID             string    `json:"externalId" db:"external_id"` // Aturjadwal Invoice ID
	UserID                 string    `json:"userId" db:"user_id"`         // Aturjadwal User ID
	Status                 string    `json:"status" db:"status"`
	Amount                 int64     `json:"amount" db:"amount"`
	PaidAmount             int64     `json:"paidAmount" db:"paid_amount"`
	AdjustedReceivedAmount int64     `json:"adjustedReceivedAmount" db:"adjusted_received_amount"`
	FeesPaidAmount         int64     `json:"feesPaidAmount" db:"fees_paid_amount"`
	PaidAt                 time.Time `json:"paidAt" db:"paid_at"`
	ExpiryDate             time.Time `json:"expiryDate" db:"expiry_date"`
	InvoiceURL             string    `json:"invoiceUrl" db:"invoice_url"`
	GivenName              string    `json:"givenName,omitempty" db:"given_name"`
	Surname                string    `json:"surname,omitempty" db:"surname"`
	Email                  string    `json:"email,omitempty" db:"email"`
	MobileNumber           string    `json:"mobileNumber,omitempty" db:"mobile_number"`
	Created                time.Time `json:"created" db:"created"`
	Updated                time.Time `json:"updated" db:"updated"`
	Currency               string    `json:"currency" db:"currency"`
	BankCode               string    `json:"bankCode" db:"bank_code"`
	PaymentMethod          string    `json:"paymentMethod" db:"payment_method"`
	PaymentChannel         string    `json:"paymentChannel" db:"payment_channel"`
	PaymentDestination     string    `json:"paymentDestination" db:"payment_destination"`
}

func (v *PaymentVendor) ScanDestinations() []interface{} {
	// TODO: Fill scan destinations of your PaymentVendor model here
	// This function returns pointers to struct fields for database row scanning
	// The order must exactly match the field order in your PaymentVendor struct definition

	return []interface{}{
		&v.UUID,
		&v.RandId,
		&v.CreatedAt,
		&v.UpdatedAt,
		&v.ID,
		&v.ExternalID,
		&v.UserID,
		&v.Status,
		&v.Amount,
		&v.PaidAmount,
		&v.AdjustedReceivedAmount,
		&v.FeesPaidAmount,
		&v.PaidAt,
		&v.ExpiryDate,
		&v.InvoiceURL,
		&v.GivenName,
		&v.Surname,
		&v.Email,
		&v.MobileNumber,
		&v.Created,
		&v.Updated,
		&v.Currency,
		&v.BankCode,
		&v.PaymentMethod,
		&v.PaymentChannel,
		&v.PaymentDestination,
	}
}

func (v *PaymentVendor) GetFields() []string {
	return helper.FetchColumns(v)
}

func NewPaymentVendor() *PaymentVendor {
	vendor := &PaymentVendor{}
	redifu.InitRecord(vendor)
	return vendor
}
