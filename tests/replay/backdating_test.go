package replay_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"inmemoryaccountledgercore/internal/domain"
	"inmemoryaccountledgercore/internal/ledger"
	"inmemoryaccountledgercore/internal/posting"
	"inmemoryaccountledgercore/internal/projection"
	"inmemoryaccountledgercore/internal/repository/memory"
)

// P09 stops before capitalization. Literal TASK inputs and NUMBERS checkpoint
// oracles exercise submission-triggered corrections with real components.
func TestBackdatedSubmissionsReconcileBeforeNextInput(t *testing.T) {
	aed, err := domain.NewCurrency("AED", 2)
	require.NoError(t, err)
	bhd, err := domain.NewCurrency("BHD", 3)
	require.NoError(t, err)
	aedZero, err := domain.NewMoney(aed, 0)
	require.NoError(t, err)
	bhdZero, err := domain.NewMoney(bhd, 0)
	require.NoError(t, err)
	charge, err := domain.NewMoney(aed, 2500)
	require.NoError(t, err)
	repository := memory.New()
	core, err := ledger.New(repository, []domain.Account{
		{ID: "ACC-001", Currency: aed, OpeningBalance: aedZero},
		{ID: "ACC-002", Currency: bhd, OpeningBalance: bhdZero},
	}, map[domain.Currency]domain.Money{aed: charge, bhd: bhdZero})
	require.NoError(t, err)
	inputs := []domain.InputEvent{
		{ID: "E1", Kind: domain.InputCredit, AccountID: "ACC-001", Currency: aed, RawAmount: "1200.00", BookingDay: 1, ValueDay: 1, Order: 1},
		{ID: "E2", Kind: domain.InputDebit, AccountID: "ACC-001", Currency: aed, RawAmount: "950.00", BookingDay: 1, ValueDay: 1, Order: 2},
		{ID: "E3", Kind: domain.InputAuthorization, AccountID: "ACC-001", Currency: aed, RawAmount: "200.00", BookingDay: 2, ValueDay: 2, Order: 3, AuthorizationID: "Auth-A"},
		{ID: "E4", Kind: domain.InputCredit, AccountID: "ACC-001", Currency: aed, RawAmount: "400.00", BookingDay: 3, ValueDay: 3, Order: 4},
		{ID: "E5", Kind: domain.InputSettlement, AccountID: "ACC-001", Currency: aed, RawAmount: "185.00", BookingDay: 4, ValueDay: 4, Order: 5, AuthorizationID: "Auth-A"},
		{ID: "E6", Kind: domain.InputSettlement, AccountID: "ACC-001", Currency: aed, RawAmount: "180.00", BookingDay: 4, ValueDay: 4, Order: 6, AuthorizationID: "Auth-Z"},
		{ID: "E7", Kind: domain.InputDebit, AccountID: "ACC-001", Currency: aed, RawAmount: "620.00", BookingDay: 5, ValueDay: 2, Order: 7},
		{ID: "E8", Kind: domain.InputAuthorization, AccountID: "ACC-001", Currency: aed, RawAmount: "90.00", BookingDay: 5, ValueDay: 5, Order: 8, AuthorizationID: "Auth-B"},
		{ID: "E9", Kind: domain.InputReversal, AccountID: "ACC-001", Currency: aed, BookingDay: 6, ValueDay: 2, Order: 9, ReversesEventID: "E7"},
		{ID: "E10", Kind: domain.InputCredit, AccountID: "ACC-002", Currency: bhd, RawAmount: "10.000", BookingDay: 5, ValueDay: 5, Order: 10, InstalmentCount: 3},
	}
	submit := func(input domain.InputEvent, status domain.OutcomeStatus) domain.InputOutcome {
		t.Helper()
		before := repository.History()
		outcome, err := core.Submit(input)
		require.NoError(t, err)
		require.Equal(t, status, outcome.Status)
		after := repository.History()
		assert.Equal(t, before.Records, after.Records[:len(before.Records)])
		require.NotNil(t, after.Records[len(before.Records)].Attempt)
		assert.Equal(t, input, after.Records[len(before.Records)].Attempt.Input)
		return outcome
	}
	closeDay := func(day domain.Day) {
		t.Helper()
		for _, id := range []domain.AccountID{"ACC-001", "ACC-002"} {
			require.NoError(t, core.CloseDay(id, day, day, domain.RecordKey(fmt.Sprintf("checkpoint/%d", day))))
		}
	}
	for i, input := range inputs[:6] {
		status := domain.OutcomeAccepted
		if input.ID == "E6" {
			status = domain.OutcomeRejected
		}
		submit(input, status)
		if day, exists := map[int]domain.Day{1: 1, 2: 2, 3: 3, 5: 4}[i]; exists {
			closeDay(day)
		}
	}
	requireReplayCheckpoint(t, repository.History(), "ACC-001", []int64{25000, 25000, 65000, 46500}, []int64{10, 10, 26, 19})
	e7 := submit(inputs[6], domain.OutcomeAccepted)
	beforeE8 := repository.History()
	requireReplayCheckpoint(t, beforeE8, "ACC-001", []int64{25000, -39500, 500, -20500, -20500}, []int64{10, 0, 0, 0})
	state := replayProjection(t, beforeE8)
	_, closed := state.Accrual("ACC-001", 5)
	assert.False(t, closed, "E7 must leave Day 5 open")
	fees := replayFeeRecords(beforeE8)
	require.Len(t, fees, 2)
	assert.Equal(t, []domain.Day{2, 4}, []domain.Day{fees[0].Entry.ValueDay, fees[1].Entry.ValueDay})
	for _, record := range fees {
		assert.Equal(t, domain.RecordKey(e7.AttemptID), record.Entry.CauseKey)
	}
	available, err := state.AvailableBalance("ACC-001", 5)
	require.NoError(t, err)
	assert.Equal(t, int64(-20500), available.MinorUnits())
	hold, err := domain.ParseMoney(aed, "90.00")
	require.NoError(t, err)
	proposed, err := available.Sub(hold)
	require.NoError(t, err)
	assert.Equal(t, int64(-29500), proposed.MinorUnits())
	e8 := submit(inputs[7], domain.OutcomeDeclined)
	afterE8 := repository.History()
	state = replayProjection(t, afterE8)
	decline, exists := state.Authorization("ACC-001", "Auth-B")
	require.True(t, exists)
	assert.Equal(t, domain.AuthorizationDeclined, decline.Transition.State)
	assert.Equal(t, e8.AttemptID, decline.Transition.AttemptID)
	holds, err := state.ActiveHolds("ACC-001")
	require.NoError(t, err)
	assert.Zero(t, holds.MinorUnits())
	assert.Equal(t, fees, replayFeeRecords(afterE8), "E8 must not close Day 5")
	closeDay(5)
	day5 := repository.History()
	requireReplayCheckpoint(t, day5, "ACC-001", []int64{25000, -39500, 500, -20500, -23000}, []int64{10, 0, 0, 0, 0})
	state = replayProjection(t, day5)
	feeFree, err := state.Balance("ACC-001", 2, projection.BalanceOptions{ExcludeAllFees: true})
	require.NoError(t, err)
	assert.Equal(t, int64(-37000), feeFree.MinorUnits())
	assessments := replayFeeRecords(day5)
	require.Len(t, assessments, 3)
	for i, day := range []domain.Day{2, 4, 5} {
		entry := assessments[i].Entry
		assert.Equal(t, domain.EntryFeeAssessment, entry.Kind)
		assert.Equal(t, domain.Day(5), entry.BookingDay)
		assert.Equal(t, day, entry.ValueDay)
		assert.Equal(t, int64(-2500), entry.Amount.MinorUnits())
	}

	e9 := submit(inputs[8], domain.OutcomeAccepted)
	afterE9 := repository.History()
	requireReplayCheckpoint(t, afterE9, "ACC-001", []int64{25000, 25000, 65000, 46500, 46500, 46500}, []int64{10, 10, 26, 19, 19})
	state = replayProjection(t, afterE9)
	authB, exists := state.Authorization("ACC-001", "Auth-B")
	require.True(t, exists)
	assert.Equal(t, decline, authB, "financial restoration preserves the original decline")
	allFees := replayFeeRecords(afterE9)
	require.Len(t, allFees, 6)
	for i, original := range assessments {
		refund := allFees[i+3].Entry
		assert.Equal(t, domain.EntryFeeRefund, refund.Kind)
		assert.Equal(t, original.Key, refund.OriginalEntryKey)
		assert.Equal(t, original.Entry.FeeObligationID, refund.FeeObligationID)
		assert.Equal(t, original.Entry.ValueDay, refund.ValueDay)
		assert.Equal(t, domain.Day(6), refund.BookingDay)
		assert.Equal(t, int64(2500), refund.Amount.MinorUnits())
		assert.Equal(t, domain.RecordKey(e9.AttemptID), refund.CauseKey)
	}
	requireReplayCheckpoint(t, afterE9, "ACC-002", []int64{0, 0, 0, 0, 0}, []int64{0, 0, 0, 0, 0})
	submit(inputs[9], domain.OutcomeAccepted)
	afterE10 := repository.History()
	requireReplayCheckpoint(t, afterE10, "ACC-002", []int64{0, 0, 0, 0, 10000, 10000}, []int64{0, 0, 0, 0, 4})
	requireReplayCheckpoint(t, afterE10, "ACC-001", []int64{25000, 25000, 65000, 46500, 46500, 46500}, []int64{10, 10, 26, 19, 19})
	assert.Equal(t, allFees, replayFeeRecords(afterE10))
	state = replayProjection(t, afterE10)
	authA, exists := state.Authorization("ACC-001", "Auth-A")
	require.True(t, exists)
	assert.Equal(t, domain.AuthorizationSettled, authA.Transition.State)
	holds, err = state.ActiveHolds("ACC-001")
	require.NoError(t, err)
	assert.Zero(t, holds.MinorUnits())
	for _, record := range afterE10.Records {
		if record.Entry != nil {
			assert.NotEqual(t, domain.EntryCapitalization, record.Entry.Kind)
		}
	}
	// The saved prefix retains Day 5's observation after restoration.
	requireReplayCheckpoint(t, day5, "ACC-001", []int64{25000, -39500, 500, -20500, -23000}, []int64{10, 0, 0, 0, 0})
	for range 2 {
		require.NoError(t, core.ReconcileClosedDays("ACC-001", 2, 6, domain.RecordKey(e9.AttemptID)))
		require.NoError(t, core.ReconcileClosedDays("ACC-002", 5, 5, "repeat/E10"))
		closeDay(5)
	}
	assert.Equal(t, afterE10, repository.History(), "unchanged internal work appends nothing")
	for _, input := range inputs {
		before := repository.History()
		outcome := submit(input, domain.OutcomeRejected)
		after := repository.History()
		require.Len(t, after.Records, len(before.Records)+3)
		diagnostic := after.Records[len(after.Records)-1]
		require.NotNil(t, diagnostic.Diagnostic)
		assert.Equal(t, outcome.DiagnosticKey, diagnostic.Key)
		assert.Equal(t, posting.DiagnosticDuplicateID, diagnostic.Diagnostic.Code)
	}
	assert.Equal(t, allFees, replayFeeRecords(repository.History()))
}

