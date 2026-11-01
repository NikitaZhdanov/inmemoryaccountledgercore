package ledger_test

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"inmemoryaccountledgercore/internal/domain"
	"inmemoryaccountledgercore/internal/ledger"
)

// Direct history fixtures record effects, not processing implementations. TASK
// supplies principal/hold values; NUMBERS.md owns the independently derived
// checkpoint balances, fees, and rounded daily accruals.
func projectionHistory(t *testing.T) domain.History {
	t.Helper()
	currencies := canonicalCurrencies(t)
	aed, bhd := currencies[0], currencies[1]
	return domain.History{Records: []domain.Record{
		{Key: "account/AED", Account: &domain.Account{ID: "ACC-001", Currency: aed, OpeningBalance: money(t, aed, 0)}},
		{Key: "account/BHD", Account: &domain.Account{ID: "ACC-002", Currency: bhd, OpeningBalance: money(t, bhd, 0)}},
		{Key: "E1", Entry: &domain.MonetaryEntry{AccountID: "ACC-001", Amount: money(t, aed, 120000), BookingDay: 1, ValueDay: 1, Kind: domain.EntryCredit}},
		{Key: "E2", Entry: &domain.MonetaryEntry{AccountID: "ACC-001", Amount: money(t, aed, -95000), BookingDay: 1, ValueDay: 1, Kind: domain.EntryDebit}},
		{Key: "close/1", Close: &domain.DayClose{AccountID: "ACC-001", Day: 1}},
		{Key: "Auth-A", Authorization: &domain.AuthorizationTransition{AccountID: "ACC-001", AuthorizationID: "Auth-A", AttemptID: "attempt/3", BookingDay: 2, ValueDay: 2, State: domain.AuthorizationApproved, Hold: money(t, aed, 20000)}},
		{Key: "close/2", Close: &domain.DayClose{AccountID: "ACC-001", Day: 2}},
		{Key: "E4", Entry: &domain.MonetaryEntry{AccountID: "ACC-001", Amount: money(t, aed, 40000), BookingDay: 3, ValueDay: 3, Kind: domain.EntryCredit}},
		{Key: "close/3", Close: &domain.DayClose{AccountID: "ACC-001", Day: 3}},
		{Key: "E5", Entry: &domain.MonetaryEntry{AccountID: "ACC-001", Amount: money(t, aed, -18500), BookingDay: 4, ValueDay: 4, Kind: domain.EntrySettlement}},
		{Key: "Auth-A/settled", Authorization: &domain.AuthorizationTransition{AccountID: "ACC-001", AuthorizationID: "Auth-A", AttemptID: "attempt/5", BookingDay: 4, ValueDay: 4, State: domain.AuthorizationSettled, Hold: money(t, aed, 20000), OriginalHoldKey: "Auth-A"}},
		{Key: "E6/outcome", Outcome: &domain.InputOutcome{AttemptID: "attempt/6", Status: domain.OutcomeRejected, DiagnosticKey: "E6/error"}},
		{Key: "E6/error", Diagnostic: &domain.Diagnostic{AccountID: "ACC-001", ObservationDay: 4, AttemptID: "attempt/6", Code: "unknown_authorization"}},
		{Key: "close/4", Close: &domain.DayClose{AccountID: "ACC-001", Day: 4}},
		{Key: "E7", Entry: &domain.MonetaryEntry{AccountID: "ACC-001", Amount: money(t, aed, -62000), BookingDay: 5, ValueDay: 2, Kind: domain.EntryDebit}},
		{Key: "fee/2", Entry: &domain.MonetaryEntry{AccountID: "ACC-001", Amount: money(t, aed, -2500), BookingDay: 5, ValueDay: 2, Kind: domain.EntryFeeAssessment, FeeObligationID: "fee/ACC-001/2"}},
		{Key: "fee/4", Entry: &domain.MonetaryEntry{AccountID: "ACC-001", Amount: money(t, aed, -2500), BookingDay: 5, ValueDay: 4, Kind: domain.EntryFeeAssessment, FeeObligationID: "fee/ACC-001/4"}},
		{Key: "Auth-B", Authorization: &domain.AuthorizationTransition{AccountID: "ACC-001", AuthorizationID: "Auth-B", AttemptID: "attempt/8", BookingDay: 5, ValueDay: 5, State: domain.AuthorizationDeclined, Hold: money(t, aed, 9000)}},
		{Key: "E8/outcome", Outcome: &domain.InputOutcome{AttemptID: "attempt/8", Status: domain.OutcomeDeclined, DiagnosticKey: "E8/error"}},
		{Key: "E8/error", Diagnostic: &domain.Diagnostic{AccountID: "ACC-001", ObservationDay: 5, AttemptID: "attempt/8", Code: "insufficient_available_balance"}},
		{Key: "fee/5", Entry: &domain.MonetaryEntry{AccountID: "ACC-001", Amount: money(t, aed, -2500), BookingDay: 5, ValueDay: 5, Kind: domain.EntryFeeAssessment, FeeObligationID: "fee/ACC-001/5"}},
		{Key: "close/5", Close: &domain.DayClose{AccountID: "ACC-001", Day: 5}},
		{Key: "close/BHD/5", Close: &domain.DayClose{AccountID: "ACC-002", Day: 5}},
		{Key: "E9", Entry: &domain.MonetaryEntry{AccountID: "ACC-001", Amount: money(t, aed, 62000), BookingDay: 6, ValueDay: 2, Kind: domain.EntryReversal, OriginalEntryKey: "E7"}},
		{Key: "refund/2", Entry: &domain.MonetaryEntry{AccountID: "ACC-001", Amount: money(t, aed, 2500), BookingDay: 6, ValueDay: 2, Kind: domain.EntryFeeRefund, FeeObligationID: "fee/ACC-001/2", OriginalEntryKey: "fee/2"}},
		{Key: "refund/4", Entry: &domain.MonetaryEntry{AccountID: "ACC-001", Amount: money(t, aed, 2500), BookingDay: 6, ValueDay: 4, Kind: domain.EntryFeeRefund, FeeObligationID: "fee/ACC-001/4", OriginalEntryKey: "fee/4"}},
		{Key: "refund/5", Entry: &domain.MonetaryEntry{AccountID: "ACC-001", Amount: money(t, aed, 2500), BookingDay: 6, ValueDay: 5, Kind: domain.EntryFeeRefund, FeeObligationID: "fee/ACC-001/5", OriginalEntryKey: "fee/5"}},
		{Key: "E10/1", Entry: &domain.MonetaryEntry{AccountID: "ACC-002", Amount: money(t, bhd, 3333), BookingDay: 5, ValueDay: 5, Kind: domain.EntryInstalment, Instalment: 1, InstalmentCount: 3}},
		{Key: "E10/2", Entry: &domain.MonetaryEntry{AccountID: "ACC-002", Amount: money(t, bhd, 3333), BookingDay: 5, ValueDay: 5, Kind: domain.EntryInstalment, Instalment: 2, InstalmentCount: 3}},
		{Key: "E10/3", Entry: &domain.MonetaryEntry{AccountID: "ACC-002", Amount: money(t, bhd, 3334), BookingDay: 5, ValueDay: 5, Kind: domain.EntryInstalment, Instalment: 3, InstalmentCount: 3}},
		{Key: "capitalization/AED", Entry: &domain.MonetaryEntry{AccountID: "ACC-001", Amount: money(t, aed, 103), BookingDay: 6, ValueDay: 6, Kind: domain.EntryCapitalization}},
		{Key: "capitalization/BHD", Entry: &domain.MonetaryEntry{AccountID: "ACC-002", Amount: money(t, bhd, 8), BookingDay: 6, ValueDay: 6, Kind: domain.EntryCapitalization}},
		{Key: "close/6", Close: &domain.DayClose{AccountID: "ACC-001", Day: 6}},
		{Key: "close/BHD/6", Close: &domain.DayClose{AccountID: "ACC-002", Day: 6}},
	}}
}

