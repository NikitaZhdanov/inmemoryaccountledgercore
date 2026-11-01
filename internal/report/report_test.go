package report_test

import (
	"math"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"inmemoryaccountledgercore/internal/domain"
	"inmemoryaccountledgercore/internal/projection"
	"inmemoryaccountledgercore/internal/report"
)

func TestFormatUsesSuppliedOrderAndPrecision(t *testing.T) {
	x, err := domain.NewCurrency("XYZ", 4)
	require.NoError(t, err)
	y, err := domain.NewCurrency("ABC", 0)
	require.NoError(t, err)
	xMoney, err := domain.NewMoney(x, 1)
	require.NoError(t, err)
	yMoney, err := domain.NewMoney(y, -1)
	require.NoError(t, err)
	history := domain.History{Records: []domain.Record{
		{Account: &domain.Account{ID: "X", Currency: x, OpeningBalance: xMoney}},
		{Account: &domain.Account{ID: "Y", Currency: y, OpeningBalance: yMoney}},
		{Close: &domain.DayClose{AccountID: "X", Day: 1}},
		{Close: &domain.DayClose{AccountID: "Y", Day: 1}},
		{Close: &domain.DayClose{AccountID: "X", Day: 2}},
		{Close: &domain.DayClose{AccountID: "Y", Day: 2}},
	}}
	before := cloneHistory(history)
	state, err := projection.Rebuild(history, len(history.Records))
	require.NoError(t, err)
	days := []domain.Day{2, 1}
	accounts := []domain.AccountID{"Y", "X"}
	got, err := report.Format(history, state, days, accounts)
	require.NoError(t, err)
	assert.Equal(t, []string{
		"2 | Y | ABC | -1 | 0 | - | - | -",
		"2 | X | XYZ | 0.0001 | 0.0000 | - | - | -",
		"1 | Y | ABC | -1 | 0 | - | - | -",
		"1 | X | XYZ | 0.0001 | 0.0000 | - | - | -",
	}, strings.Split(strings.TrimSuffix(got, "\n"), "\n")[4:])
	again, err := report.Format(history, state, days, accounts)
	require.NoError(t, err)
	assert.Equal(t, got, again)
	assert.Equal(t, before, history)
	assert.Equal(t, []domain.Day{2, 1}, days)
	assert.Equal(t, []domain.AccountID{"Y", "X"}, accounts)
	unchanged, err := state.Balance("X", 1, projection.BalanceOptions{})
	require.NoError(t, err)
	assert.Equal(t, xMoney, unchanged)
}

