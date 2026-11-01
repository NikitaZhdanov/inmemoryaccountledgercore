package replay_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"inmemoryaccountledgercore/internal/domain"
	"inmemoryaccountledgercore/internal/ledger"
	"inmemoryaccountledgercore/internal/repository/memory"
)

// Literal inputs and expected balances come from TASK E1/E2/E4 and NUMBERS.
// This composes real components without claiming to run the full event stream.
func TestOrdinaryPostings(t *testing.T) {
	aed, err := domain.NewCurrency("AED", 2)
	require.NoError(t, err)
	bhd, err := domain.NewCurrency("BHD", 3)
	require.NoError(t, err)
	aedZero, err := domain.NewMoney(aed, 0)
	require.NoError(t, err)
	bhdZero, err := domain.NewMoney(bhd, 0)
	require.NoError(t, err)
	aedFee, err := domain.NewMoney(aed, 2500)
	require.NoError(t, err)
	repository := memory.New()
	var storage ledger.Repository = repository
	core, err := ledger.New(storage, []domain.Account{
		{ID: "ACC-001", Currency: aed, OpeningBalance: aedZero},
		{ID: "ACC-002", Currency: bhd, OpeningBalance: bhdZero},
	}, map[domain.Currency]domain.Money{aed: aedFee, bhd: bhdZero})
	require.NoError(t, err)
	for _, tc := range []struct {
		input domain.InputEvent
		want  int64
	}{
		{domain.InputEvent{ID: "E1", Kind: domain.InputCredit, AccountID: "ACC-001", Currency: aed, RawAmount: "1200.00", BookingDay: 1, ValueDay: 1, Order: 1}, 120000},
		{domain.InputEvent{ID: "E2", Kind: domain.InputDebit, AccountID: "ACC-001", Currency: aed, RawAmount: "950.00", BookingDay: 1, ValueDay: 1, Order: 2}, 25000},
		{domain.InputEvent{ID: "E4", Kind: domain.InputCredit, AccountID: "ACC-001", Currency: aed, RawAmount: "400.00", BookingDay: 3, ValueDay: 3, Order: 4}, 65000},
	} {
		start := len(repository.History().Records)
		outcome, err := core.Submit(tc.input)
		require.NoError(t, err)
		assert.Equal(t, domain.OutcomeAccepted, outcome.Status)
		history := repository.History()
		assert.Equal(t, tc.input, history.Records[start].Attempt.Input)
		projection, err := ledger.Rebuild(history, len(history.Records))
		require.NoError(t, err)
		balance, err := projection.Balance("ACC-001", tc.input.ValueDay, ledger.BalanceOptions{})
		require.NoError(t, err)
		assert.Equal(t, tc.want, balance.MinorUnits())
		balance, err = projection.Balance("ACC-002", tc.input.ValueDay, ledger.BalanceOptions{})
		require.NoError(t, err)
		assert.Equal(t, bhdZero, balance)
		before := repository.History()
		duplicate, err := core.Submit(tc.input)
		require.NoError(t, err)
		assert.Equal(t, domain.OutcomeRejected, duplicate.Status)
		after := repository.History()
		require.Len(t, after.Records, len(before.Records)+3)
		assert.Equal(t, before.Records, after.Records[:len(before.Records)])
		assert.Equal(t, ledger.DiagnosticDuplicateID, after.Records[len(after.Records)-1].Diagnostic.Code)
	}
}
