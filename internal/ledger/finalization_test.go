package ledger

import (
	"errors"
	"fmt"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"inmemoryaccountledgercore/internal/domain"
	"inmemoryaccountledgercore/internal/projection"
)

func capitalizationRecords(history domain.History) []domain.Record {
	var records []domain.Record
	for _, record := range history.Records {
		if record.Entry != nil && record.Entry.Kind == domain.EntryCapitalization {
			records = append(records, record)
		}
	}
	return records
}

func TestFinalizationCapitalizationFeeBoundaries(t *testing.T) {
	for _, tc := range []struct {
		base, fee, close int64
	}{
		{-1, 0, 1},
		{-2, 0, 0},
		{-3, -2500, -2501},
	} {
		t.Run(fmt.Sprint(tc.base), func(t *testing.T) {
			core, repository, input := ordinaryFixture(t, 3750)
			require.NoError(t, core.CloseDay(input.AccountID, 5, 5, "prior"))
			appendCloseFixtureEntry(t, repository, input.Currency, "boundary-debit", tc.base-3750, 6, 6)
			before, appends := repository.History(), repository.appends
			require.NoError(t, core.FinalizeDay(input.AccountID, 6, 6, "final"))
			after := repository.History()
			assert.Equal(t, appends+1, repository.appends, "one finalization batch")
			assert.Equal(t, before.Records, after.Records[:len(before.Records)])
			requireBalance(t, repository, input.AccountID, 6, tc.close)
			state := closingProjection(t, repository)
			fee, err := state.FeeNet(input.AccountID, 6)
			require.NoError(t, err)
			assert.Equal(t, tc.fee, fee.MinorUnits())
			accrual, exists := state.Accrual(input.AccountID, 6)
			require.True(t, exists)
			assert.Zero(t, accrual.MinorUnits())
			caps := capitalizationRecords(after)
			require.Len(t, caps, 1)
			assert.Equal(t, int64(2), caps[0].Entry.Amount.MinorUnits())
			assert.Equal(t, domain.Day(6), caps[0].Entry.BookingDay)
			assert.Equal(t, domain.Day(6), caps[0].Entry.ValueDay)
			assert.Equal(t, domain.RecordKey("final"), caps[0].Entry.CauseKey)
			unpaid, err := state.UnpaidAccruals(input.AccountID)
			require.NoError(t, err)
			assert.Empty(t, unpaid)
			require.NoError(t, core.FinalizeDay(input.AccountID, 6, 7, "another-cause"))
			assert.Equal(t, after, repository.History())
			assert.Equal(t, appends+1, repository.appends)
		})
	}
}

func TestFinalizationExcludesCapitalizationFromSameDayInterest(t *testing.T) {
	core, repository, input := ordinaryFixture(t, 3749)
	require.NoError(t, core.CloseDay(input.AccountID, 5, 5, "prior"))
	require.NoError(t, core.FinalizeDay(input.AccountID, 6, 6, "final"))
	requireBalance(t, repository, input.AccountID, 6, 3751)
	state := closingProjection(t, repository)
	for _, day := range []domain.Day{5, 6} {
		accrual, exists := state.Accrual(input.AccountID, day)
		require.True(t, exists)
		assert.Equal(t, int64(1), accrual.MinorUnits())
	}
	caps := capitalizationRecords(repository.History())
	require.Len(t, caps, 1)
	assert.Equal(t, int64(2), caps[0].Entry.Amount.MinorUnits())
}

func TestFinalizationUsesConfiguredChargeIncludingZero(t *testing.T) {
	for _, tc := range []struct {
		code          string
		scale         int
		charge, close int64
	}{
		{"BHD", 3, 0, -1},
		{"XYZ", 4, 1, -2},
	} {
		t.Run(tc.code, func(t *testing.T) {
			currency := fixtureCurrency(t, tc.code, tc.scale)
			repository := &journal{}
			core, err := New(repository, []domain.Account{{ID: "ACC-001", Currency: currency, OpeningBalance: fixtureMoney(t, currency, -1)}},
				map[domain.Currency]domain.Money{currency: fixtureMoney(t, currency, tc.charge)})
			require.NoError(t, err)
			require.NoError(t, core.FinalizeDay("ACC-001", 6, 6, "final"))
			requireBalance(t, repository, "ACC-001", 6, tc.close)
			caps := capitalizationRecords(repository.History())
			require.Len(t, caps, 1, "zero capitalization records the paid window")
			assert.Zero(t, caps[0].Entry.Amount.MinorUnits())
			for _, record := range repository.records {
				assert.Nil(t, record.Diagnostic)
			}
			if tc.charge == 0 {
				assert.Empty(t, feeRecords(repository.History()))
			}
			before, appends := repository.History(), repository.appends
			require.NoError(t, core.FinalizeDay("ACC-001", 6, 6, "final"))
			assert.Equal(t, before, repository.History())
			assert.Equal(t, appends, repository.appends)
		})
	}
}

