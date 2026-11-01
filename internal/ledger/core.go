package ledger

import (
	"errors"
	"fmt"

	"inmemoryaccountledgercore/internal/domain"
)

var ErrInvalidConfiguration = errors.New("invalid ledger configuration")

// Core processes sequential submissions against an injected repository. Account
// and policy configuration is private; authoritative financial state is history.
// It does not support concurrent writers.
type Core struct {
	repository Repository
	accounts   map[domain.AccountID]domain.Account
	charges    map[domain.Currency]domain.Money
}

// New validates and copies all configuration before atomically appending account
// records. Use a fresh repository for initialization; an append failure returns
// no core and leaves history unchanged under the Repository contract.
// Every account currency requires an explicit policy; zero means no fee.
func New(repository Repository, accounts []domain.Account, charges map[domain.Currency]domain.Money) (*Core, error) {
	if repository == nil {
		return nil, fmt.Errorf("%w: missing repository", ErrInvalidConfiguration)
	}
	c := &Core{
		repository: repository,
		accounts:   make(map[domain.AccountID]domain.Account, len(accounts)),
		charges:    make(map[domain.Currency]domain.Money, len(charges)),
	}
	if err := c.configure(accounts, charges); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidConfiguration, err)
	}
	batch := domain.Batch{Records: make([]domain.Record, 0, len(accounts))}
	for _, account := range accounts {
		copy := account
		batch.Records = append(batch.Records, domain.Record{
			Key: domain.RecordKey("account/" + string(account.ID)), Account: &copy,
		})
	}
	if err := repository.Append(batch); err != nil {
		return nil, err
	}
	return c, nil
}
