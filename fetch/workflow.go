package fetch

import (
	"github.com/faizauthar12/paystore/config"
	"github.com/faizauthar12/paystore/lib/payment"
	"github.com/redis/go-redis/v9"
)

type PaystoreFetcher struct {
	paymentFetcher payment.FetcherClient
}

func (pf *PaystoreFetcher) FetchByBalance(lastRandId []string, balanceUUID string) ([]*payment.Payment, *string, *string, bool, error) {
	isBlank, err := pf.paymentFetcher.IsBlankByBalance(balanceUUID)
	if err != nil {
		return nil, nil, nil, false, err
	}
	if isBlank {
		return nil, nil, nil, false, nil
	}

	payments, validLastRandId, position, errFetch := pf.paymentFetcher.FetchByBalance(lastRandId, balanceUUID)
	if errFetch != nil {
		return nil, nil, nil, false, errFetch
	}
	if int64(len(payments)) < pf.paymentFetcher.GetItemPerPage() {
		return payments, nil, nil, true, nil
	}

	return payments, &validLastRandId, &position, false, nil
}

func NewFetcher(redis redis.UniversalClient, config *config.App) *PaystoreFetcher {
	paymentFetcher := payment.NewFetcher(redis, config)
	return &PaystoreFetcher{
		paymentFetcher: paymentFetcher,
	}
}