func observeThrough(t *testing.T, history domain.History, key domain.RecordKey) *ledger.Projection {
	t.Helper()
	projection, err := ledger.Rebuild(history, recordCountThrough(t, history, key))
	require.NoError(t, err)
	return projection
}

func recordCountThrough(t *testing.T, history domain.History, key domain.RecordKey) int {
	t.Helper()
	for i, record := range history.Records {
		if record.Key == key {
			return i + 1
		}
	}
	t.Fatalf("fixture has no record %q", key)
	return 0
}

func TestBalanceObservations(t *testing.T) {
	history := projectionHistory(t)
	for _, tc := range []struct {
		checkpoint domain.RecordKey
		day        domain.Day
		options    ledger.BalanceOptions
		want       int64
	}{
		{"E2", 1, ledger.BalanceOptions{}, 25000},
		{"E4", 2, ledger.BalanceOptions{}, 25000}, // Future value day excluded.
		{"E4", 3, ledger.BalanceOptions{}, 65000},
		{"close/4", 4, ledger.BalanceOptions{}, 46500},
		{"close/5", 2, ledger.BalanceOptions{ExcludeAllFees: true}, -37000},
		{"close/5", 2, ledger.BalanceOptions{}, -39500},
		{"close/5", 2, ledger.BalanceOptions{ExcludeFeeDay: 2}, -37000},
		{"close/5", 3, ledger.BalanceOptions{}, 500},
		{"close/5", 4, ledger.BalanceOptions{}, -20500},
		{"close/5", 4, ledger.BalanceOptions{ExcludeFeeDay: 4}, -18000}, // Earlier fee stays.
		{"close/5", 5, ledger.BalanceOptions{}, -23000},
		{"close/5", 5, ledger.BalanceOptions{ExcludeFeeDay: 5}, -20500},
		{"close/5", 5, ledger.BalanceOptions{ExcludeAllFees: true}, -15500},
		{"refund/5", 2, ledger.BalanceOptions{}, 25000}, // Booking Day 6 still restores Day 2.
		{"refund/5", 5, ledger.BalanceOptions{}, 46500},
		{"close/6", 6, ledger.BalanceOptions{}, 46603},
	} {
		projection := observeThrough(t, history, tc.checkpoint)
		got, err := projection.Balance("ACC-001", tc.day, tc.options)
		require.NoError(t, err)
		assert.Equal(t, money(t, canonicalCurrencies(t)[0], tc.want), got, "%s/day %d/%+v", tc.checkpoint, tc.day, tc.options)
	}
	before := observeThrough(t, history, "refund/5")
	after := observeThrough(t, history, "E10/3")
	got, err := before.Balance("ACC-002", 5, ledger.BalanceOptions{})
	require.NoError(t, err)
	assert.Equal(t, int64(0), got.MinorUnits())
	got, err = after.Balance("ACC-002", 5, ledger.BalanceOptions{})
	require.NoError(t, err)
	assert.Equal(t, money(t, canonicalCurrencies(t)[1], 10000), got, "E10 is observed after E9 despite its earlier booking day")
}