func TestFormatRecordedLifecycleAndLinkedFeeHistory(t *testing.T) {
	currency, err := domain.NewCurrency("XYZ", 4)
	require.NoError(t, err)
	money := func(units int64) domain.Money {
		t.Helper()
		value, err := domain.NewMoney(currency, units)
		require.NoError(t, err)
		return value
	}
	history := domain.History{Records: []domain.Record{
		{Account: &domain.Account{ID: "X", Currency: currency, OpeningBalance: money(1)}},
		{Key: "a", Authorization: &domain.AuthorizationTransition{AccountID: "X", AuthorizationID: "Z", BookingDay: 1, State: domain.AuthorizationApproved, Hold: money(1)}},
		{Key: "b", Authorization: &domain.AuthorizationTransition{AccountID: "X", AuthorizationID: "A", BookingDay: 1, State: domain.AuthorizationDeclined, Hold: money(1)}},
		{Key: "settled", Authorization: &domain.AuthorizationTransition{AccountID: "X", AuthorizationID: "Z", BookingDay: 2, State: domain.AuthorizationSettled, Hold: money(1), OriginalHoldKey: "a"}},
		{Key: "charge", Entry: &domain.MonetaryEntry{AccountID: "X", Kind: domain.EntryFeeAssessment, Amount: money(-1), BookingDay: 2, ValueDay: 1, FeeObligationID: "fee", CauseKey: "late"}},
		{Key: "refund", Entry: &domain.MonetaryEntry{AccountID: "X", Kind: domain.EntryFeeRefund, Amount: money(1), BookingDay: 3, ValueDay: 1, FeeObligationID: "fee", OriginalEntryKey: "charge", CauseKey: "correction"}},
		{Key: "renew", Entry: &domain.MonetaryEntry{AccountID: "X", Kind: domain.EntryFeeAdjustment, Amount: money(-1), BookingDay: 4, ValueDay: 1, FeeObligationID: "fee", OriginalEntryKey: "charge", CauseKey: "renewed"}},
		{Attempt: &domain.InputAttempt{ID: "one", Input: domain.InputEvent{ID: "E"}}},
		{Outcome: &domain.InputOutcome{AttemptID: "one", Status: domain.OutcomeDeclined}},
		{Key: "reason", Diagnostic: &domain.Diagnostic{AccountID: "X", AttemptID: "one", ObservationDay: 1, Code: "DECLINE", Reason: "original decision"}},
		{Attempt: &domain.InputAttempt{ID: "two", Input: domain.InputEvent{ID: "E"}}},
		{Outcome: &domain.InputOutcome{AttemptID: "two", Status: domain.OutcomeRejected}},
		{Key: "duplicate", Diagnostic: &domain.Diagnostic{AccountID: "X", AttemptID: "two", ObservationDay: 2, Code: "DUPLICATE", Reason: "reused ID"}},
		{Key: "internal", Diagnostic: &domain.Diagnostic{AccountID: "X", ObservationDay: 2, Code: "NOTICE", Reason: "internal reason"}},
		{Close: &domain.DayClose{AccountID: "X", Day: 1}},
		{Close: &domain.DayClose{AccountID: "X", Day: 2}},
	}}
	before := cloneHistory(history)
	state, err := projection.Rebuild(history, len(history.Records))
	require.NoError(t, err)
	got, err := report.Format(history, state, []domain.Day{1, 2}, []domain.AccountID{"X"})
	require.NoError(t, err)
	rows := strings.Split(strings.TrimSuffix(got, "\n"), "\n")[4:]
	assert.Equal(t, "1 | X | XYZ | 0.0000 | 0.0000 | FEE_ASSESSMENT -0.0001 [key=charge; obligation=fee; booked=2; value=1; cause=late]; FEE_REFUND +0.0001 [key=refund; obligation=fee; booked=3; value=1; original=charge; cause=correction]; FEE_ADJUSTMENT -0.0001 [key=renew; obligation=fee; booked=4; value=1; original=charge; cause=renewed] | Z ACTIVE (hold 0.0001); A DECLINED | E DECLINED [DECLINE]: original decision", rows[0])
	assert.Equal(t, "2 | X | XYZ | 0.0000 | 0.0000 | - | Z SETTLED; A DECLINED | E REJECTED [DUPLICATE]: reused ID; internal [NOTICE]: internal reason", rows[1])
	assert.Equal(t, before, history)
}

func TestFormatReturnsNoPartialReportOnFailure(t *testing.T) {
	currency, err := domain.NewCurrency("XYZ", 4)
	require.NoError(t, err)
	max, err := domain.NewMoney(currency, math.MaxInt64)
	require.NoError(t, err)
	one, err := domain.NewMoney(currency, 1)
	require.NoError(t, err)
	history := domain.History{Records: []domain.Record{
		{Account: &domain.Account{ID: "X", Currency: currency, OpeningBalance: max}},
		{Close: &domain.DayClose{AccountID: "X", Day: 1}},
		{Entry: &domain.MonetaryEntry{AccountID: "X", Amount: one, ValueDay: 2, Kind: domain.EntryCredit}},
	}}
	state, err := projection.Rebuild(history, len(history.Records))
	require.NoError(t, err)
	for _, tc := range []struct {
		name     string
		days     []domain.Day
		accounts []domain.AccountID
		wantErr  error
	}{
		{"unknown account after valid row", []domain.Day{1}, []domain.AccountID{"X", "missing"}, projection.ErrUnknownAccount},
		{"balance overflow after valid row", []domain.Day{1, 2}, []domain.AccountID{"X"}, domain.ErrArithmeticOverflow},
		{"unclosed day", []domain.Day{0}, []domain.AccountID{"X"}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := report.Format(history, state, tc.days, tc.accounts)
			require.Error(t, err)
			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)
			}
			assert.Empty(t, got)
		})
	}
	got, err := report.Format(history, nil, []domain.Day{1}, []domain.AccountID{"X"})
	require.Error(t, err)
	assert.Empty(t, got)
}

func cloneHistory(history domain.History) domain.History {
	copy := domain.History{Records: make([]domain.Record, len(history.Records))}
	for i, record := range history.Records {
		copy.Records[i] = record.Clone()
	}
	return copy
}
