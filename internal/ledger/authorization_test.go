package ledger

import (
	"errors"
	"fmt"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"inmemoryaccountledgercore/internal/domain"
)

func authorizationFixture(t *testing.T, opening int64) (*Core, *journal, domain.InputEvent) {
	t.Helper()
	core, repository, input := ordinaryFixture(t, opening)
	input.Kind, input.RawAmount, input.AuthorizationID = domain.InputAuthorization, "200.00", "Auth-A"
	input.BookingDay, input.ValueDay, input.Order = 2, 2, 3
	return core, repository, input
}

func authorizationProjection(t *testing.T, repository Repository, day domain.Day, ledger, holds, available int64) *Projection {
	t.Helper()
	history := repository.History()
	projection, err := Rebuild(history, len(history.Records))
	require.NoError(t, err)
	balance, err := projection.Balance("ACC-001", day, BalanceOptions{})
	require.NoError(t, err)
	assert.Equal(t, ledger, balance.MinorUnits())
	active, err := projection.ActiveHolds("ACC-001")
	require.NoError(t, err)
	assert.Equal(t, holds, active.MinorUnits())
	spendable, err := projection.AvailableBalance("ACC-001", day)
	require.NoError(t, err)
	assert.Equal(t, available, spendable.MinorUnits())
	return projection
}

func TestAuthorizationAvailabilityBoundaries(t *testing.T) {
	core, repository, input := authorizationFixture(t, 25000)
	outcome, err := core.Submit(input)
	require.NoError(t, err)
	assert.Equal(t, domain.OutcomeAccepted, outcome.Status)
	assert.Empty(t, outcome.DiagnosticKey)
	history := repository.History()
	require.Len(t, history.Records, 4)
	assert.Equal(t, input, history.Records[1].Attempt.Input)
	assert.Equal(t, outcome, *history.Records[2].Outcome)
	approval := history.Records[3]
	require.NotNil(t, approval.Authorization)
	assert.Equal(t, domain.AuthorizationTransition{
		AccountID: input.AccountID, AuthorizationID: input.AuthorizationID, AttemptID: outcome.AttemptID,
		BookingDay: 2, ValueDay: 2, State: domain.AuthorizationApproved, Hold: fixtureMoney(t, input.Currency, 20000),
	}, *approval.Authorization)
	authorizationProjection(t, repository, 2, 25000, 20000, 5000)

	input.ID, input.AuthorizationID, input.RawAmount = "exact", "exact", "50.00"
	outcome, err = core.Submit(input)
	require.NoError(t, err)
	assert.Equal(t, domain.OutcomeAccepted, outcome.Status, "exactly zero proposed availability approves")
	authorizationProjection(t, repository, 2, 25000, 25000, 0)

	before := repository.History()
	input.ID, input.AuthorizationID, input.RawAmount = "decline", "decline", "0.01"
	outcome, err = core.Submit(input)
	require.NoError(t, err)
	assert.Equal(t, domain.OutcomeDeclined, outcome.Status)
	projection := authorizationProjection(t, repository, 2, 25000, 25000, 0)
	decline, exists := projection.Authorization(input.AccountID, input.AuthorizationID)
	require.True(t, exists)
	assert.Equal(t, domain.AuthorizationDeclined, decline.Transition.State)
	assert.Equal(t, int64(1), decline.Transition.Hold.MinorUnits(), "retain the requested amount without reserving it")
	after := repository.History()
	assert.Equal(t, before.Records, after.Records[:len(before.Records)])
	require.Len(t, after.Records, len(before.Records)+4)
	diagnostic := after.Records[len(after.Records)-1]
	assert.Equal(t, outcome.DiagnosticKey, diagnostic.Key)
	assert.Equal(t, domain.Diagnostic{
		Code: DiagnosticInsufficientAvailability, Reason: "hold would make available balance negative",
		AccountID: input.AccountID, ObservationDay: 2, AttemptID: outcome.AttemptID,
	}, *diagnostic.Diagnostic)
}