func TestRecordedHoldsAndAvailability(t *testing.T) {
	history := projectionHistory(t)
	for _, tc := range []struct {
		key                 domain.RecordKey
		day                 domain.Day
		ledger, hold, avail int64
		state               domain.AuthorizationState
	}{
		{"close/2", 2, 25000, 20000, 5000, domain.AuthorizationApproved},
		{"close/3", 3, 65000, 20000, 45000, domain.AuthorizationApproved},
		{"close/4", 4, 46500, 0, 46500, domain.AuthorizationSettled},
		{"Auth-B", 5, -20500, 0, -20500, domain.AuthorizationSettled},
		{"refund/5", 5, 46500, 0, 46500, domain.AuthorizationSettled},
	} {
		p := observeThrough(t, history, tc.key)
		balance, err := p.Balance("ACC-001", tc.day, ledger.BalanceOptions{})
		require.NoError(t, err)
		holds, err := p.ActiveHolds("ACC-001")
		require.NoError(t, err)
		available, err := p.AvailableBalance("ACC-001", tc.day)
		require.NoError(t, err)
		assert.Equal(t, tc.ledger, balance.MinorUnits())
		assert.Equal(t, tc.hold, holds.MinorUnits())
		assert.Equal(t, tc.avail, available.MinorUnits())
		auth, exists := p.Authorization("ACC-001", "Auth-A")
		require.True(t, exists)
		assert.Equal(t, tc.state, auth.Transition.State)
	}
	p := observeThrough(t, history, "refund/5")
	auth, exists := p.Authorization("ACC-001", "Auth-B")
	require.True(t, exists)
	assert.Equal(t, domain.AuthorizationDeclined, auth.Transition.State, "restoration never reapproves a declined request")
	assert.Equal(t, domain.RecordKey("Auth-B"), auth.Key)
	auth.Transition.State = domain.AuthorizationApproved
	again, _ := p.Authorization("ACC-001", "Auth-B")
	assert.Equal(t, domain.AuthorizationDeclined, again.Transition.State)
	_, exists = p.Authorization("ACC-001", "Auth-Z")
	assert.False(t, exists)
	_, exists = p.Authorization("ACC-002", "Auth-A")
	assert.False(t, exists, "authorization lookup is account scoped")
}

