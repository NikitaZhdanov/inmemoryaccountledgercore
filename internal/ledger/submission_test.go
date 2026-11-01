package ledger

import (
	"errors"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"inmemoryaccountledgercore/internal/domain"
)

func ordinaryFixture(t *testing.T, opening int64) (*Core, *journal, domain.InputEvent) {
	t.Helper()
	currency := fixtureCurrency(t, "AED", 2)
	repository := &journal{}
	core, err := New(repository,
		[]domain.Account{{ID: "ACC-001", Currency: currency, OpeningBalance: fixtureMoney(t, currency, opening)}},
		map[domain.Currency]domain.Money{currency: fixtureMoney(t, currency, 2500)})
	require.NoError(t, err)
	return core, repository, domain.InputEvent{
		ID: "input", Kind: domain.InputCredit, AccountID: "ACC-001", Currency: currency,
		RawAmount: "1.000", BookingDay: 5, ValueDay: 2, Order: 7,
	}
}

func requireBalance(t *testing.T, repository Repository, id domain.AccountID, day domain.Day, want int64) {
	t.Helper()
	history := repository.History()
	projection, err := Rebuild(history, len(history.Records))
	require.NoError(t, err)
	balance, err := projection.Balance(id, day, BalanceOptions{})
	require.NoError(t, err)
	assert.Equal(t, want, balance.MinorUnits())
}

func requireAttempt(t *testing.T, repository Repository, start int, input domain.InputEvent, outcome domain.InputOutcome) {
	t.Helper()
	records := repository.History().Records[start:]
	require.Len(t, records, 3)
	require.NotNil(t, records[0].Attempt)
	assert.Equal(t, domain.InputAttempt{ID: outcome.AttemptID, Input: input}, *records[0].Attempt)
	require.NotNil(t, records[1].Outcome)
	assert.Equal(t, outcome, *records[1].Outcome)
	if outcome.Status == domain.OutcomeRejected {
		require.NotNil(t, records[2].Diagnostic)
		assert.Equal(t, outcome.DiagnosticKey, records[2].Key)
		assert.Equal(t, input.AccountID, records[2].Diagnostic.AccountID)
		assert.Equal(t, input.BookingDay, records[2].Diagnostic.ObservationDay)
		assert.Equal(t, outcome.AttemptID, records[2].Diagnostic.AttemptID)
		assert.NotEmpty(t, records[2].Diagnostic.Reason)
	} else {
		require.NotNil(t, records[2].Entry)
		assert.Empty(t, outcome.DiagnosticKey)
		assert.Equal(t, outcome.AttemptID, records[2].Entry.SourceAttemptID)
		assert.Equal(t, input.BookingDay, records[2].Entry.BookingDay)
		assert.Equal(t, input.ValueDay, records[2].Entry.ValueDay)
	}
}

func TestOrdinaryPostingAndSourceRetention(t *testing.T) {
	for _, tc := range []struct {
		kind domain.InputKind
		want int64
	}{
		{domain.InputCredit, 100},
		{domain.InputDebit, -100},
	} {
		t.Run(string(tc.kind), func(t *testing.T) {
			core, repository, input := ordinaryFixture(t, 0)
			input.Kind = tc.kind
			// All original fields survive even if unused by an ordinary posting.
			input.AuthorizationID, input.ReversesEventID = "reference", "original"
			outcome, err := core.Submit(input)
			require.NoError(t, err)
			assert.Equal(t, domain.OutcomeAccepted, outcome.Status)
			requireAttempt(t, repository, 1, input, outcome)
			assert.Equal(t, tc.want, repository.records[3].Entry.Amount.MinorUnits())
			assert.Equal(t, domain.EntryKind(tc.kind), repository.records[3].Entry.Kind)
			requireBalance(t, repository, input.AccountID, 1, 0)
			requireBalance(t, repository, input.AccountID, 2, tc.want)
			before := repository.History()
			input.RawAmount = "changed"
			snapshot := repository.History()
			snapshot.Records[1].Attempt.Input.RawAmount = "changed snapshot"
			snapshot.Records[2].Outcome.Status = domain.OutcomeRejected
			assert.Equal(t, before, repository.History())
		})
	}
}

