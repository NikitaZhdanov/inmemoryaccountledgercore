package ledger

import (
	"errors"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"inmemoryaccountledgercore/internal/domain"
)

func reversalFixture(t *testing.T) (*Core, *journal, domain.InputEvent, domain.Record) {
	t.Helper()
	core, repository, debit := ordinaryFixture(t, 25000)
	debit.ID, debit.Kind, debit.RawAmount = "E7", domain.InputDebit, "620.00"
	_, err := core.Submit(debit)
	require.NoError(t, err)
	return core, repository, domain.InputEvent{
		ID: "E9", Kind: domain.InputReversal, AccountID: debit.AccountID, Currency: debit.Currency,
		BookingDay: 6, ValueDay: 2, ReversesEventID: "E7", Order: 9,
	}, repository.History().Records[3]
}

func TestLinkedPrincipalReversalAndIdempotence(t *testing.T) {
	core, repository, input, original := reversalFixture(t)
	requireBalance(t, repository, input.AccountID, 2, -37000)
	// Source lookup must select E7's original attempt, even after a rejected reuse.
	duplicateSource := repository.records[1].Attempt.Input
	duplicateSource.RawAmount = "1.00"
	_, err := core.Submit(duplicateSource)
	require.NoError(t, err)
	before := repository.History()
	outcome, err := core.Submit(input)
	require.NoError(t, err)
	assert.Equal(t, domain.OutcomeAccepted, outcome.Status)
	requireAttempt(t, repository, len(before.Records), input, outcome)
	after := repository.History()
	assert.Equal(t, before.Records, after.Records[:len(before.Records)])
	assert.Equal(t, domain.MonetaryEntry{
		AccountID: input.AccountID, Amount: fixtureMoney(t, input.Currency, 62000),
		BookingDay: 6, ValueDay: 2, Kind: domain.EntryReversal,
		SourceAttemptID: outcome.AttemptID, CauseKey: domain.RecordKey(outcome.AttemptID), OriginalEntryKey: original.Key,
	}, *after.Records[len(after.Records)-1].Entry)
	requireBalance(t, repository, input.AccountID, 1, 25000)
	requireBalance(t, repository, input.AccountID, 2, 25000)

	appends := repository.appends
	for range 2 {
		batch, err := principalCorrection(repository.History(), *after.Records[len(before.Records)].Attempt, original)
		require.NoError(t, err)
		assert.Empty(t, batch.Records, "repeated internal correction has no records to append")
	}
	assert.Equal(t, appends, repository.appends)
	assert.Equal(t, after, repository.History())

	for _, changed := range []bool{false, true} {
		duplicate := input
		if changed {
			duplicate.AccountID, duplicate.RawAmount, duplicate.ReversesEventID = "unknown", "invalid", "missing"
			duplicate.InstalmentCount = -1
		}
		start := len(repository.records)
		rejected, err := core.Submit(duplicate)
		require.NoError(t, err)
		assert.Equal(t, domain.OutcomeRejected, rejected.Status)
		requireAttempt(t, repository, start, duplicate, rejected)
		assert.Equal(t, DiagnosticDuplicateID, repository.records[start+2].Diagnostic.Code)
		requireBalance(t, repository, input.AccountID, 6, 25000)
	}
}

func TestReversalValidationAndUnsupportedExtensions(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(*domain.InputEvent)
		code   string
	}{
		{"account", func(e *domain.InputEvent) { e.AccountID = "unknown" }, DiagnosticUnknownAccount},
		{"currency", func(e *domain.InputEvent) { e.Currency = domain.Currency{} }, DiagnosticInvalidCurrency},
		{"currency mismatch", func(e *domain.InputEvent) { e.Currency = fixtureCurrency(t, "BHD", 3) }, DiagnosticCurrencyMismatch},
		{"missing reference", func(e *domain.InputEvent) { e.ReversesEventID = "" }, ""},
		{"unknown reference", func(e *domain.InputEvent) { e.ReversesEventID = "missing" }, ""},
		{"different value day", func(e *domain.InputEvent) { e.ValueDay = 3 }, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			core, repository, input, _ := reversalFixture(t)
			tc.change(&input)
			before := repository.History()
			outcome, err := core.Submit(input)
			if tc.code == "" {
				require.ErrorIs(t, err, ErrUnsupportedInput)
				assert.Equal(t, domain.InputOutcome{}, outcome)
				assert.Equal(t, before, repository.History())
			} else {
				require.NoError(t, err)
				assert.Equal(t, domain.OutcomeRejected, outcome.Status)
				requireAttempt(t, repository, len(before.Records), input, outcome)
				assert.Equal(t, tc.code, repository.records[len(before.Records)+2].Diagnostic.Code)
			}
			requireBalance(t, repository, "ACC-001", 6, -37000)
		})
	}
}

