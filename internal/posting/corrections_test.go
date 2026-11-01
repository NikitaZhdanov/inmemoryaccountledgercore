package posting

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"inmemoryaccountledgercore/internal/domain"
)

func TestPrincipalCorrectionIdempotenceAndConflictingCause(t *testing.T) {
	currency := canonicalCurrencies(t)[0]
	original := domain.Record{Key: "entry/E7", Entry: &domain.MonetaryEntry{
		AccountID: "ACC-001", Amount: money(t, currency, -62000),
		BookingDay: 5, ValueDay: 2, Kind: domain.EntryDebit,
	}}
	attempt := domain.InputAttempt{ID: "attempt/2", Input: domain.InputEvent{
		ID: "E9", Kind: domain.InputReversal, AccountID: "ACC-001", Currency: currency,
		BookingDay: 6, ValueDay: 2, ReversesEventID: "E7",
	}}
	history := domain.History{Records: []domain.Record{original}}
	batch, err := principalCorrection(history, attempt, original)
	require.NoError(t, err)
	require.Len(t, batch.Records, 1)
	assert.Equal(t, domain.MonetaryEntry{
		AccountID: "ACC-001", Amount: money(t, currency, 62000), BookingDay: 6, ValueDay: 2,
		Kind: domain.EntryReversal, SourceAttemptID: attempt.ID,
		CauseKey: domain.RecordKey(attempt.ID), OriginalEntryKey: original.Key,
	}, *batch.Records[0].Entry)
	assert.Equal(t, []domain.Record{original}, history.Records)
	history.Records = append(history.Records, batch.Records...)
	for range 2 {
		repeated, err := principalCorrection(history, attempt, original)
		require.NoError(t, err)
		assert.Empty(t, repeated.Records, "repeated internal correction has no records to append")
	}
	attempt.ID = "attempt/3"
	conflict, err := principalCorrection(history, attempt, original)
	require.ErrorIs(t, err, errAlreadyReversed)
	assert.NotErrorIs(t, err, ErrUnsupportedInput)
	assert.Empty(t, conflict.Records)
}

func TestPrincipalCorrectionUnrepresentableInverse(t *testing.T) {
	currency := canonicalCurrencies(t)[0]
	original := domain.Record{Key: "debit", Entry: &domain.MonetaryEntry{
		AccountID: "ACC-001", Amount: money(t, currency, math.MinInt64), Kind: domain.EntryDebit,
	}}
	batch, err := principalCorrection(domain.History{}, domain.InputAttempt{ID: "correction"}, original)
	require.ErrorIs(t, err, domain.ErrArithmeticOverflow)
	assert.Empty(t, batch.Records)
}