func replayProjection(t *testing.T, history domain.History) *projection.Projection {
	t.Helper()
	state, err := projection.Rebuild(history, len(history.Records))
	require.NoError(t, err)
	return state
}

func requireReplayCheckpoint(t *testing.T, history domain.History, id domain.AccountID, balances, accruals []int64) {
	t.Helper()
	state := replayProjection(t, history)
	for i, want := range balances {
		balance, err := state.Balance(id, domain.Day(i+1), projection.BalanceOptions{})
		require.NoError(t, err)
		assert.Equal(t, want, balance.MinorUnits(), "%s Day %d balance", id, i+1)
	}
	unpaid, err := state.UnpaidAccruals(id)
	require.NoError(t, err)
	require.Len(t, unpaid, len(accruals))
	for i, want := range accruals {
		assert.Equal(t, domain.Day(i+1), unpaid[i].Day)
		assert.Equal(t, want, unpaid[i].Amount.MinorUnits(), "%s Day %d accrual", id, i+1)
	}
}

func replayFeeRecords(history domain.History) []domain.Record {
	var fees []domain.Record
	for _, record := range history.Records {
		if entry := record.Entry; entry != nil && (entry.Kind == domain.EntryFeeAssessment || entry.Kind == domain.EntryFeeRefund || entry.Kind == domain.EntryFeeAdjustment) {
			fees = append(fees, record)
		}
	}
	return fees
}
