package replay_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"inmemoryaccountledgercore/internal/domain"
	"inmemoryaccountledgercore/internal/ledger"
	"inmemoryaccountledgercore/internal/posting"
	"inmemoryaccountledgercore/internal/projection"
	"inmemoryaccountledgercore/internal/repository/memory"
	"inmemoryaccountledgercore/internal/scenario"
)

func TestCanonicalScenario(t *testing.T) {
	repository, core, primary, instalment := canonicalComponents(t)
	require.NoError(t, scenario.Run(core, primary, instalment))
	history := repository.History()

	// Observe committed record prefixes, independently of the runner's schedule.
	before := func(event domain.EventID) domain.History {
		t.Helper()
		for i, record := range history.Records {
			if record.Attempt != nil && record.Attempt.Input.ID == event {
				return domain.History{Records: history.Records[:i]}
			}
		}
		t.Fatalf("missing event %s", event)
		return domain.History{}
	}
	assertAvailability := func(event domain.EventID, day domain.Day, balance, available int64) {
		t.Helper()
		state := replayProjection(t, before(event))
		posted, err := state.Balance(primary.ID, day, projection.BalanceOptions{})
		require.NoError(t, err)
		assert.Equal(t, balance, posted.MinorUnits())
		spendable, err := state.AvailableBalance(primary.ID, day)
		require.NoError(t, err)
		assert.Equal(t, available, spendable.MinorUnits())
	}
	assertAvailability("E4", 2, 25000, 5000)
	assertAvailability("E5", 3, 65000, 45000)
	assertAvailability("E6", 4, 46500, 46500)
	assertAvailability("E7", 4, 46500, 46500)
	assertAvailability("E8", 5, -20500, -20500)
	requireReplayCheckpoint(t, before("E8"), primary.ID, []int64{25000, -39500, 500, -20500, -20500}, []int64{10, 0, 0, 0})
	day5 := before("E9")
	requireReplayCheckpoint(t, day5, primary.ID, []int64{25000, -39500, 500, -20500, -23000}, []int64{10, 0, 0, 0, 0})
	feeFree, err := replayProjection(t, day5).Balance(primary.ID, 2, projection.BalanceOptions{ExcludeAllFees: true})
	require.NoError(t, err)
	assert.Equal(t, int64(-37000), feeFree.MinorUnits())
	afterE9 := before("E10")
	requireReplayCheckpoint(t, afterE9, primary.ID, []int64{25000, 25000, 65000, 46500, 46500, 46500}, []int64{10, 10, 26, 19, 19})
	requireReplayCheckpoint(t, afterE9, instalment.ID, []int64{0, 0, 0, 0, 0}, []int64{0, 0, 0, 0, 0})
	for i, record := range history.Records {
		if record.Close != nil && record.Close.Day == 6 {
			preFinalization := domain.History{Records: history.Records[:i]}
			requireReplayCheckpoint(t, preFinalization, instalment.ID, []int64{0, 0, 0, 0, 10000, 10000}, []int64{0, 0, 0, 0, 4})
			break
		}
	}

	// Explicit close positions ensure E10's earlier booking day never closes a
	// day implicitly; both accounts finish only after all ten supplied inputs.
	var latest domain.EventID
	var eventIDs []domain.EventID
	closes := make(map[domain.AccountID][]domain.Day)
	wantAfter := map[domain.Day]domain.EventID{1: "E2", 2: "E3", 3: "E4", 4: "E6", 5: "E8", 6: "E10"}
	var instalments []int64
	var originalDebit domain.Record
	var reversal domain.Record
	var diagnostics []domain.Diagnostic
	for _, record := range history.Records {
		if record.Attempt != nil {
			latest = record.Attempt.Input.ID
			eventIDs = append(eventIDs, latest)
		}
		if record.Outcome != nil {
			want := domain.OutcomeAccepted
			if latest == "E6" {
				want = domain.OutcomeRejected
			} else if latest == "E8" {
				want = domain.OutcomeDeclined
			}
			assert.Equal(t, want, record.Outcome.Status, "%s outcome", latest)
		}
		if record.Close != nil {
			assert.Equal(t, wantAfter[record.Close.Day], latest)
			closes[record.Close.AccountID] = append(closes[record.Close.AccountID], record.Close.Day)
		}
		if record.Diagnostic != nil {
			diagnostics = append(diagnostics, *record.Diagnostic)
		}
		if entry := record.Entry; entry != nil {
			switch entry.Kind {
			case domain.EntryInstalment:
				instalments = append(instalments, entry.Amount.MinorUnits())
				assert.Equal(t, instalment.ID, entry.AccountID)
				assert.Equal(t, domain.Day(5), entry.BookingDay)
				assert.Equal(t, domain.Day(5), entry.ValueDay)
				assert.Equal(t, len(instalments), entry.Instalment)
				assert.Equal(t, 3, entry.InstalmentCount)
			case domain.EntryDebit:
				if latest == "E7" {
					originalDebit = record
				}
			case domain.EntryReversal:
				reversal = record
			}
		}
	}
	assert.Equal(t, []domain.EventID{"E1", "E2", "E3", "E4", "E5", "E6", "E7", "E8", "E9", "E10"}, eventIDs)
	assert.Equal(t, []domain.Day{1, 2, 3, 4, 5, 6}, closes[primary.ID])
	assert.Equal(t, []domain.Day{1, 2, 3, 4, 5, 6}, closes[instalment.ID])
	assert.Equal(t, []int64{3333, 3333, 3334}, instalments)
	require.NotNil(t, originalDebit.Entry)
	require.NotNil(t, reversal.Entry)
	assert.Equal(t, originalDebit.Key, reversal.Entry.OriginalEntryKey)
	assert.Equal(t, int64(-62000), originalDebit.Entry.Amount.MinorUnits())
	assert.Equal(t, int64(62000), reversal.Entry.Amount.MinorUnits())
	assert.Equal(t, domain.Day(6), reversal.Entry.BookingDay)
	assert.Equal(t, domain.Day(2), reversal.Entry.ValueDay)
	require.Len(t, diagnostics, 2)
	assert.Equal(t, posting.DiagnosticUnknownAuthorization, diagnostics[0].Code)
	assert.Equal(t, domain.Day(4), diagnostics[0].ObservationDay)
	assert.Equal(t, posting.DiagnosticInsufficientAvailability, diagnostics[1].Code)
	assert.Equal(t, domain.Day(5), diagnostics[1].ObservationDay)

	fees := replayFeeRecords(history)
	require.Len(t, fees, 6)
	for i, day := range []domain.Day{2, 4, 5} {
		assessment, refund := fees[i], fees[i+3]
		assert.Equal(t, domain.EntryFeeAssessment, assessment.Entry.Kind)
		assert.Equal(t, int64(-2500), assessment.Entry.Amount.MinorUnits())
		assert.Equal(t, day, assessment.Entry.ValueDay)
		assert.Equal(t, domain.Day(5), assessment.Entry.BookingDay)
		assert.Equal(t, domain.EntryFeeRefund, refund.Entry.Kind)
		assert.Equal(t, int64(2500), refund.Entry.Amount.MinorUnits())
		assert.Equal(t, day, refund.Entry.ValueDay)
		assert.Equal(t, domain.Day(6), refund.Entry.BookingDay)
		assert.Equal(t, assessment.Key, refund.Entry.OriginalEntryKey)
		assert.Equal(t, assessment.Entry.FeeObligationID, refund.Entry.FeeObligationID)
	}
	state := replayProjection(t, history)
	for _, tc := range []struct {
		id                 domain.AccountID
		balances, accruals []int64
		capitalization     int64
	}{
		{primary.ID, []int64{25000, 25000, 65000, 46500, 46500, 46603}, []int64{10, 10, 26, 19, 19, 19}, 103},
		{instalment.ID, []int64{0, 0, 0, 0, 10000, 10008}, []int64{0, 0, 0, 0, 4, 4}, 8},
	} {
		var sum int64
		for i, want := range tc.balances {
			day := domain.Day(i + 1)
			balance, err := state.Balance(tc.id, day, projection.BalanceOptions{})
			require.NoError(t, err)
			assert.Equal(t, want, balance.MinorUnits())
			accrual, exists := state.Accrual(tc.id, day)
			require.True(t, exists)
			assert.Equal(t, tc.accruals[i], accrual.MinorUnits())
			sum += tc.accruals[i]
			fee, err := state.FeeNet(tc.id, day)
			require.NoError(t, err)
			assert.Zero(t, fee.MinorUnits())
		}
		assert.Equal(t, tc.capitalization, sum)
		var credits []domain.MonetaryEntry
		for _, record := range history.Records {
			if entry := record.Entry; entry != nil && entry.AccountID == tc.id && entry.Kind == domain.EntryCapitalization {
				credits = append(credits, *entry)
			}
		}
		require.Len(t, credits, 1)
		assert.Equal(t, tc.capitalization, credits[0].Amount.MinorUnits())
		unpaid, err := state.UnpaidAccruals(tc.id)
		require.NoError(t, err)
		assert.Empty(t, unpaid)
		holds, err := state.ActiveHolds(tc.id)
		require.NoError(t, err)
		assert.Zero(t, holds.MinorUnits())
	}
	authA, exists := state.Authorization(primary.ID, "Auth-A")
	require.True(t, exists)
	assert.Equal(t, domain.AuthorizationSettled, authA.Transition.State)
	decline, exists := replayProjection(t, day5).Authorization(primary.ID, "Auth-B")
	require.True(t, exists)
	assert.Equal(t, domain.AuthorizationDeclined, decline.Transition.State)
	authB, exists := state.Authorization(primary.ID, "Auth-B")
	require.True(t, exists)
	assert.Equal(t, decline, authB)

	other, otherCore, otherPrimary, otherInstalment := canonicalComponents(t)
	require.NoError(t, scenario.Run(otherCore, otherPrimary, otherInstalment))
	assert.Equal(t, history, other.History(), "fresh runs produce identical history")
	assertCanonicalStateEqual(t, state, replayProjection(t, other.History()))

	require.NoError(t, scenario.Run(core, primary, instalment))
	repeated := repository.History()
	assert.Equal(t, history.Records, repeated.Records[:len(history.Records)], "all originals survive")
	// Every duplicate contributes only attempt/outcome/diagnostic records.
	require.Len(t, repeated.Records, len(history.Records)+3*len(eventIDs))
	for _, record := range repeated.Records[len(history.Records):] {
		assert.Nil(t, record.Entry)
		assert.Nil(t, record.Authorization)
		assert.Nil(t, record.Close)
		if record.Outcome != nil {
			assert.Equal(t, domain.OutcomeRejected, record.Outcome.Status)
		}
		if record.Diagnostic != nil {
			assert.Equal(t, posting.DiagnosticDuplicateID, record.Diagnostic.Code)
		}
	}
	assertCanonicalStateEqual(t, state, replayProjection(t, repeated))
}