func TestAuthorizationIncludesPostedFeesExcludesUnpaidInterestAndFutureValues(t *testing.T) {
	core, repository, input := authorizationFixture(t, 25000)
	// A posted fee reduces 250.00 to 225.00. Day 1's rounded unpaid
	// accrual is 0.09; a future-value credit is not available on Day 2.
	require.NoError(t, repository.Append(domain.Batch{Records: []domain.Record{
		{Key: "fee", Entry: &domain.MonetaryEntry{AccountID: input.AccountID, Kind: domain.EntryFeeAssessment, Amount: fixtureMoney(t, input.Currency, -2500), BookingDay: 1, ValueDay: 1}},
		{Key: "close", Close: &domain.DayClose{AccountID: input.AccountID, Day: 1}},
		{Key: "future", Entry: &domain.MonetaryEntry{AccountID: input.AccountID, Kind: domain.EntryCredit, Amount: fixtureMoney(t, input.Currency, 1), BookingDay: 1, ValueDay: 3}},
	}}))
	projection := authorizationProjection(t, repository, 2, 22500, 0, 22500)
	accrual, exists := projection.Accrual(input.AccountID, 1)
	require.True(t, exists)
	assert.Equal(t, int64(9), accrual.MinorUnits())
	input.RawAmount = "225.01"
	outcome, err := core.Submit(input)
	require.NoError(t, err)
	assert.Equal(t, domain.OutcomeDeclined, outcome.Status)
	input.ID, input.AuthorizationID, input.RawAmount = "exact", "exact", "225.00"
	outcome, err = core.Submit(input)
	require.NoError(t, err)
	assert.Equal(t, domain.OutcomeAccepted, outcome.Status)
	authorizationProjection(t, repository, 2, 22500, 22500, 0)
}

func TestFinalSettlementReleasesOnlyItsEntireHold(t *testing.T) {
	core, repository, input := authorizationFixture(t, 65000)
	_, err := core.Submit(input)
	require.NoError(t, err)
	approval := repository.History().Records[3]
	other := input
	other.ID, other.AuthorizationID, other.RawAmount = "other", "other", "50.00"
	_, err = core.Submit(other)
	require.NoError(t, err)
	input.ID, input.Kind, input.RawAmount = "E5", domain.InputSettlement, "185.00"
	input.BookingDay, input.ValueDay, input.Order = 4, 4, 5
	start := len(repository.records)
	outcome, err := core.Submit(input)
	require.NoError(t, err)
	assert.Equal(t, domain.OutcomeAccepted, outcome.Status)
	records := repository.History().Records[start:]
	require.Len(t, records, 4, "one debit and one transition, without an unused-reservation credit")
	assert.Equal(t, input, records[0].Attempt.Input)
	assert.Equal(t, outcome, *records[1].Outcome)
	assert.Equal(t, domain.MonetaryEntry{
		AccountID: input.AccountID, Amount: fixtureMoney(t, input.Currency, -18500), BookingDay: 4, ValueDay: 4,
		Kind: domain.EntrySettlement, SourceAttemptID: outcome.AttemptID,
	}, *records[2].Entry)
	assert.Equal(t, domain.AuthorizationTransition{
		AccountID: input.AccountID, AuthorizationID: input.AuthorizationID, AttemptID: outcome.AttemptID,
		BookingDay: 4, ValueDay: 4, State: domain.AuthorizationSettled,
		Hold: approval.Authorization.Hold, OriginalHoldKey: approval.Key,
	}, *records[3].Authorization)
	projection := authorizationProjection(t, repository, 4, 46500, 5000, 41500)
	settled, exists := projection.Authorization(input.AccountID, input.AuthorizationID)
	require.True(t, exists)
	assert.Equal(t, domain.AuthorizationSettled, settled.Transition.State)
	assert.Equal(t, approval, repository.History().Records[3])
}

