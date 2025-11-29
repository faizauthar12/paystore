package config

import (
	"time"

	"github.com/faizauthar12/paystore/lib/helper"
	"github.com/faizauthar12/paystore/user"
)

type App struct {
	ItemPerPage              int64
	RecordAge                time.Duration
	PaginationAge            time.Duration
	PaymentVendorTableAlias  string
	PaymentVendorTableName   string
	paymentVendorSampleItem  *user.PaymentVendor
	WithdrawVendorTableAlias string
	WithdrawVendorTableName  string
	withdrawVendorSampleItem *user.WithdrawVendor
}

func (a *App) GetPaymentVendorTableAlias() string {
	return a.PaymentVendorTableAlias
}

func (a *App) GetPaymentVendorTableName() string {
	return a.PaymentVendorTableName
}

func (a *App) GetPaymentVendorModelFields() []string {
	return helper.FetchColumns(a.paymentVendorSampleItem)
}

func (a *App) GetWithdrawVendorTableAlias() string {
	return a.WithdrawVendorTableAlias
}

func (a *App) GetWithdrawVendorTableName() string {
	return a.WithdrawVendorTableName
}

func (a *App) GetWithdrawVendorModelFields() []string {
	return helper.FetchColumns(a.withdrawVendorSampleItem)
}

func DefaultConfig(paymentVendorTableName string, withdrawVendorTableName string) *App {
	var paymentVendorTableAlias string
	var withdrawVendorTableAlias string

	if len(paymentVendorTableName) > 0 {
		firstChar := string(paymentVendorTableName[0])
		if firstChar == "p" {
			paymentVendorTableAlias = "q"
		} else {
			paymentVendorTableAlias = firstChar
		}
	}

	if len(withdrawVendorTableName) > 0 {
		firstChar := string(withdrawVendorTableName[0])
		if firstChar == "w" {
			withdrawVendorTableAlias = "x"
		}
	}

	paymentVendorSampleItem := user.NewPaymentVendor()
	withdrawVendorSampleItem := user.NewWithdrawVendor()

	return &App{
		ItemPerPage:              50,
		RecordAge:                time.Hour * 12,
		PaginationAge:            time.Hour * 24,
		PaymentVendorTableName:   paymentVendorTableName,
		PaymentVendorTableAlias:  paymentVendorTableAlias,
		paymentVendorSampleItem:  paymentVendorSampleItem,
		WithdrawVendorTableAlias: withdrawVendorTableAlias,
		WithdrawVendorTableName:  withdrawVendorTableName,
		withdrawVendorSampleItem: withdrawVendorSampleItem,
	}
}
