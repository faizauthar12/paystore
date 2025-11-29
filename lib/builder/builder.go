package builder

import (
	"github.com/faizauthar12/paystore/config"
	"github.com/faizauthar12/paystore/lib/transaction"
)

func JoinBuilder(firstPartSelectQuery string, transcationType transaction.TransactionType, config *config.App) string {
	var finalQuery string

	finalQuery += firstPartSelectQuery
	finalQuery += `, `

	if transcationType == transaction.TypePayment {
		fields := config.GetPaymentVendorModelFields()
		for _, field := range fields {
			finalQuery += config.GetPaymentVendorTableAlias() + "." + field + ", "
		}

		finalQuery += `FROM payment p`
		finalQuery += ` `
		finalQuery += `LEFT JOIN ` + config.GetPaymentVendorTableName() + ` ` + config.GetPaymentVendorTableAlias()
		finalQuery += ` `
	} else if transcationType == transaction.TypeWithdraw {
		fields := config.GetWithdrawVendorModelFields()
		for _, field := range fields {
			finalQuery += config.GetWithdrawVendorTableAlias() + "." + field + ", "
		}

		finalQuery += `FROM withdraw w`
		finalQuery += ` `
		finalQuery += `LEFT JOIN ` + config.GetWithdrawVendorTableName() + ` ` + config.GetWithdrawVendorTableAlias()
		finalQuery += ` `
	}

	return finalQuery
}
