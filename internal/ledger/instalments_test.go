package ledger

import (
	"errors"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"inmemoryaccountledgercore/internal/domain"
	"inmemoryaccountledgercore/internal/posting"
)

func instalmentFixture(t *testing.T, opening int64) (*Core, *journal, domain.InputEvent) {
	t.Helper()
	bhd := fixtureCurrency(t, "BHD", 3)
	zero := fixtureMoney(t, bhd, 0)
	repository := &journal{}
	core, err := New(repository, []domain.Account{{ID: "ACC-002", Currency: bhd, OpeningBalance: fixtureMoney(t, bhd, opening)}}, map[domain.Currency]domain.Money{bhd: zero})
	require.NoError(t, err)
	return core, repository, domain.InputEvent{
		ID: "E10", Kind: domain.InputCredit, AccountID: "ACC-002", Currency: bhd,
		RawAmount: "10.000", BookingDay: 5, ValueDay: 5, Order: 10, InstalmentCount: 3,
	}
}

func TestOrderedInstalmentPostings(t *testing.T) {
	for _, tc := range []struct {
		raw   string
		count int
		want  []int64
		total int64
	}{
		{"10.000", 3, []int64{3333, 3333, 3334}, 10000},
		{"0.002", 3, []int64{0, 0, 2}, 2},
		{"0.000", 3, []int64{0, 0, 0}, 0},
		{"10.0000", 1, []int64{10000}, 10000},
	} {
		t.Run(tc.raw, func(t *testing.T) {
			core, repository, input := instalmentFixture(t, 0)
			input.RawAmount, input.InstalmentCount = tc.raw, tc.count
			appends := repository.appends
			outcome, err := core.Submit(input)
			require.NoError(t, err)
			assert.Equal(t, domain.OutcomeAccepted, outcome.Status)
			assert.Equal(t, appends+1, repository.appends, "all instalments commit in one batch")
			history := repository.History()
			require.Len(t, history.Records, 3+tc.count)
			assert.Equal(t, input, history.Records[1].Attempt.Input)
			assert.Equal(t, outcome, *history.Records[2].Outcome)
			keys := map[domain.RecordKey]bool{}
			var total int64
			for i, want := range tc.want {
				record := history.Records[3+i]
				require.NotNil(t, record.Entry)
				assert.Equal(t, domain.MonetaryEntry{
					AccountID: input.AccountID, Amount: fixtureMoney(t, input.Currency, want),
					BookingDay: 5, ValueDay: 5, Kind: domain.EntryInstalment, SourceAttemptID: outcome.AttemptID,
					CauseKey: domain.RecordKey(outcome.AttemptID), Instalment: i + 1, InstalmentCount: tc.count,
				}, *record.Entry)
				assert.NotEmpty(t, record.Key)
				assert.False(t, keys[record.Key])
				keys[record.Key] = true
				total += record.Entry.Amount.MinorUnits()
			}
			assert.Equal(t, tc.total, total)
			requireBalance(t, repository, input.AccountID, 4, 0)
			requireBalance(t, repository, input.AccountID, 5, tc.total)
			for _, changed := range []bool{false, true} {
				duplicate := input
				if changed {
					duplicate.RawAmount, duplicate.InstalmentCount = "invalid", -1
				}
				start := len(repository.records)
				rejected, err := core.Submit(duplicate)
				require.NoError(t, err)
				assert.Equal(t, domain.OutcomeRejected, rejected.Status)
				requireAttempt(t, repository, start, duplicate, rejected)
				assert.Equal(t, posting.DiagnosticDuplicateID, repository.records[start+2].Diagnostic.Code)
				requireBalance(t, repository, input.AccountID, 6, tc.total)
			}
		})
	}
}

func TestInstalmentRejectionsAndContinuation(t *testing.T) {
	for _, tc := range []struct{ raw, code string }{
		{"10.0001", posting.DiagnosticAmountPrecision},
		{"9223372036854775.808", posting.DiagnosticAmountRange},
		{"-0.001", posting.DiagnosticNegativeAmount},
	} {
		core, repository, input := instalmentFixture(t, 0)
		input.RawAmount = tc.raw
		outcome, err := core.Submit(input)
		require.NoError(t, err)
		assert.Equal(t, domain.OutcomeRejected, outcome.Status)
		requireAttempt(t, repository, 1, input, outcome)
		assert.Equal(t, tc.code, repository.records[3].Diagnostic.Code)
		requireBalance(t, repository, input.AccountID, 6, 0)
		input.RawAmount = "10.000"
		outcome, err = core.Submit(input)
		require.NoError(t, err)
		assert.Equal(t, domain.OutcomeRejected, outcome.Status, "rejected IDs are reserved too")
		assert.Equal(t, posting.DiagnosticDuplicateID, repository.records[6].Diagnostic.Code)
		input.ID = "fresh"
		outcome, err = core.Submit(input)
		require.NoError(t, err)
		assert.Equal(t, domain.OutcomeAccepted, outcome.Status)
		requireBalance(t, repository, input.AccountID, 6, 10000)
	}
}

func TestInstalmentTechnicalFailuresAreAtomic(t *testing.T) {
	t.Run("overflow at last instalment", func(t *testing.T) {
		core, repository, input := instalmentFixture(t, math.MaxInt64-9999)
		before := repository.History()
		outcome, err := core.Submit(input)
		require.ErrorIs(t, err, domain.ErrArithmeticOverflow)
		assert.Equal(t, domain.InputOutcome{}, outcome)
		assert.Equal(t, before, repository.History())
		assert.Equal(t, 1, repository.appends)
	})
	t.Run("append failure and retry", func(t *testing.T) {
		core, repository, input := instalmentFixture(t, 0)
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
		requireBalance(t, repository, input.AccountID, 6, 10000)
	})
}