func TestAuthBDeclineSurvivesFinancialRestoration(t *testing.T) {
	core, repository, input := authorizationFixture(t, 46500)
	input.ID, input.AuthorizationID, input.RawAmount = "E8", "Auth-B", "90.00"
	input.BookingDay, input.ValueDay, input.Order = 5, 5, 8
	// Direct posted corrections isolate the P06 checkpoint. Producing these
	// entries through E7/E9 reconciliation is covered by later integration.
	require.NoError(t, repository.Append(domain.Batch{Records: []domain.Record{
		{Key: "E7", Entry: &domain.MonetaryEntry{AccountID: input.AccountID, Kind: domain.EntryDebit, Amount: fixtureMoney(t, input.Currency, -62000), BookingDay: 5, ValueDay: 2}},
		{Key: "fee/day2", Entry: &domain.MonetaryEntry{AccountID: input.AccountID, Kind: domain.EntryFeeAssessment, Amount: fixtureMoney(t, input.Currency, -2500), BookingDay: 5, ValueDay: 2}},
		{Key: "fee/day4", Entry: &domain.MonetaryEntry{AccountID: input.AccountID, Kind: domain.EntryFeeAssessment, Amount: fixtureMoney(t, input.Currency, -2500), BookingDay: 5, ValueDay: 4}},
	}}))
	authorizationProjection(t, repository, 5, -20500, 0, -20500)
	outcome, err := core.Submit(input)
	require.NoError(t, err)
	assert.Equal(t, domain.OutcomeDeclined, outcome.Status)
	before := repository.History()
	require.NoError(t, repository.Append(domain.Batch{Records: []domain.Record{
		{Key: "E9", Entry: &domain.MonetaryEntry{AccountID: input.AccountID, Kind: domain.EntryReversal, Amount: fixtureMoney(t, input.Currency, 62000), BookingDay: 6, ValueDay: 2}},
		{Key: "refund/day2", Entry: &domain.MonetaryEntry{AccountID: input.AccountID, Kind: domain.EntryFeeRefund, Amount: fixtureMoney(t, input.Currency, 2500), BookingDay: 6, ValueDay: 2}},
		{Key: "refund/day4", Entry: &domain.MonetaryEntry{AccountID: input.AccountID, Kind: domain.EntryFeeRefund, Amount: fixtureMoney(t, input.Currency, 2500), BookingDay: 6, ValueDay: 4}},
	}}))
	projection := authorizationProjection(t, repository, 5, 46500, 0, 46500)
	auth, exists := projection.Authorization(input.AccountID, input.AuthorizationID)
	require.True(t, exists)
	assert.Equal(t, domain.AuthorizationDeclined, auth.Transition.State)
	assert.Equal(t, outcome.AttemptID, auth.Transition.AttemptID)
	assert.Equal(t, before.Records, repository.History().Records[:len(before.Records)])
}

func TestAuthorizationAndSettlementValidation(t *testing.T) {
	for _, kind := range []domain.InputKind{domain.InputAuthorization, domain.InputSettlement} {
		for _, tc := range []struct {
			name   string
			change func(*domain.InputEvent)
			code   string
		}{
			{"precision", func(e *domain.InputEvent) { e.RawAmount = "1.005" }, DiagnosticAmountPrecision},
			{"range", func(e *domain.InputEvent) { e.RawAmount = "92233720368547758.08" }, DiagnosticAmountRange},
			{"syntax", func(e *domain.InputEvent) { e.RawAmount = "invalid" }, DiagnosticInvalidAmount},
			{"unknown account", func(e *domain.InputEvent) { e.AccountID = "unknown" }, DiagnosticUnknownAccount},
			{"currency mismatch", func(e *domain.InputEvent) { e.Currency = fixtureCurrency(t, "BHD", 3) }, DiagnosticCurrencyMismatch},
		} {
			t.Run(string(kind)+"/"+tc.name, func(t *testing.T) {
				core, repository, input := authorizationFixture(t, 25000)
				input.Kind = kind
				tc.change(&input)
				outcome, err := core.Submit(input)
				require.NoError(t, err)
				assert.Equal(t, domain.OutcomeRejected, outcome.Status)
				requireAttempt(t, repository, 1, input, outcome)
				assert.Equal(t, tc.code, repository.records[3].Diagnostic.Code)
				authorizationProjection(t, repository, 2, 25000, 0, 25000)
			})
		}
	}
}