func assertCanonicalStateEqual(t *testing.T, want, got *projection.Projection) {
	t.Helper()
	// Compare observable state; projections also own their differing audit history.
	for _, id := range []domain.AccountID{"ACC-001", "ACC-002"} {
		for day := domain.Day(1); day <= 6; day++ {
			wantBalance, err := want.AvailableBalance(id, day)
			require.NoError(t, err)
			gotBalance, err := got.AvailableBalance(id, day)
			require.NoError(t, err)
			assert.Equal(t, wantBalance, gotBalance)
			wantBalance, err = want.Balance(id, day, projection.BalanceOptions{})
			require.NoError(t, err)
			gotBalance, err = got.Balance(id, day, projection.BalanceOptions{})
			require.NoError(t, err)
			assert.Equal(t, wantBalance, gotBalance)
			wantAccrual, wantExists := want.Accrual(id, day)
			gotAccrual, gotExists := got.Accrual(id, day)
			assert.Equal(t, wantExists, gotExists)
			assert.Equal(t, wantAccrual, gotAccrual)
			wantFee, err := want.FeeNet(id, day)
			require.NoError(t, err)
			gotFee, err := got.FeeNet(id, day)
			require.NoError(t, err)
			assert.Equal(t, wantFee, gotFee)
		}
		wantHolds, err := want.ActiveHolds(id)
		require.NoError(t, err)
		gotHolds, err := got.ActiveHolds(id)
		require.NoError(t, err)
		assert.Equal(t, wantHolds, gotHolds)
		for _, auth := range []domain.AuthorizationID{"Auth-A", "Auth-B"} {
			wantAuth, wantExists := want.Authorization(id, auth)
			gotAuth, gotExists := got.Authorization(id, auth)
			assert.Equal(t, wantExists, gotExists)
			assert.Equal(t, wantAuth, gotAuth)
		}
	}
}

func canonicalComponents(t *testing.T) (*memory.Repository, *ledger.Core, domain.Account, domain.Account) {
	t.Helper()
	aed, err := domain.NewCurrency("AED", 2)
	require.NoError(t, err)
	bhd, err := domain.NewCurrency("BHD", 3)
	require.NoError(t, err)
	aedZero, err := domain.NewMoney(aed, 0)
	require.NoError(t, err)
	bhdZero, err := domain.NewMoney(bhd, 0)
	require.NoError(t, err)
	fee, err := domain.ParseMoney(aed, "25.00")
	require.NoError(t, err)
	primary := domain.Account{ID: "ACC-001", Currency: aed, OpeningBalance: aedZero}
	instalment := domain.Account{ID: "ACC-002", Currency: bhd, OpeningBalance: bhdZero}
	repository := memory.New()
	core, err := ledger.New(repository, []domain.Account{primary, instalment}, map[domain.Currency]domain.Money{aed: fee, bhd: bhdZero})
	require.NoError(t, err)
	return repository, core, primary, instalment
}