func TestAccrualReplacementAndClosedDays(t *testing.T) {
	history := projectionHistory(t)
	for _, tc := range []struct {
		key    domain.RecordKey
		want   []int64
		unpaid int
	}{
		{"close/4", []int64{10, 10, 26, 19}, 4},
		{"close/5", []int64{10, 0, 0, 0, 0}, 5},
		{"refund/5", []int64{10, 10, 26, 19, 19}, 5},
		{"close/6", []int64{10, 10, 26, 19, 19, 19}, 0},
	} {
		p := observeThrough(t, history, tc.key)
		for i, want := range tc.want {
			got, exists := p.Accrual("ACC-001", domain.Day(i+1))
			require.True(t, exists)
			assert.Equal(t, want, got.MinorUnits(), "%s/day %d", tc.key, i+1)
		}
		unpaid, err := p.UnpaidAccruals("ACC-001")
		require.NoError(t, err)
		assert.Len(t, unpaid, tc.unpaid)
		for i, accrual := range unpaid {
			assert.Equal(t, domain.Day(i+1), accrual.Day)
			assert.Equal(t, tc.want[i], accrual.Amount.MinorUnits())
		}
		_, exists := p.Accrual("ACC-001", domain.Day(len(tc.want)+1))
		assert.False(t, exists, "open days do not accrue")
	}
	before := observeThrough(t, history, "refund/5")
	after := observeThrough(t, history, "E10/3")
	old, exists := before.Accrual("ACC-002", 5)
	require.True(t, exists)
	assert.Equal(t, int64(0), old.MinorUnits())
	corrected, exists := after.Accrual("ACC-002", 5)
	require.True(t, exists)
	assert.Equal(t, money(t, canonicalCurrencies(t)[1], 4), corrected)
	// Rebuilding yields replacements, not added accruals; source state survives.
	final := observeThrough(t, history, "close/BHD/6")
	got, exists := final.Accrual("ACC-002", 6)
	require.True(t, exists)
	assert.Equal(t, int64(4), got.MinorUnits())
	unpaid, err := final.UnpaidAccruals("ACC-002")
	require.NoError(t, err)
	assert.Empty(t, unpaid)
	assert.Equal(t, projectionHistory(t), history)
}

func TestProjectionOwnsHistoryAndReturnedContainers(t *testing.T) {
	history := projectionHistory(t)
	p, err := ledger.Rebuild(history, len(history.Records))
	require.NoError(t, err)
	assert.Equal(t, projectionHistory(t), history, "rebuild does not change source records or decisions")
	for i := range history.Records {
		if entry := history.Records[i].Entry; entry != nil {
			entry.Amount = money(t, canonicalCurrencies(t)[0], 0)
		}
		history.Records[i] = domain.Record{}
	}
	got, err := p.Balance("ACC-001", 2, ledger.BalanceOptions{})
	require.NoError(t, err)
	assert.Equal(t, int64(25000), got.MinorUnits())
	days, err := p.ClosedDays("ACC-001")
	require.NoError(t, err)
	assert.Equal(t, []domain.Day{1, 2, 3, 4, 5, 6}, days)
	days[0] = 99
	again, err := p.ClosedDays("ACC-001")
	require.NoError(t, err)
	assert.Equal(t, []domain.Day{1, 2, 3, 4, 5, 6}, again)
	unpaidProjection := observeThrough(t, projectionHistory(t), "close/4")
	unpaid, err := unpaidProjection.UnpaidAccruals("ACC-001")
	require.NoError(t, err)
	unpaid[0].Amount = money(t, canonicalCurrencies(t)[0], 0)
	accrual, _ := unpaidProjection.Accrual("ACC-001", 1)
	assert.Equal(t, int64(10), accrual.MinorUnits())
}