func TestAuthorizationAndSettlementDuplicateOutcomes(t *testing.T) {
	for _, scenario := range []string{"approved", "declined", "rejected authorization", "settled", "unknown settlement"} {
		t.Run(scenario, func(t *testing.T) {
			core, repository, input := authorizationFixture(t, 25000)
			switch scenario {
			case "declined":
				input.RawAmount = "250.01"
			case "rejected authorization":
				input.RawAmount = "1.005"
			case "settled":
				_, err := core.Submit(input)
				require.NoError(t, err)
				input.ID, input.Kind, input.RawAmount = "settlement", domain.InputSettlement, "185.00"
				input.BookingDay, input.ValueDay = 4, 4
			case "unknown settlement":
				input.Kind = domain.InputSettlement
			}
			_, err := core.Submit(input)
			require.NoError(t, err)
			before := repository.History()
			for _, changed := range []bool{false, true} {
				duplicate := input
				if changed {
					duplicate.Kind, duplicate.AccountID, duplicate.RawAmount = domain.InputReversal, "unknown", "invalid"
					duplicate.AuthorizationID = "another"
				}
				start := len(repository.records)
				outcome, err := core.Submit(duplicate)
				require.NoError(t, err)
				assert.Equal(t, domain.OutcomeRejected, outcome.Status)
				requireAttempt(t, repository, start, duplicate, outcome)
				assert.Equal(t, DiagnosticDuplicateID, repository.records[start+2].Diagnostic.Code)
				assert.Equal(t, before.Records, repository.History().Records[:len(before.Records)])
			}
			// Every record added since the original is a rejected attempt,
			// outcome, or diagnostic: no monetary or hold effects.
			for _, record := range repository.History().Records[len(before.Records):] {
				assert.Nil(t, record.Entry)
				assert.Nil(t, record.Authorization)
			}
		})
	}
}

func TestAuthorizationTechnicalFailuresAreAtomic(t *testing.T) {
	t.Run("proposed availability underflow", func(t *testing.T) {
		core, repository, input := authorizationFixture(t, math.MinInt64)
		input.RawAmount = "0.01"
		before := repository.History()
		outcome, err := core.Submit(input)
		require.ErrorIs(t, err, domain.ErrArithmeticOverflow)
		assert.Equal(t, domain.InputOutcome{}, outcome)
		assert.Equal(t, before, repository.History())
		assert.Equal(t, 1, repository.appends)
	})
	t.Run("later value-date availability underflow", func(t *testing.T) {
		core, repository, input := authorizationFixture(t, 25000)
		// The later debit leaves MinInt64 + 1. The two-unit hold fits
		// on Day 2 but would underflow availability at that later date.
		require.NoError(t, repository.Append(domain.Batch{Records: []domain.Record{
			{Key: "future/first", Entry: &domain.MonetaryEntry{AccountID: input.AccountID, Kind: domain.EntryDebit, Amount: fixtureMoney(t, input.Currency, math.MinInt64), ValueDay: 3}},
			{Key: "future/second", Entry: &domain.MonetaryEntry{AccountID: input.AccountID, Kind: domain.EntryDebit, Amount: fixtureMoney(t, input.Currency, -24999), ValueDay: 3}},
		}}))
		input.RawAmount = "0.02"
		before := repository.History()
		outcome, err := core.Submit(input)
		require.ErrorIs(t, err, domain.ErrArithmeticOverflow)
		assert.Equal(t, domain.InputOutcome{}, outcome)
		assert.Equal(t, before, repository.History())
	})
	for _, scenario := range []string{"approved", "declined", "settled", "unknown settlement", "negative authorization", "negative settlement"} {
		t.Run(scenario, func(t *testing.T) {
			core, repository, input := authorizationFixture(t, 25000)
			if scenario == "declined" {
				input.RawAmount = "250.01"
			}
			if scenario == "settled" {
				_, err := core.Submit(input)
				require.NoError(t, err)
				input.ID = "settlement"
			}
			if scenario == "settled" || scenario == "unknown settlement" || scenario == "negative settlement" {
				input.Kind, input.RawAmount = domain.InputSettlement, "185.00"
			}
			if scenario == "negative authorization" || scenario == "negative settlement" {
				input.RawAmount = "-0.01"
			}
			before := repository.History()
			appends := repository.appends
			failure := errors.New("append unavailable")
			repository.appendError = failure
			outcome, err := core.Submit(input)
			require.ErrorIs(t, err, failure)
			assert.Equal(t, domain.InputOutcome{}, outcome)
			assert.Equal(t, before, repository.History())
			assert.Equal(t, appends+1, repository.appends)
			repository.appendError = nil
			_, err = core.Submit(input)
			require.NoError(t, err, "uncommitted attempts reserve neither input nor authorization IDs")
			assert.Equal(t, appends+2, repository.appends, "retry commits exactly one batch")
		})
	}
}