func TestOrdinaryInputRejectionsAndContinuation(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(*domain.InputEvent)
		code   string
	}{
		{"precision", func(e *domain.InputEvent) { e.RawAmount = "1.005" }, DiagnosticAmountPrecision},
		{"upper range", func(e *domain.InputEvent) { e.RawAmount = "92233720368547758.08" }, DiagnosticAmountRange},
		{"lower range", func(e *domain.InputEvent) { e.RawAmount = "-92233720368547758.09" }, DiagnosticAmountRange},
		{"syntax", func(e *domain.InputEvent) { e.RawAmount = "1e2" }, DiagnosticInvalidAmount},
		{"negative credit", func(e *domain.InputEvent) { e.RawAmount = "-0.01" }, DiagnosticNegativeAmount},
		{"negative debit", func(e *domain.InputEvent) { e.Kind = domain.InputDebit; e.RawAmount = "-0.01" }, DiagnosticNegativeAmount},
		{"minimum negative", func(e *domain.InputEvent) { e.Kind = domain.InputDebit; e.RawAmount = "-92233720368547758.08" }, DiagnosticNegativeAmount},
		{"unknown account", func(e *domain.InputEvent) { e.AccountID = "unknown" }, DiagnosticUnknownAccount},
		{"invalid currency", func(e *domain.InputEvent) { e.Currency = domain.Currency{} }, DiagnosticInvalidCurrency},
		{"different code", func(e *domain.InputEvent) { e.Currency = fixtureCurrency(t, "BHD", 3) }, DiagnosticCurrencyMismatch},
		{"different scale", func(e *domain.InputEvent) { e.Currency = fixtureCurrency(t, "AED", 3) }, DiagnosticCurrencyMismatch},
		{"empty ID", func(e *domain.InputEvent) { e.ID = "" }, DiagnosticInvalidEventID},
	} {
		t.Run(tc.name, func(t *testing.T) {
			core, repository, input := ordinaryFixture(t, 0)
			valid := input
			tc.change(&input)
			outcome, err := core.Submit(input)
			require.NoError(t, err)
			assert.Equal(t, domain.OutcomeRejected, outcome.Status)
			requireAttempt(t, repository, 1, input, outcome)
			assert.Equal(t, tc.code, repository.records[3].Diagnostic.Code)
			requireBalance(t, repository, valid.AccountID, 6, 0)
			valid.ID = "fresh"
			accepted, err := core.Submit(valid)
			require.NoError(t, err)
			assert.Equal(t, domain.OutcomeAccepted, accepted.Status)
			assert.NotEqual(t, outcome.AttemptID, accepted.AttemptID)
			requireBalance(t, repository, valid.AccountID, 6, 100)
		})
	}
}

func TestDuplicateInputsPreserveOriginalOutcomes(t *testing.T) {
	for _, originalRejected := range []bool{false, true} {
		core, repository, input := ordinaryFixture(t, 0)
		if originalRejected {
			input.RawAmount = "1.005"
		}
		original, err := core.Submit(input)
		require.NoError(t, err)
		before := repository.History()
		attempts := map[domain.AttemptID]bool{original.AttemptID: true}
		for _, changed := range []bool{false, true} {
			duplicate := input
			if changed {
				// Duplication precedes kind dispatch, amount/account validation,
				// and all effects, including future event-specific processing.
				duplicate.Kind = domain.InputReversal
				duplicate.RawAmount = "invalid"
				duplicate.AccountID = "unknown"
				duplicate.BookingDay, duplicate.ValueDay = 6, 1
			}
			start := len(repository.records)
			outcome, err := core.Submit(duplicate)
			require.NoError(t, err)
			assert.Equal(t, domain.OutcomeRejected, outcome.Status)
			requireAttempt(t, repository, start, duplicate, outcome)
			assert.Equal(t, DiagnosticDuplicateID, repository.records[start+2].Diagnostic.Code)
			assert.False(t, attempts[outcome.AttemptID], "every committed attempt has a fresh identity")
			attempts[outcome.AttemptID] = true
			assert.Equal(t, before.Records, repository.History().Records[:len(before.Records)])
		}
		want := int64(100)
		if originalRejected {
			want = 0
		}
		requireBalance(t, repository, input.AccountID, 6, want)
	}
}

func TestZeroAndEndpointPostings(t *testing.T) {
	for _, tc := range []struct {
		kind    domain.InputKind
		raw     string
		opening int64
		want    int64
	}{
		{domain.InputCredit, "0.000", 0, 0},
		{domain.InputDebit, "-0.00", 0, 0},
		{domain.InputCredit, "92233720368547758.07", 0, math.MaxInt64},
		{domain.InputDebit, "92233720368547758.07", 0, -math.MaxInt64},
		{domain.InputDebit, "0.01", -math.MaxInt64, math.MinInt64},
	} {
		core, repository, input := ordinaryFixture(t, tc.opening)
		input.Kind, input.RawAmount = tc.kind, tc.raw
		outcome, err := core.Submit(input)
		require.NoError(t, err)
		assert.Equal(t, domain.OutcomeAccepted, outcome.Status)
		requireBalance(t, repository, input.AccountID, 6, tc.want)
	}
}