func TestFeeNetAndExclusionIncludeRefundsAndAdjustments(t *testing.T) {
	history := projectionHistory(t)
	for _, tc := range []struct {
		key  domain.RecordKey
		want int64
	}{{"close/5", -2500}, {"refund/5", 0}} {
		p := observeThrough(t, history, tc.key)
		for _, day := range []domain.Day{2, 4, 5} {
			net, err := p.FeeNet("ACC-001", day)
			require.NoError(t, err)
			assert.Equal(t, tc.want, net.MinorUnits())
		}
		zero, err := p.FeeNet("ACC-002", 2)
		require.NoError(t, err)
		assert.Equal(t, money(t, canonicalCurrencies(t)[1], 0), zero)
	}
	aed := canonicalCurrencies(t)[0]
	// Renewed eligibility is represented under the same obligation, after refund.
	base := history.Records[:recordCountThrough(t, history, "refund/5")]
	base = append(base, domain.Record{Key: "adjustment/2", Entry: &domain.MonetaryEntry{AccountID: "ACC-001", Amount: money(t, aed, -2500), BookingDay: 6, ValueDay: 2, Kind: domain.EntryFeeAdjustment, FeeObligationID: "fee/ACC-001/2", OriginalEntryKey: "fee/2"}})
	p, err := ledger.Rebuild(domain.History{Records: base}, len(base))
	require.NoError(t, err)
	net, err := p.FeeNet("ACC-001", 2)
	require.NoError(t, err)
	assert.Equal(t, int64(-2500), net.MinorUnits())
	for _, options := range []ledger.BalanceOptions{{ExcludeAllFees: true}, {ExcludeFeeDay: 2}} {
		balance, err := p.Balance("ACC-001", 2, options)
		require.NoError(t, err)
		assert.Equal(t, int64(25000), balance.MinorUnits(), "exclude every effect of the obligation together")
	}
}

func TestCapitalizationDoesNotEarnSameDayInterest(t *testing.T) {
	aed := canonicalCurrencies(t)[0]
	history := domain.History{Records: []domain.Record{
		{Key: "account", Account: &domain.Account{ID: "A", Currency: aed, OpeningBalance: money(t, aed, 0)}},
		{Key: "credit", Entry: &domain.MonetaryEntry{AccountID: "A", Amount: money(t, aed, 3749), BookingDay: 1, ValueDay: 1, Kind: domain.EntryCredit}},
		{Key: "close/1", Close: &domain.DayClose{AccountID: "A", Day: 1}},
		{Key: "capitalization", Entry: &domain.MonetaryEntry{AccountID: "A", Amount: money(t, aed, 2), BookingDay: 2, ValueDay: 2, Kind: domain.EntryCapitalization}},
		{Key: "close/2", Close: &domain.DayClose{AccountID: "A", Day: 2}},
		{Key: "close/3", Close: &domain.DayClose{AccountID: "A", Day: 3}},
	}}
	p, err := ledger.Rebuild(history, len(history.Records))
	require.NoError(t, err)
	for i, want := range []int64{1, 1, 2} {
		got, exists := p.Accrual("A", domain.Day(i+1))
		require.True(t, exists)
		assert.Equal(t, want, got.MinorUnits())
	}
	paidDay, err := p.Balance("A", 2, ledger.BalanceOptions{})
	require.NoError(t, err)
	assert.Equal(t, int64(3751), paidDay.MinorUnits())
	unpaid, err := p.UnpaidAccruals("A")
	require.NoError(t, err)
	assert.Equal(t, []ledger.DailyAccrual{{Day: 3, Amount: money(t, aed, 2)}}, unpaid)
}

