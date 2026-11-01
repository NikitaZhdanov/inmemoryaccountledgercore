package replay_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"inmemoryaccountledgercore/internal/domain"
	"inmemoryaccountledgercore/internal/ledger"
	"inmemoryaccountledgercore/internal/repository/memory"
)

// This posting-layer integration omits E8 and all closes. Fee reconciliation and
// full replay remain P08/P09/P11; amounts and dates are literal TASK fixtures.
func TestLinkedReversalAndLateInstalments(t *testing.T) {
	aed, err := domain.NewCurrency("AED", 2)
	require.NoError(t, err)
	bhd, err := domain.NewCurrency("BHD", 3)
	require.NoError(t, err)
	aedZero, err := domain.NewMoney(aed, 0)
	require.NoError(t, err)
	bhdZero, err := domain.NewMoney(bhd, 0)
	require.NoError(t, err)
	fee, err := domain.NewMoney(aed, 2500)
	require.NoError(t, err)
	repository := memory.New()
	core, err := ledger.New(repository, []domain.Account{
		{ID: "ACC-001", Currency: aed, OpeningBalance: aedZero},
		{ID: "ACC-002", Currency: bhd, OpeningBalance: bhdZero},
	}, map[domain.Currency]domain.Money{aed: fee, bhd: bhdZero})
	require.NoError(t, err)
	inputs := []domain.InputEvent{
		{ID: "E1", Kind: domain.InputCredit, AccountID: "ACC-001", Currency: aed, RawAmount: "1200.00", BookingDay: 1, ValueDay: 1, Order: 1},
		{ID: "E2", Kind: domain.InputDebit, AccountID: "ACC-001", Currency: aed, RawAmount: "950.00", BookingDay: 1, ValueDay: 1, Order: 2},
		{ID: "E3", Kind: domain.InputAuthorization, AccountID: "ACC-001", Currency: aed, RawAmount: "200.00", BookingDay: 2, ValueDay: 2, Order: 3, AuthorizationID: "Auth-A"},
		{ID: "E4", Kind: domain.InputCredit, AccountID: "ACC-001", Currency: aed, RawAmount: "400.00", BookingDay: 3, ValueDay: 3, Order: 4},
		{ID: "E5", Kind: domain.InputSettlement, AccountID: "ACC-001", Currency: aed, RawAmount: "185.00", BookingDay: 4, ValueDay: 4, Order: 5, AuthorizationID: "Auth-A"},
		{ID: "E6", Kind: domain.InputSettlement, AccountID: "ACC-001", Currency: aed, RawAmount: "180.00", BookingDay: 4, ValueDay: 4, Order: 6, AuthorizationID: "Auth-Z"},
		{ID: "E7", Kind: domain.InputDebit, AccountID: "ACC-001", Currency: aed, RawAmount: "620.00", BookingDay: 5, ValueDay: 2, Order: 7},
		{ID: "E9", Kind: domain.InputReversal, AccountID: "ACC-001", Currency: aed, BookingDay: 6, ValueDay: 2, Order: 9, ReversesEventID: "E7"},
		{ID: "E10", Kind: domain.InputCredit, AccountID: "ACC-002", Currency: bhd, RawAmount: "10.000", BookingDay: 5, ValueDay: 5, Order: 10, InstalmentCount: 3},
	}
	var original domain.Record
	var reversalAttempt domain.AttemptID
	for _, input := range inputs {
		before := repository.History()
		outcome, err := core.Submit(input)
		require.NoError(t, err)
		wantStatus := domain.OutcomeAccepted
		if input.ID == "E6" {
			wantStatus = domain.OutcomeRejected
		}
		assert.Equal(t, wantStatus, outcome.Status)
		history := repository.History()
		assert.Equal(t, before.Records, history.Records[:len(before.Records)])
		assert.Equal(t, input, history.Records[len(before.Records)].Attempt.Input)
		if input.ID == "E7" {
			original = history.Records[len(history.Records)-1]
			require.NotNil(t, original.Entry)
			assert.Equal(t, int64(-62000), original.Entry.Amount.MinorUnits())
		}
		if input.ID == "E9" {
			reversalAttempt = outcome.AttemptID
			reversal := history.Records[len(history.Records)-1].Entry
			require.NotNil(t, reversal)
			assert.Equal(t, original.Key, reversal.OriginalEntryKey)
			assert.Equal(t, int64(62000), reversal.Amount.MinorUnits())
			assert.Equal(t, domain.Day(6), reversal.BookingDay)
			assert.Equal(t, domain.Day(2), reversal.ValueDay)
		}
		if input.ID == "E10" {
			records := history.Records[len(before.Records):]
			require.Len(t, records, 5)
			for i, want := range []int64{3333, 3333, 3334} {
				entry := records[i+2].Entry
				require.NotNil(t, entry)
				assert.Equal(t, want, entry.Amount.MinorUnits())
				assert.Equal(t, bhd, entry.Amount.Currency())
				assert.Equal(t, domain.Day(5), entry.BookingDay)
				assert.Equal(t, domain.Day(5), entry.ValueDay)
				assert.Equal(t, i+1, entry.Instalment)
				assert.Equal(t, 3, entry.InstalmentCount)
			}
		}
	}
	history := repository.History()
	projection, err := ledger.Rebuild(history, len(history.Records))
	require.NoError(t, err)
	for _, tc := range []struct {
		id   domain.AccountID
		day  domain.Day
		want int64
	}{
		{"ACC-001", 1, 25000}, {"ACC-001", 2, 25000}, {"ACC-001", 3, 65000},
		{"ACC-001", 4, 46500}, {"ACC-001", 5, 46500}, {"ACC-001", 6, 46500},
		{"ACC-002", 4, 0}, {"ACC-002", 5, 10000}, {"ACC-002", 6, 10000},
	} {
		balance, err := projection.Balance(tc.id, tc.day, ledger.BalanceOptions{})
		require.NoError(t, err)
		assert.Equal(t, tc.want, balance.MinorUnits())
	}
	for _, input := range inputs[len(inputs)-2:] {
		outcome, err := core.Submit(input)
		require.NoError(t, err)
		assert.Equal(t, domain.OutcomeRejected, outcome.Status)
	}
	after := repository.History()
	require.Len(t, after.Records, len(history.Records)+6)
	assert.Equal(t, history.Records, after.Records[:len(history.Records)])
	secondReversal := inputs[len(inputs)-2]
	secondReversal.ID = "second-reversal"
	rejected, err := core.Submit(secondReversal)
	require.NoError(t, err)
	assert.Equal(t, domain.OutcomeRejected, rejected.Status)
	assert.NotEqual(t, reversalAttempt, rejected.AttemptID)
	recorded := repository.History()
	require.Len(t, recorded.Records, len(after.Records)+3)
	assert.Equal(t, after.Records, recorded.Records[:len(after.Records)])
	diagnostic := recorded.Records[len(recorded.Records)-1]
	require.NotNil(t, diagnostic.Diagnostic)
	assert.Equal(t, rejected.DiagnosticKey, diagnostic.Key)
	assert.Equal(t, ledger.DiagnosticAlreadyReversed, diagnostic.Diagnostic.Code)
	projection, err = ledger.Rebuild(recorded, len(recorded.Records))
	require.NoError(t, err)
	balance, err := projection.Balance("ACC-001", 6, ledger.BalanceOptions{})
	require.NoError(t, err)
	assert.Equal(t, int64(46500), balance.MinorUnits())
}
