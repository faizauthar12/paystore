package payment

import (
	"github.com/21strive/redifu"
	"github.com/faizauthar12/paystore/config"
	"github.com/redis/go-redis/v9"
)

type FetcherClient interface {
	FetchByBalance(lastRandId []string, balanceRandId string) ([]*Payment, string, string, error)
	IsBlankByBalance(balanceRandId string) (bool, error)
	GetItemPerPage() int64
}

type Fetcher struct {
	base              *redifu.Base[*Payment]
	timelineByBalance *redifu.Timeline[*Payment]
}

func (f *Fetcher) FetchByBalance(lastRandId []string, balanceRandId string) ([]*Payment, string, string, error) {
	return f.timelineByBalance.Fetch([]string{balanceRandId}, lastRandId, nil, nil)
}

func (f *Fetcher) IsBlankByBalance(balanceRandId string) (bool, error) {
	isBlank, errCheck := f.timelineByBalance.IsBlankPage([]string{balanceRandId})
	if errCheck != nil {
		return false, errCheck
	}

	return isBlank, nil
}

func (f *Fetcher) GetItemPerPage() int64 {
	return f.timelineByBalance.GetItemPerPage()
}

func NewFetcher(redis redis.UniversalClient, config *config.App) *Fetcher {
	base := redifu.NewBase[*Payment](redis, "payment:%s", config.RecordAge)
	timelineByBalance := redifu.NewTimeline[*Payment](redis, base, "payment-balance:%s", config.ItemPerPage, redifu.Descending, config.PaginationAge)
	return &Fetcher{
		base:              base,
		timelineByBalance: timelineByBalance,
	}
}