func TestProjectionBoundariesAndErrors(t *testing.T) {
	aed := canonicalCurrencies(t)[0]
	account := domain.Record{Key: "account", Account: &domain.Account{ID: "A", Currency: aed, OpeningBalance: money(t, aed, 1)}}
	history := domain.History{Records: []domain.Record{account,
		{Key: "future", Entry: &domain.MonetaryEntry{AccountID: "A", Amount: money(t, aed, 1), BookingDay: 1, ValueDay: 2, Kind: domain.EntryCredit}},
		{Key: "hold/1", Authorization: &domain.AuthorizationTransition{AccountID: "A", AuthorizationID: "one", State: domain.AuthorizationApproved, Hold: money(t, aed, 1)}},
		{Key: "hold/2", Authorization: &domain.AuthorizationTransition{AccountID: "A", AuthorizationID: "two", State: domain.AuthorizationApproved, Hold: money(t, aed, 1)}},
		{Key: "close/2", Close: &domain.DayClose{AccountID: "A", Day: 2}},
		{Key: "close/1", Close: &domain.DayClose{AccountID: "A", Day: 1}},
		{Key: "close/1/repeat", Close: &domain.DayClose{AccountID: "A", Day: 1}},
	}}
	p, err := ledger.Rebuild(history, len(history.Records))
	require.NoError(t, err)
	balance, err := p.Balance("A", 1, ledger.BalanceOptions{})
	require.NoError(t, err)
	assert.Equal(t, int64(1), balance.MinorUnits(), "opening money included before first value-dated posting")
	available, err := p.AvailableBalance("A", 2)
	require.NoError(t, err)
	assert.Equal(t, int64(0), available.MinorUnits(), "all active holds are summed")
	days, err := p.ClosedDays("A")
	require.NoError(t, err)
	assert.Equal(t, []domain.Day{1, 2}, days)
	for _, count := range []int{-1, len(history.Records) + 1} {
		got, err := ledger.Rebuild(history, count)
		require.ErrorIs(t, err, ledger.ErrInvalidObservation)
		assert.Nil(t, got)
	}
	empty, err := ledger.Rebuild(history, 0)
	require.NoError(t, err)
	_, err = empty.Balance("A", 1, ledger.BalanceOptions{})
	require.ErrorIs(t, err, ledger.ErrUnknownAccount)
	_, err = p.Balance("unknown", 1, ledger.BalanceOptions{})
	require.ErrorIs(t, err, ledger.ErrUnknownAccount)
	_, err = p.AvailableBalance("unknown", 1)
	require.ErrorIs(t, err, ledger.ErrUnknownAccount)
	_, err = p.ActiveHolds("unknown")
	require.ErrorIs(t, err, ledger.ErrUnknownAccount)
	_, err = p.FeeNet("unknown", 1)
	require.ErrorIs(t, err, ledger.ErrUnknownAccount)
	_, err = p.ClosedDays("unknown")
	require.ErrorIs(t, err, ledger.ErrUnknownAccount)
	_, err = p.UnpaidAccruals("unknown")
	require.ErrorIs(t, err, ledger.ErrUnknownAccount)
}