func TestNegativeAuthorizationAndSettlementAmountsRejectWithoutEffects(t *testing.T) {
	for _, kind := range []domain.InputKind{domain.InputAuthorization, domain.InputSettlement} {
		for _, raw := range []string{"-0.01", "-92233720368547758.08"} {
			t.Run(string(kind)+"/"+raw, func(t *testing.T) {
				core, repository, input := authorizationFixture(t, 25000)
				var holds int64
				if kind == domain.InputSettlement {
					_, err := core.Submit(input)
					require.NoError(t, err)
					input.ID = "settlement"
					holds = 20000
				}
				input.Kind, input.RawAmount = kind, raw
				before := repository.History()
				appends := repository.appends
				outcome, err := core.Submit(input)
				require.NoError(t, err)
				assert.Equal(t, domain.OutcomeRejected, outcome.Status)
				assert.Equal(t, appends+1, repository.appends, "rejection commits exactly once")
				requireAttempt(t, repository, len(before.Records), input, outcome)
				after := repository.History()
				assert.Equal(t, DiagnosticNegativeAmount, after.Records[len(after.Records)-1].Diagnostic.Code)
				assert.Equal(t, before.Records, after.Records[:len(before.Records)])
				authorizationProjection(t, repository, 2, 25000, holds, 25000-holds)
			})
		}
	}
}

func TestAuthorizationBuildersRejectNegativeAmounts(t *testing.T) {
	for _, kind := range []domain.InputKind{domain.InputAuthorization, domain.InputSettlement} {
		for _, minor := range []int64{-1, math.MinInt64} {
			t.Run(fmt.Sprintf("%s/%d", kind, minor), func(t *testing.T) {
				core, repository, input := authorizationFixture(t, 25000)
				var holds int64
				if kind == domain.InputSettlement {
					_, err := core.Submit(input)
					require.NoError(t, err)
					input.ID = "settlement"
					holds = 20000
				}
				input.Kind = kind
				before := repository.History()
				attempt, _ := prepareAttempt(before, input)
				amount := fixtureMoney(t, input.Currency, minor)
				builder := authorizationBatch
				if kind == domain.InputSettlement {
					builder = settlementBatch
				}
				// Bypass Submit's parser to exercise each workflow's own guard.
				batch, outcome, err := builder(before, attempt, amount)
				require.NoError(t, err)
				assert.Equal(t, domain.OutcomeRejected, outcome.Status)
				require.Len(t, batch.Records, 3)
				assert.Equal(t, attempt, *batch.Records[0].Attempt)
				assert.Equal(t, outcome, *batch.Records[1].Outcome)
				assert.Equal(t, DiagnosticNegativeAmount, batch.Records[2].Diagnostic.Code)
				assert.Equal(t, outcome.DiagnosticKey, batch.Records[2].Key)
				for _, record := range batch.Records {
					assert.Nil(t, record.Entry, "negative settlement must not produce a credit")
					assert.Nil(t, record.Authorization, "negative request must not create or release a hold")
				}
				assert.Equal(t, before, repository.History(), "building a batch cannot commit it")
				authorizationProjection(t, repository, 2, 25000, holds, 25000-holds)
			})
		}
	}
}

