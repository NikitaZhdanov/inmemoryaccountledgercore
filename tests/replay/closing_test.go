package replay_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"inmemoryaccountledgercore/internal/domain"
	"inmemoryaccountledgercore/internal/ledger"
	"inmemoryaccountledgercore/internal/projection"
	"inmemoryaccountledgercore/internal/repository/memory"
)

// Account isolation covers explicit closing and submission-triggered refunds.
func TestClosingObligationsRemainIndependentAcrossAccounts(t *testing.T) {
	currency, err := domain.NewCurrency("AED", 2)
	require.NoError(t, err)
	opening, err := domain.NewMoney(currency, -1)
	require.NoError(t, err)
	charge, err := domain.NewMoney(currency, 2500)
	require.NoError(t, err)
	repository := memory.New()
	core, err := ledger.New(repository, []domain.Account{
		{ID: "first", Currency: currency, OpeningBalance: opening},
		{ID: "second", Currency: currency, OpeningBalance: opening},
	}, map[domain.Currency]domain.Money{currency: charge})
	require.NoError(t, err)
	originals := make(map[domain.AccountID]domain.Record)
	for _, id := range []domain.AccountID{"first", "second"} {
		require.NoError(t, core.CloseDay(id, 2, 2, "checkpoint/2"))
		history := repository.History()
		for _, record := range history.Records {
			if record.Entry != nil && record.Entry.AccountID == id {
				originals[id] = record
			}
		}
	}
	require.Len(t, originals, 2)
	assert.NotEqual(t, originals["first"].Entry.FeeObligationID, originals["second"].Entry.FeeObligationID)
	before := repository.History()
	for _, id := range []domain.AccountID{"first", "second"} {
		require.NoError(t, core.CloseDay(id, 2, 2, "checkpoint/2"))
		require.NoError(t, core.ReconcileClosedDays(id, 1, 3, "repeat"))
	}
	assert.Equal(t, before, repository.History())

	for _, id := range []domain.AccountID{"first", "second"} {
		outcome, err := core.Submit(domain.InputEvent{
			ID: domain.EventID("restore/" + string(id)), Kind: domain.InputCredit,
			AccountID: id, Currency: currency, RawAmount: "0.01", BookingDay: 3, ValueDay: 2,
		})
		require.NoError(t, err)
		require.Equal(t, domain.OutcomeAccepted, outcome.Status)
		// Submission must refund this account before returning. Repeating
		// the same reconciliation performs no additional append.
		corrected := repository.History()
		require.NoError(t, core.ReconcileClosedDays(id, 2, 3, "restore"))
		assert.Equal(t, corrected, repository.History())
	}
	history := repository.History()
	state, err := projection.Rebuild(history, len(history.Records))
	require.NoError(t, err)
	refunds := 0
	for _, record := range history.Records {
		if record.Entry != nil && record.Entry.Kind == domain.EntryFeeRefund {
			refunds++
			original := originals[record.Entry.AccountID]
			assert.Equal(t, original.Key, record.Entry.OriginalEntryKey)
			assert.Equal(t, original.Entry.FeeObligationID, record.Entry.FeeObligationID)
			assert.Equal(t, int64(2500), record.Entry.Amount.MinorUnits())
		}
	}
	assert.Equal(t, 2, refunds)
	for _, id := range []domain.AccountID{"first", "second"} {
		balance, err := state.Balance(id, 2, projection.BalanceOptions{})
		require.NoError(t, err)
		assert.Zero(t, balance.MinorUnits())
		require.NoError(t, core.ReconcileClosedDays(id, 2, 3, "restore"))
	}
	assert.Equal(t, history, repository.History())
}