func TestProjectionArithmeticFailures(t *testing.T) {
	currencies := canonicalCurrencies(t)
	aed, bhd := currencies[0], currencies[1]
	account := domain.Record{Key: "account", Account: &domain.Account{ID: "A", Currency: aed, OpeningBalance: money(t, aed, 0)}}
	for _, tc := range []struct {
		name    string
		records []domain.Record
		wantErr error
	}{
		{"fee net underflow", []domain.Record{
			{Key: "fee/1", Entry: &domain.MonetaryEntry{AccountID: "A", ValueDay: 1, Kind: domain.EntryFeeAssessment, Amount: money(t, aed, math.MinInt64)}},
			{Key: "fee/adjust", Entry: &domain.MonetaryEntry{AccountID: "A", ValueDay: 1, Kind: domain.EntryFeeAdjustment, Amount: money(t, aed, -1)}},
		}, domain.ErrArithmeticOverflow},
		{"active hold overflow", []domain.Record{
			{Key: "hold/1", Authorization: &domain.AuthorizationTransition{AccountID: "A", AuthorizationID: "one", State: domain.AuthorizationApproved, Hold: money(t, aed, math.MaxInt64)}},
			{Key: "hold/2", Authorization: &domain.AuthorizationTransition{AccountID: "A", AuthorizationID: "two", State: domain.AuthorizationApproved, Hold: money(t, aed, 1)}},
		}, domain.ErrArithmeticOverflow},
		{"balance intermediate overflow on closed day", []domain.Record{
			{Key: "max", Entry: &domain.MonetaryEntry{AccountID: "A", ValueDay: 1, Amount: money(t, aed, math.MaxInt64)}},
			{Key: "plus", Entry: &domain.MonetaryEntry{AccountID: "A", ValueDay: 1, Amount: money(t, aed, 1)}},
			{Key: "minus", Entry: &domain.MonetaryEntry{AccountID: "A", ValueDay: 1, Amount: money(t, aed, -1)}},
			{Key: "close", Close: &domain.DayClose{AccountID: "A", Day: 1}},
		}, domain.ErrArithmeticOverflow},
		{"closed balance currency mismatch", []domain.Record{
			{Key: "foreign", Entry: &domain.MonetaryEntry{AccountID: "A", ValueDay: 1, Amount: money(t, bhd, 1)}},
			{Key: "close", Close: &domain.DayClose{AccountID: "A", Day: 1}},
		}, domain.ErrCurrencyMismatch},
		{"unknown posting account", []domain.Record{{Key: "entry", Entry: &domain.MonetaryEntry{AccountID: "unknown", Amount: money(t, aed, 1)}}}, ledger.ErrUnknownAccount},
		{"unknown close account", []domain.Record{{Key: "close", Close: &domain.DayClose{AccountID: "unknown", Day: 1}}}, ledger.ErrUnknownAccount},
		{"unknown hold account", []domain.Record{{Key: "hold", Authorization: &domain.AuthorizationTransition{AccountID: "unknown", State: domain.AuthorizationApproved, Hold: money(t, aed, 1)}}}, ledger.ErrUnknownAccount},
		{"unknown authorization account precedes later fee overflow", []domain.Record{
			{Key: "hold", Authorization: &domain.AuthorizationTransition{AccountID: "unknown", State: domain.AuthorizationDeclined, Hold: money(t, aed, 1)}},
			{Key: "fee/1", Entry: &domain.MonetaryEntry{AccountID: "A", ValueDay: 1, Kind: domain.EntryFeeAssessment, Amount: money(t, aed, math.MinInt64)}},
			{Key: "fee/adjust", Entry: &domain.MonetaryEntry{AccountID: "A", ValueDay: 1, Kind: domain.EntryFeeAdjustment, Amount: money(t, aed, -1)}},
		}, ledger.ErrUnknownAccount},
	} {
		t.Run(tc.name, func(t *testing.T) {
			history := domain.History{Records: append([]domain.Record{account}, tc.records...)}
			got, err := ledger.Rebuild(history, len(history.Records))
			require.ErrorIs(t, err, tc.wantErr)
			assert.Nil(t, got, "no partial derived state escapes")
		})
	}
	// Query-time failures also return no partial money. No repository is involved,
	// so these operations cannot append diagnostics, rejections, or partial batches.
	history := domain.History{Records: []domain.Record{account,
		{Key: "minimum", Entry: &domain.MonetaryEntry{AccountID: "A", ValueDay: 1, Amount: money(t, aed, math.MinInt64)}},
		{Key: "hold", Authorization: &domain.AuthorizationTransition{AccountID: "A", AuthorizationID: "one", State: domain.AuthorizationApproved, Hold: money(t, aed, 1)}},
	}}
	p, err := ledger.Rebuild(history, len(history.Records))
	require.NoError(t, err)
	got, err := p.AvailableBalance("A", 1)
	require.ErrorIs(t, err, domain.ErrArithmeticOverflow)
	assert.Equal(t, domain.Money{}, got)
	history.Records = append(history.Records, domain.Record{Key: "minus", Entry: &domain.MonetaryEntry{AccountID: "A", ValueDay: 1, Amount: money(t, aed, -1)}})
	p, err = ledger.Rebuild(history, len(history.Records))
	require.NoError(t, err)
	got, err = p.Balance("A", 1, ledger.BalanceOptions{})
	require.ErrorIs(t, err, domain.ErrArithmeticOverflow)
	assert.Equal(t, domain.Money{}, got)
	got, err = p.AvailableBalance("A", 1)
	require.ErrorIs(t, err, domain.ErrArithmeticOverflow)
	assert.Equal(t, domain.Money{}, got)
}