func TestZeroAuthorizationAndSettlementAmounts(t *testing.T) {
	for _, raw := range []string{"0.00", "-0.00"} {
		t.Run("authorization/"+raw, func(t *testing.T) {
			core, repository, input := authorizationFixture(t, 25000)
			input.RawAmount = raw
			outcome, err := core.Submit(input)
			require.NoError(t, err)
			assert.Equal(t, domain.OutcomeAccepted, outcome.Status)
			projection := authorizationProjection(t, repository, 2, 25000, 0, 25000)
			auth, exists := projection.Authorization(input.AccountID, input.AuthorizationID)
			require.True(t, exists)
			assert.Equal(t, domain.AuthorizationApproved, auth.Transition.State)
			assert.Zero(t, auth.Transition.Hold.MinorUnits())
		})
		t.Run("settlement/"+raw, func(t *testing.T) {
			core, repository, input := authorizationFixture(t, 25000)
			_, err := core.Submit(input)
			require.NoError(t, err)
			approval := repository.History().Records[3]
			input.ID, input.Kind, input.RawAmount = "settlement", domain.InputSettlement, raw
			outcome, err := core.Submit(input)
			require.NoError(t, err)
			assert.Equal(t, domain.OutcomeAccepted, outcome.Status)
			projection := authorizationProjection(t, repository, 2, 25000, 0, 25000)
			auth, exists := projection.Authorization(input.AccountID, input.AuthorizationID)
			require.True(t, exists)
			assert.Equal(t, domain.AuthorizationSettled, auth.Transition.State)
			assert.Equal(t, approval.Key, auth.Transition.OriginalHoldKey)
			assert.Equal(t, approval.Authorization.Hold, auth.Transition.Hold)
			history := repository.History()
			assert.Equal(t, domain.EntrySettlement, history.Records[len(history.Records)-2].Entry.Kind)
			assert.Zero(t, history.Records[len(history.Records)-2].Entry.Amount.MinorUnits())
		})
	}
	core, repository, input := authorizationFixture(t, -1)
	input.RawAmount = "0.00"
	outcome, err := core.Submit(input)
	require.NoError(t, err)
	assert.Equal(t, domain.OutcomeDeclined, outcome.Status, "zero passes validation but still requires nonnegative proposed availability")
	authorizationProjection(t, repository, 2, -1, 0, -1)
	input.ID, input.Kind, input.AuthorizationID = "unknown settlement", domain.InputSettlement, "unknown"
	outcome, err = core.Submit(input)
	require.NoError(t, err)
	assert.Equal(t, domain.OutcomeRejected, outcome.Status)
	history := repository.History()
	assert.Equal(t, DiagnosticUnknownAuthorization, history.Records[len(history.Records)-1].Diagnostic.Code)
}

func TestDeferredAuthorizationWorkflowsAppendNothing(t *testing.T) {
	for _, scenario := range []string{"different authorization dates", "missing authorization ID", "missing settlement ID", "reused authorization ID", "inactive settlement", "over-capture"} {
		t.Run(scenario, func(t *testing.T) {
			core, repository, input := authorizationFixture(t, 25000)
			switch scenario {
			case "different authorization dates":
				input.ValueDay = 1
			case "missing authorization ID":
				input.AuthorizationID = ""
			case "missing settlement ID":
				input.Kind, input.AuthorizationID = domain.InputSettlement, ""
			default:
				if scenario == "inactive settlement" {
					input.RawAmount = "250.01"
				}
				_, err := core.Submit(input)
				require.NoError(t, err)
				input.ID = "fresh"
				if scenario != "reused authorization ID" {
					input.Kind = domain.InputSettlement
				}
				if scenario == "over-capture" {
					input.RawAmount = "200.01"
				}
			}
			before := repository.History()
			outcome, err := core.Submit(input)
			require.ErrorIs(t, err, ErrUnsupportedInput)
			assert.Equal(t, domain.InputOutcome{}, outcome)
			assert.Equal(t, before, repository.History())
		})
	}
}