func TestAlreadyReversedDebitIsBusinessRejection(t *testing.T) {
	for _, tc := range []struct {
		name       string
		failAppend bool
	}{
		{"successful rejection", false},
		{"failed rejection append and retry", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			core, repository, input, original := reversalFixture(t)
			accepted, err := core.Submit(input)
			require.NoError(t, err)
			require.Equal(t, domain.OutcomeAccepted, accepted.Status)
			before := repository.History()
			input.ID = "second-cause"
			attempt, duplicate := prepareAttempt(before, input)
			require.False(t, duplicate)
			require.NotEqual(t, accepted.AttemptID, attempt.ID)
			batch, err := principalCorrection(before, attempt, original)
			require.ErrorIs(t, err, errAlreadyReversed)
			assert.NotErrorIs(t, err, ErrUnsupportedInput)
			assert.Empty(t, batch.Records)
			if tc.failAppend {
				failure := errors.New("rejection append unavailable")
				repository.appendError = failure
				outcome, err := core.Submit(input)
				require.ErrorIs(t, err, failure)
				assert.Equal(t, domain.InputOutcome{}, outcome)
				assert.Equal(t, before, repository.History())
				repository.appendError = nil
			}
			appends := repository.appends
			rejected, err := core.Submit(input)
			require.NoError(t, err)
			assert.Equal(t, domain.OutcomeRejected, rejected.Status)
			assert.Equal(t, attempt.ID, rejected.AttemptID)
			assert.Equal(t, appends+1, repository.appends)
			requireAttempt(t, repository, len(before.Records), input, rejected)
			after := repository.History()
			assert.Equal(t, before.Records, after.Records[:len(before.Records)])
			assert.Equal(t, DiagnosticAlreadyReversed, after.Records[len(before.Records)+2].Diagnostic.Code)
			requireBalance(t, repository, input.AccountID, 6, 25000)

			// A rejected reversal reserves its input ID like all other rejections.
			start := len(after.Records)
			duplicateOutcome, err := core.Submit(input)
			require.NoError(t, err)
			assert.Equal(t, domain.OutcomeRejected, duplicateOutcome.Status)
			requireAttempt(t, repository, start, input, duplicateOutcome)
			assert.Equal(t, DiagnosticDuplicateID, repository.records[start+2].Diagnostic.Code)
			requireBalance(t, repository, input.AccountID, 6, 25000)

			input.ID, input.Kind, input.RawAmount, input.ReversesEventID = "continuation", domain.InputCredit, "0.01", ""
			outcome, err := core.Submit(input)
			require.NoError(t, err)
			assert.Equal(t, domain.OutcomeAccepted, outcome.Status)
			requireBalance(t, repository, input.AccountID, 6, 25001)
		})
	}
}

func TestReversalTechnicalFailuresAreAtomic(t *testing.T) {
	t.Run("append failure and retry", func(t *testing.T) {
		core, repository, input, _ := reversalFixture(t)
		before := repository.History()
		failure := errors.New("append unavailable")
		repository.appendError = failure
		outcome, err := core.Submit(input)
		require.ErrorIs(t, err, failure)
		assert.Equal(t, domain.InputOutcome{}, outcome)
		assert.Equal(t, before, repository.History())
		repository.appendError = nil
		outcome, err = core.Submit(input)
		require.NoError(t, err)
		assert.Equal(t, domain.OutcomeAccepted, outcome.Status)
		requireBalance(t, repository, input.AccountID, 6, 25000)
	})
	t.Run("later value day overflow", func(t *testing.T) {
		core, repository, input := ordinaryFixture(t, 1)
		input.ID, input.Kind, input.RawAmount = "debit", domain.InputDebit, "0.01"
		_, err := core.Submit(input)
		require.NoError(t, err)
		input.ID, input.Kind, input.RawAmount, input.ValueDay = "future", domain.InputCredit, "92233720368547758.07", 3
		_, err = core.Submit(input)
		require.NoError(t, err)
		before := repository.History()
		input.ID, input.Kind, input.RawAmount, input.ValueDay = "reverse", domain.InputReversal, "", 2
		input.ReversesEventID = "debit"
		outcome, err := core.Submit(input)
		require.ErrorIs(t, err, domain.ErrArithmeticOverflow)
		assert.Equal(t, domain.InputOutcome{}, outcome)
		assert.Equal(t, before, repository.History())
	})
	t.Run("unrepresentable inverse", func(t *testing.T) {
		_, repository, input, original := reversalFixture(t)
		original.Entry.Amount = fixtureMoney(t, input.Currency, math.MinInt64)
		batch, err := principalCorrection(repository.History(), domain.InputAttempt{ID: "correction", Input: input}, original)
		require.ErrorIs(t, err, domain.ErrArithmeticOverflow)
		assert.Empty(t, batch.Records)
	})
}