func TestOrdinaryConfiguredPrecision(t *testing.T) {
	for _, tc := range []struct {
		code       string
		scale      int
		raw, bad   string
		minorUnits int64
	}{
		{"BHD", 3, "1.0000", "1.0001", 1000},
		{"TOK", 0, "1.000", "1.1", 1},
		{"TOK", 4, "0.00010", "0.00001", 1},
		{"TOK", 18, "0.0000000000000000010", "0.0000000000000000001", 1},
	} {
		currency := fixtureCurrency(t, tc.code, tc.scale)
		zero := fixtureMoney(t, currency, 0)
		repository := &journal{}
		core, err := New(repository, []domain.Account{{ID: "A", Currency: currency, OpeningBalance: zero}}, map[domain.Currency]domain.Money{currency: zero})
		require.NoError(t, err)
		input := domain.InputEvent{ID: "invalid", Kind: domain.InputCredit, AccountID: "A", Currency: currency, RawAmount: tc.bad, BookingDay: 1, ValueDay: 1}
		outcome, err := core.Submit(input)
		require.NoError(t, err)
		assert.Equal(t, domain.OutcomeRejected, outcome.Status)
		assert.Equal(t, DiagnosticAmountPrecision, repository.records[3].Diagnostic.Code)
		input.ID, input.RawAmount = "valid", tc.raw
		outcome, err = core.Submit(input)
		require.NoError(t, err)
		assert.Equal(t, domain.OutcomeAccepted, outcome.Status)
		requireAttempt(t, repository, 4, input, outcome)
		requireBalance(t, repository, "A", 1, tc.minorUnits)
	}
}

func TestDerivedPostingFailuresAreAtomic(t *testing.T) {
	for _, tc := range []struct {
		name    string
		opening int64
		kind    domain.InputKind
	}{
		{"credit overflow", math.MaxInt64, domain.InputCredit},
		{"debit underflow", math.MinInt64, domain.InputDebit},
	} {
		t.Run(tc.name, func(t *testing.T) {
			core, repository, input := ordinaryFixture(t, tc.opening)
			before := repository.History()
			input.Kind, input.RawAmount = tc.kind, "0.01"
			outcome, err := core.Submit(input)
			require.ErrorIs(t, err, domain.ErrArithmeticOverflow)
			assert.Equal(t, domain.InputOutcome{}, outcome)
			assert.Equal(t, before, repository.History())
			assert.Equal(t, 1, repository.appends, "only initialization appended")
		})
	}
	core, repository, input := ordinaryFixture(t, 0)
	input.ID, input.RawAmount, input.ValueDay = "future", "92233720368547758.07", 3
	_, err := core.Submit(input)
	require.NoError(t, err)
	before := repository.History()
	input.ID, input.RawAmount, input.ValueDay = "backdated", "0.01", 2
	outcome, err := core.Submit(input)
	require.ErrorIs(t, err, domain.ErrArithmeticOverflow, "Day 2 fits but Day 3 cannot")
	assert.Equal(t, domain.InputOutcome{}, outcome)
	assert.Equal(t, before, repository.History())
}

func TestSubmissionRepositoryFailures(t *testing.T) {
	for _, rejection := range []bool{false, true} {
		core, repository, input := ordinaryFixture(t, 0)
		if rejection {
			input.RawAmount = "1.005"
		}
		failure := errors.New("append unavailable")
		repository.appendError = failure
		before := repository.History()
		outcome, err := core.Submit(input)
		require.ErrorIs(t, err, failure)
		assert.Equal(t, domain.InputOutcome{}, outcome)
		assert.Equal(t, before, repository.History())
		repository.appendError = nil
		input.RawAmount = "1.000"
		outcome, err = core.Submit(input)
		require.NoError(t, err)
		assert.Equal(t, domain.OutcomeAccepted, outcome.Status, "uncommitted attempts do not reserve event IDs")
		requireAttempt(t, repository, 1, input, outcome)
	}
}

func TestUnimplementedWorkflowsAppendNothing(t *testing.T) {
	core, repository, input := ordinaryFixture(t, 0)
	before := repository.History()
	for _, kind := range []domain.InputKind{domain.InputAuthorization, domain.InputSettlement, domain.InputReversal, "unknown"} {
		input.Kind = kind
		outcome, err := core.Submit(input)
		require.ErrorIs(t, err, ErrUnsupportedInput)
		assert.Equal(t, domain.InputOutcome{}, outcome)
		assert.Equal(t, before, repository.History())
	}
	input.Kind, input.InstalmentCount = domain.InputCredit, 3
	_, err := core.Submit(input)
	require.ErrorIs(t, err, ErrUnsupportedInput)
	assert.Equal(t, before, repository.History())
}
