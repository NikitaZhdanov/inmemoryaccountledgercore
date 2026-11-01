package ledger

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"inmemoryaccountledgercore/internal/domain"
)

// journal is a detached, atomic test double. Integration with real memory
// storage is exercised separately in tests/replay.
type journal struct {
	records     []domain.Record
	appendError error
	appends     int
}

func (r *journal) History() domain.History {
	history := domain.History{Records: make([]domain.Record, len(r.records))}
	for i, record := range r.records {
		history.Records[i] = record.Clone()
	}
	return history
}

func (r *journal) Append(batch domain.Batch) error {
	r.appends++
	if r.appendError != nil {
		return r.appendError
	}
	for _, record := range batch.Records {
		r.records = append(r.records, record.Clone())
	}
	return nil
}

func fixtureCurrency(t *testing.T, code string, scale int) domain.Currency {
	t.Helper()
	currency, err := domain.NewCurrency(code, scale)
	require.NoError(t, err)
	return currency
}

func fixtureMoney(t *testing.T, currency domain.Currency, minor int64) domain.Money {
	t.Helper()
	amount, err := domain.NewMoney(currency, minor)
	require.NoError(t, err)
	return amount
}

func TestConfigurationValidationBeforeAppend(t *testing.T) {
	aed := fixtureCurrency(t, "AED", 2)
	bhd := fixtureCurrency(t, "BHD", 3)
	aedOtherScale := fixtureCurrency(t, "AED", 3)
	account := domain.Account{ID: "ACC-001", Currency: aed, OpeningBalance: fixtureMoney(t, aed, 0)}
	validPolicy := map[domain.Currency]domain.Money{aed: fixtureMoney(t, aed, 2500)}
	for _, tc := range []struct {
		name     string
		accounts []domain.Account
		policies map[domain.Currency]domain.Money
	}{
		{"missing policy", []domain.Account{account}, nil},
		{"another currency is not coverage", []domain.Account{account}, map[domain.Currency]domain.Money{bhd: fixtureMoney(t, bhd, 0)}},
		{"invalid account descriptor", []domain.Account{{ID: "A", OpeningBalance: account.OpeningBalance}}, validPolicy},
		{"invalid policy descriptor", []domain.Account{account}, map[domain.Currency]domain.Money{aed: validPolicy[aed], {}: account.OpeningBalance}},
		{"conflicting account scales", []domain.Account{account, {ID: "B", Currency: aedOtherScale, OpeningBalance: fixtureMoney(t, aedOtherScale, 0)}}, map[domain.Currency]domain.Money{aed: validPolicy[aed], aedOtherScale: fixtureMoney(t, aedOtherScale, 0)}},
		{"conflicting unused policy scale", []domain.Account{account}, map[domain.Currency]domain.Money{aed: validPolicy[aed], aedOtherScale: fixtureMoney(t, aedOtherScale, 0)}},
		{"negative charge", []domain.Account{account}, map[domain.Currency]domain.Money{aed: fixtureMoney(t, aed, -1)}},
		{"charge currency mismatch", []domain.Account{account}, map[domain.Currency]domain.Money{aed: fixtureMoney(t, bhd, 0)}},
		{"charge scale mismatch", []domain.Account{account}, map[domain.Currency]domain.Money{aed: fixtureMoney(t, aedOtherScale, 0)}},
		{"invalid charge money", []domain.Account{account}, map[domain.Currency]domain.Money{aed: {}}},
		{"invalid opening money", []domain.Account{{ID: "A", Currency: aed}}, validPolicy},
		{"opening currency mismatch", []domain.Account{{ID: "A", Currency: aed, OpeningBalance: fixtureMoney(t, bhd, 0)}}, validPolicy},
		{"duplicate account ID", []domain.Account{account, account}, validPolicy},
		{"empty account ID", []domain.Account{{Currency: aed, OpeningBalance: account.OpeningBalance}}, validPolicy},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repository := &journal{}
			core, err := New(repository, tc.accounts, tc.policies)
			require.ErrorIs(t, err, ErrInvalidConfiguration)
			assert.Nil(t, core)
			assert.Zero(t, repository.appends, "validation completes before any append")
			assert.Empty(t, repository.records)

			core, err = New(repository, []domain.Account{account}, map[domain.Currency]domain.Money{aed: fixtureMoney(t, aed, 0)})
			require.NoError(t, err)
			require.NotNil(t, core)
			require.Len(t, repository.records, 1)
			assert.Equal(t, account, *repository.records[0].Account)
			assert.Equal(t, fixtureMoney(t, aed, 0), core.charges[aed])
		})
	}
}

func TestConfigurationOwnsContainers(t *testing.T) {
	aed := fixtureCurrency(t, "AED", 2)
	bhd := fixtureCurrency(t, "BHD", 3)
	accounts := []domain.Account{
		{ID: "ACC-001", Currency: aed, OpeningBalance: fixtureMoney(t, aed, 0)},
		{ID: "ACC-002", Currency: bhd, OpeningBalance: fixtureMoney(t, bhd, 0)},
	}
	charges := map[domain.Currency]domain.Money{aed: fixtureMoney(t, aed, 2500), bhd: fixtureMoney(t, bhd, 0)}
	repository := &journal{}
	core, err := New(repository, accounts, charges)
	require.NoError(t, err)
	before := repository.History()
	accounts[0] = domain.Account{}
	delete(charges, aed)
	charges[bhd] = fixtureMoney(t, bhd, 1)
	assert.Equal(t, *before.Records[0].Account, core.accounts["ACC-001"])
	assert.Equal(t, fixtureMoney(t, aed, 2500), core.charges[aed])
	assert.Equal(t, fixtureMoney(t, bhd, 0), core.charges[bhd])
	assert.Equal(t, before, repository.History())
}

func TestConfigurationAcceptsCustomCurrencies(t *testing.T) {
	for _, scale := range []int{0, 4, 18} {
		currency := fixtureCurrency(t, "TOK", scale)
		equalCurrency := fixtureCurrency(t, "TOK", scale)
		zero := fixtureMoney(t, equalCurrency, 0)
		core, err := New(&journal{}, []domain.Account{{ID: "A", Currency: currency, OpeningBalance: zero}}, map[domain.Currency]domain.Money{equalCurrency: zero})
		require.NoError(t, err)
		assert.Equal(t, zero, core.charges[currency])
	}
}

func TestInitializationTechnicalFailure(t *testing.T) {
	core, err := New(nil, nil, nil)
	require.ErrorIs(t, err, ErrInvalidConfiguration)
	assert.Nil(t, core)

	currency := fixtureCurrency(t, "AED", 2)
	zero := fixtureMoney(t, currency, 0)
	failure := errors.New("repository unavailable")
	repository := &journal{appendError: failure}
	core, err = New(repository, []domain.Account{{ID: "A", Currency: currency, OpeningBalance: zero}}, map[domain.Currency]domain.Money{currency: zero})
	require.ErrorIs(t, err, failure)
	require.NotErrorIs(t, err, ErrInvalidConfiguration)
	assert.Nil(t, core)
	assert.Empty(t, repository.records)
}