func TestFinalizationReconcilesExistingOwnFee(t *testing.T) {
	core, repository, input := ordinaryFixture(t, 3750)
	require.NoError(t, core.CloseDay(input.AccountID, 5, 5, "prior"))
	appendCloseFixtureEntry(t, repository, input.Currency, "boundary-debit", -3751, 6, 6)
	require.NoError(t, core.CloseDay(input.AccountID, 6, 6, "ordinary"))
	original := feeRecords(repository.History())[0]
	require.NoError(t, core.FinalizeDay(input.AccountID, 6, 6, "final"))
	fees := feeRecords(repository.History())
	require.Len(t, fees, 2)
	assert.Equal(t, domain.EntryFeeRefund, fees[1].Entry.Kind)
	assert.Equal(t, original.Key, fees[1].Entry.OriginalEntryKey)
	assert.Equal(t, original.Entry.FeeObligationID, fees[1].Entry.FeeObligationID)
	requireBalance(t, repository, input.AccountID, 6, 1)
	assert.Equal(t, []domain.Day{5, 6}, mustClosedDays(t, repository, input.AccountID))
}

func mustClosedDays(t *testing.T, repository Repository, id domain.AccountID) []domain.Day {
	t.Helper()
	days, err := closingProjection(t, repository).ClosedDays(id)
	require.NoError(t, err)
	return days
}

func TestFinalizationFailuresAppendNothing(t *testing.T) {
	for _, tc := range []struct {
		name    string
		opening int64
		future  bool
	}{
		{"capitalization balance overflow", math.MaxInt64, false},
		{"fee underflow", math.MinInt64, false},
		{"future balance overflow", 2500, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			core, repository, input := ordinaryFixture(t, tc.opening)
			if tc.future {
				appendCloseFixtureEntry(t, repository, input.Currency, "future-credit", math.MaxInt64-2500, 7, 7)
			}
			before, appends := repository.History(), repository.appends
			require.ErrorIs(t, core.FinalizeDay(input.AccountID, 6, 6, "final"), domain.ErrArithmeticOverflow)
			assert.Equal(t, before, repository.History())
			assert.Equal(t, appends, repository.appends)
		})
	}
	t.Run("repository failure and retry", func(t *testing.T) {
		core, repository, input := ordinaryFixture(t, -1)
		before, appends := repository.History(), repository.appends
		failure := errors.New("finalization append unavailable")
		repository.appendError = failure
		require.ErrorIs(t, core.FinalizeDay(input.AccountID, 6, 6, "final"), failure)
		assert.Equal(t, before, repository.History())
		assert.Equal(t, appends+1, repository.appends)
		repository.appendError = nil
		require.NoError(t, core.FinalizeDay(input.AccountID, 6, 6, "final"))
		requireBalance(t, repository, input.AccountID, 6, -2501)
		assert.Len(t, capitalizationRecords(repository.History()), 1)
		assert.Len(t, feeRecords(repository.History()), 1)
	})
}

func TestFinalizationAccrualSumOverflow(t *testing.T) {
	// Max-base daily interest is 3,689,348,814,741,910 minor units.
	// 2,500 days fit; 2,501 days overflow the accumulated rounded total.
	for _, priorDays := range []domain.Day{2500, 2501} {
		t.Run(fmt.Sprint(priorDays), func(t *testing.T) {
			core, repository, input := ordinaryFixture(t, math.MaxInt64)
			var records []domain.Record
			for day := domain.Day(1); day <= priorDays; day++ {
				records = append(records, domain.Record{Key: domain.RecordKey(fmt.Sprintf("prior/%d", day)), Close: &domain.DayClose{AccountID: input.AccountID, Day: day}})
			}
			require.NoError(t, repository.Append(domain.Batch{Records: records}))
			before, appends := repository.History(), repository.appends
			require.ErrorIs(t, core.FinalizeDay(input.AccountID, priorDays+1, priorDays+1, "final"), domain.ErrArithmeticOverflow)
			assert.Equal(t, before, repository.History())
			assert.Equal(t, appends, repository.appends)
		})
	}
}

func TestFinalizationRequestValidation(t *testing.T) {
	core, repository, _ := ordinaryFixture(t, 0)
	before, appends := repository.History(), repository.appends
	for _, tc := range []struct {
		day, booking domain.Day
		cause        domain.RecordKey
	}{
		{0, 6, "final"}, {-1, 6, "final"}, {6, 0, "final"}, {6, -1, "final"}, {6, 6, ""},
	} {
		require.ErrorIs(t, core.FinalizeDay("ACC-001", tc.day, tc.booking, tc.cause), ErrInvalidReconciliation)
	}
	require.ErrorIs(t, core.FinalizeDay("unknown", 6, 6, "final"), projection.ErrUnknownAccount)
	assert.Equal(t, before, repository.History())
	assert.Equal(t, appends, repository.appends)
}
