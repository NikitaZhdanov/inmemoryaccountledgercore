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

func appendCloseFixtureEntry(t *testing.T, repository *journal, currency domain.Currency, key domain.RecordKey, minor int64, booking, value domain.Day) {
	t.Helper()
	kind := domain.EntryCredit
	if minor < 0 {
		kind = domain.EntryDebit
	}
	require.NoError(t, repository.Append(domain.Batch{Records: []domain.Record{{
		Key: key, Entry: &domain.MonetaryEntry{
			AccountID: "ACC-001", Amount: fixtureMoney(t, currency, minor),
			BookingDay: booking, ValueDay: value, Kind: kind,
		},
	}}}))
}

func closingProjection(t *testing.T, repository Repository) *Projection {
	t.Helper()
	history := repository.History()
	projection, err := Rebuild(history, len(history.Records))
	require.NoError(t, err)
	return projection
}

func feeRecords(history domain.History) []domain.Record {
	var records []domain.Record
	for _, record := range history.Records {
		if record.Entry != nil && isFee(record.Entry.Kind) {
			records = append(records, record)
		}
	}
	return records
}

func requireClosingState(t *testing.T, repository Repository, balances, accruals []int64) {
	t.Helper()
	projection := closingProjection(t, repository)
	unpaid, err := projection.UnpaidAccruals("ACC-001")
	require.NoError(t, err)
	require.Len(t, unpaid, len(accruals))
	for i, want := range balances {
		day := domain.Day(i + 1)
		balance, err := projection.Balance("ACC-001", day, BalanceOptions{})
		require.NoError(t, err)
		assert.Equal(t, want, balance.MinorUnits(), "Day %d balance", day)
		available, err := projection.AvailableBalance("ACC-001", day)
		require.NoError(t, err)
		assert.Equal(t, balance, available, "unpaid accrual is not spendable")
		if i < len(accruals) {
			accrual, exists := projection.Accrual("ACC-001", day)
			require.True(t, exists)
			assert.Equal(t, accruals[i], accrual.MinorUnits(), "Day %d accrual", day)
			assert.Equal(t, DailyAccrual{Day: day, Amount: accrual}, unpaid[i])
		}
	}
}

// Controlled principal fixtures isolate P08 from P09's automatic submission
// reconciliation. Expected fees, balances, and accruals come from NUMBERS.md.
func TestChronologicalClosingAndAccrualRestoration(t *testing.T) {
	core, repository, input := ordinaryFixture(t, 25000)
	for day := domain.Day(1); day <= 4; day++ {
		if day == 3 {
			appendCloseFixtureEntry(t, repository, input.Currency, "E4", 40000, 3, 3)
		}
		if day == 4 {
			appendCloseFixtureEntry(t, repository, input.Currency, "E5", -18500, 4, 4)
		}
		require.NoError(t, core.CloseDay(input.AccountID, day, day, domain.RecordKey(fmt.Sprintf("checkpoint/%d", day))))
	}
	requireClosingState(t, repository, []int64{25000, 25000, 65000, 46500}, []int64{10, 10, 26, 19})
	appendCloseFixtureEntry(t, repository, input.Currency, "E7", -62000, 5, 2)
	before := repository.History()
	require.NoError(t, core.ReconcileClosedDays(input.AccountID, 2, 5, "E7"))
	assert.Equal(t, before.Records, repository.History().Records[:len(before.Records)])
	requireClosingState(t, repository, []int64{25000, -39500, 500, -20500, -20500}, []int64{10, 0, 0, 0})
	projection := closingProjection(t, repository)
	_, exists := projection.Accrual(input.AccountID, 5)
	assert.False(t, exists, "reconciliation does not close open Day 5")
	fees := feeRecords(repository.History())
	require.Len(t, fees, 2)
	assert.Equal(t, domain.Day(2), fees[0].Entry.ValueDay)
	assert.Equal(t, domain.Day(4), fees[1].Entry.ValueDay)

	require.NoError(t, core.CloseDay(input.AccountID, 5, 5, "checkpoint/5"))
	requireClosingState(t, repository, []int64{25000, -39500, 500, -20500, -23000}, []int64{10, 0, 0, 0, 0})
	fees = feeRecords(repository.History())
	require.Len(t, fees, 3)
	for i, day := range []domain.Day{2, 4, 5} {
		entry := fees[i].Entry
		assert.Equal(t, domain.EntryFeeAssessment, entry.Kind)
		assert.Equal(t, day, entry.ValueDay)
		assert.Equal(t, domain.Day(5), entry.BookingDay)
		assert.Equal(t, int64(-2500), entry.Amount.MinorUnits())
		assert.NotEmpty(t, entry.FeeObligationID)
		assert.Empty(t, entry.OriginalEntryKey)
	}
	assert.NotEqual(t, fees[0].Entry.FeeObligationID, fees[1].Entry.FeeObligationID)
	assert.NotEqual(t, fees[1].Entry.FeeObligationID, fees[2].Entry.FeeObligationID)
	unchanged := repository.History()
	appends := repository.appends
	for range 2 {
		require.NoError(t, core.ReconcileClosedDays(input.AccountID, 2, 5, "E7"))
		require.NoError(t, core.CloseDay(input.AccountID, 5, 5, "checkpoint/5"))
	}
	assert.Equal(t, unchanged, repository.History())
	assert.Equal(t, appends, repository.appends)

	appendCloseFixtureEntry(t, repository, input.Currency, "E9", 62000, 6, 2)
	before = repository.History()
	require.NoError(t, core.ReconcileClosedDays(input.AccountID, 2, 6, "E9"))
	assert.Equal(t, before.Records, repository.History().Records[:len(before.Records)])
	requireClosingState(t, repository, []int64{25000, 25000, 65000, 46500, 46500}, []int64{10, 10, 26, 19, 19})
	after := repository.History()
	allFees := feeRecords(after)
	require.Len(t, allFees, 6)
	for i, original := range fees {
		refund := allFees[i+3].Entry
		assert.Equal(t, domain.MonetaryEntry{
			AccountID: input.AccountID, Amount: fixtureMoney(t, input.Currency, 2500),
			BookingDay: 6, ValueDay: original.Entry.ValueDay, Kind: domain.EntryFeeRefund,
			CauseKey: "E9", OriginalEntryKey: original.Key, FeeObligationID: original.Entry.FeeObligationID,
		}, *refund)
	}
	appends = repository.appends
	require.NoError(t, core.ReconcileClosedDays(input.AccountID, 2, 6, "E9"))
	require.NoError(t, core.CloseDay(input.AccountID, 5, 6, "checkpoint/5"))
	assert.Equal(t, after, repository.History())
	assert.Equal(t, appends, repository.appends)
	for _, record := range after.Records {
		assert.Nil(t, record.Diagnostic)
		if record.Entry != nil {
			assert.True(t, isFee(record.Entry.Kind) || record.Key == "E4" || record.Key == "E5" || record.Key == "E7" || record.Key == "E9", "no daily accrual postings")
		}
	}
}

func TestFeeSelfEligibilityRenewalAndEarlierFeeCarry(t *testing.T) {
	core, repository, input := ordinaryFixture(t, -1)
	appendCloseFixtureEntry(t, repository, input.Currency, "later-credit", 1, 3, 3)
	// Record the later close first: CloseDay must still reconcile in day order
	// when an earlier close is subsequently introduced.
	require.NoError(t, core.CloseDay(input.AccountID, 3, 3, "close/3"))
	assert.Empty(t, feeRecords(repository.History()))
	require.NoError(t, core.CloseDay(input.AccountID, 2, 3, "close/2"))
	fees := feeRecords(repository.History())
	require.Len(t, fees, 2)
	assert.Equal(t, domain.Day(2), fees[0].Entry.ValueDay)
	assert.Equal(t, domain.Day(3), fees[1].Entry.ValueDay)
	requireBalance(t, repository, input.AccountID, 2, -2501)
	requireBalance(t, repository, input.AccountID, 3, -5000)

	for cycle := 1; cycle <= 2; cycle++ {
		credit := domain.RecordKey(fmt.Sprintf("restore/%d", cycle))
		appendCloseFixtureEntry(t, repository, input.Currency, credit, 1, 4, 2)
		require.NoError(t, core.ReconcileClosedDays(input.AccountID, 2, 4, credit))
		// Eligibility is exactly zero without its own fee. Each refund must
		// also remove the earlier fee before testing the following day's zero.
		requireBalance(t, repository, input.AccountID, 2, 0)
		requireBalance(t, repository, input.AccountID, 3, 1)
		debit := domain.RecordKey(fmt.Sprintf("renew/%d", cycle))
		appendCloseFixtureEntry(t, repository, input.Currency, debit, -1, 5, 2)
		require.NoError(t, core.ReconcileClosedDays(input.AccountID, 2, 5, debit))
		requireBalance(t, repository, input.AccountID, 2, -2501)
		requireBalance(t, repository, input.AccountID, 3, -5000)
		all := feeRecords(repository.History())
		for i, record := range all[2:] {
			original := fees[i%2]
			assert.Equal(t, original.Key, record.Entry.OriginalEntryKey)
			assert.Equal(t, original.Entry.FeeObligationID, record.Entry.FeeObligationID)
			assert.Equal(t, original.Entry.ValueDay, record.Entry.ValueDay)
			if (i/2)%2 == 0 {
				assert.Equal(t, domain.EntryFeeRefund, record.Entry.Kind)
				assert.Equal(t, int64(2500), record.Entry.Amount.MinorUnits())
			} else {
				assert.Equal(t, domain.EntryFeeAdjustment, record.Entry.Kind)
				assert.Equal(t, int64(-2500), record.Entry.Amount.MinorUnits())
			}
		}
		before := repository.History()
		appends := repository.appends
		require.NoError(t, core.ReconcileClosedDays(input.AccountID, 2, 5, debit))
		assert.Equal(t, before, repository.History())
		assert.Equal(t, appends, repository.appends)
	}
	all := feeRecords(repository.History())
	require.Len(t, all, 10)
	keys := make(map[domain.RecordKey]bool)
	for _, record := range repository.History().Records {
		assert.False(t, keys[record.Key], "unique record key %q", record.Key)
		keys[record.Key] = true
	}
}

func TestClosingConfiguredCurrenciesAndInterestTies(t *testing.T) {
	for _, tc := range []struct {
		name, code          string
		scale               int
		opening, charge     int64
		wantBalance, accrue int64
		wantFees            int
	}{
		{"BHD explicit no fee", "BHD", 3, -1, 0, -1, 0, 0},
		{"custom configured fee", "XYZ", 4, -1, 1, -2, 0, 1},
		{"zero balance", "AED", 2, 0, 2500, 0, 0, 0},
		{"AED even tie down", "AED", 2, 1250, 2500, 1250, 0, 0},
		{"AED odd tie up", "AED", 2, 3750, 2500, 3750, 2, 0},
		{"BHD even tie down", "BHD", 3, 1250, 0, 1250, 0, 0},
		{"BHD odd tie up", "BHD", 3, 3750, 0, 3750, 2, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			currency := fixtureCurrency(t, tc.code, tc.scale)
			repository := &journal{}
			core, err := New(repository, []domain.Account{{ID: "ACC-001", Currency: currency, OpeningBalance: fixtureMoney(t, currency, tc.opening)}},
				map[domain.Currency]domain.Money{currency: fixtureMoney(t, currency, tc.charge)})
			require.NoError(t, err)
			require.NoError(t, core.CloseDay("ACC-001", 1, 1, "close/1"))
			requireClosingState(t, repository, []int64{tc.wantBalance}, []int64{tc.accrue})
			history := repository.History()
			fees := feeRecords(history)
			assert.Len(t, fees, tc.wantFees)
			for _, record := range history.Records {
				assert.Nil(t, record.Diagnostic)
				if record.Entry != nil {
					assert.Equal(t, currency, record.Entry.Amount.Currency())
				}
			}
			appends := repository.appends
			require.NoError(t, core.CloseDay("ACC-001", 1, 1, "close/1"))
			require.NoError(t, core.ReconcileClosedDays("ACC-001", 1, 1, "close/1"))
			assert.Equal(t, history, repository.History())
			assert.Equal(t, appends, repository.appends)
		})
	}
}

func TestClosingValidationAndAtomicFailures(t *testing.T) {
	for _, tc := range []struct {
		name      string
		account   domain.AccountID
		day, book domain.Day
		cause     domain.RecordKey
		want      error
	}{
		{"unknown account", "missing", 1, 1, "cause", ErrUnknownAccount},
		{"nonpositive day", "ACC-001", 0, 1, "cause", ErrInvalidReconciliation},
		{"missing booking day", "ACC-001", 1, 0, "cause", ErrInvalidReconciliation},
		{"missing cause", "ACC-001", 1, 1, "", ErrInvalidReconciliation},
	} {
		t.Run(tc.name, func(t *testing.T) {
			core, repository, _ := ordinaryFixture(t, 0)
			before, appends := repository.History(), repository.appends
			require.ErrorIs(t, core.CloseDay(tc.account, tc.day, tc.book, tc.cause), tc.want)
			require.ErrorIs(t, core.ReconcileClosedDays(tc.account, tc.day, tc.book, tc.cause), tc.want)
			assert.Equal(t, before, repository.History())
			assert.Equal(t, appends, repository.appends)
		})
	}
	t.Run("open days remain open", func(t *testing.T) {
		core, repository, input := ordinaryFixture(t, -1)
		before, appends := repository.History(), repository.appends
		require.NoError(t, core.ReconcileClosedDays(input.AccountID, 1, 5, "correction"))
		assert.Equal(t, before, repository.History())
		assert.Equal(t, appends, repository.appends)
	})
	t.Run("only requested range is reconciled", func(t *testing.T) {
		core, repository, input := ordinaryFixture(t, -1)
		require.NoError(t, repository.Append(domain.Batch{Records: []domain.Record{
			{Key: "fixture/close/1", Close: &domain.DayClose{AccountID: input.AccountID, Day: 1}},
			{Key: "fixture/close/2", Close: &domain.DayClose{AccountID: input.AccountID, Day: 2}},
		}}))
		require.NoError(t, core.ReconcileClosedDays(input.AccountID, 2, 5, "correction"))
		fees := feeRecords(repository.History())
		require.Len(t, fees, 1)
		assert.Equal(t, domain.Day(2), fees[0].Entry.ValueDay)
		requireBalance(t, repository, input.AccountID, 1, -1)
		requireBalance(t, repository, input.AccountID, 2, -2501)
	})
	t.Run("fee and close append together or neither", func(t *testing.T) {
		core, repository, input := ordinaryFixture(t, -1)
		before := repository.History()
		failure := errors.New("close append unavailable")
		repository.appendError = failure
		require.ErrorIs(t, core.CloseDay(input.AccountID, 1, 1, "close/1"), failure)
		assert.Equal(t, before, repository.History())
		repository.appendError = nil
		require.NoError(t, core.CloseDay(input.AccountID, 1, 1, "close/1"))
		assert.Len(t, repository.records, len(before.Records)+2)
		requireBalance(t, repository, input.AccountID, 1, -2501)
	})
	t.Run("fee underflow", func(t *testing.T) {
		core, repository, input := ordinaryFixture(t, math.MinInt64)
		before, appends := repository.History(), repository.appends
		require.ErrorIs(t, core.CloseDay(input.AccountID, 1, 1, "close/1"), domain.ErrArithmeticOverflow)
		assert.Equal(t, before, repository.History())
		assert.Equal(t, appends, repository.appends)
	})
	t.Run("later value day underflow", func(t *testing.T) {
		core, repository, input := ordinaryFixture(t, -1)
		appendCloseFixtureEntry(t, repository, input.Currency, "future-debit", math.MinInt64+1, 3, 3)
		before, appends := repository.History(), repository.appends
		require.ErrorIs(t, core.CloseDay(input.AccountID, 2, 2, "close/2"), domain.ErrArithmeticOverflow)
		assert.Equal(t, before, repository.History())
		assert.Equal(t, appends, repository.appends)
	})
	t.Run("refund append failure and retry", func(t *testing.T) {
		core, repository, input := ordinaryFixture(t, -1)
		require.NoError(t, core.CloseDay(input.AccountID, 1, 1, "close/1"))
		appendCloseFixtureEntry(t, repository, input.Currency, "restore", 1, 2, 1)
		before := repository.History()
		failure := errors.New("refund append unavailable")
		repository.appendError = failure
		require.ErrorIs(t, core.ReconcileClosedDays(input.AccountID, 1, 2, "restore"), failure)
		assert.Equal(t, before, repository.History())
		repository.appendError = nil
		require.NoError(t, core.ReconcileClosedDays(input.AccountID, 1, 2, "restore"))
		requireBalance(t, repository, input.AccountID, 1, 0)
		assert.Len(t, feeRecords(repository.History()), 2)
	})
}

func TestClosingScheduleIsSuppliedByCaller(t *testing.T) {
	// Day 6 is the canonical capitalization day; Day 7 is outside its window.
	// Neither ordinal limits ordinary closing for a different supplied schedule.
	for _, day := range []domain.Day{6, 7} {
		t.Run(fmt.Sprintf("day %d", day), func(t *testing.T) {
			core, repository, input := ordinaryFixture(t, -1)
			require.NoError(t, core.CloseDay(input.AccountID, day, day, "ordinary-close"))
			fees := feeRecords(repository.History())
			require.Len(t, fees, 1)
			assert.Equal(t, day, fees[0].Entry.ValueDay)
			assert.Equal(t, day, fees[0].Entry.BookingDay)
			requireBalance(t, repository, input.AccountID, day, -2501)
			projection := closingProjection(t, repository)
			days, err := projection.ClosedDays(input.AccountID)
			require.NoError(t, err)
			assert.Equal(t, []domain.Day{day}, days)
			accrual, exists := projection.Accrual(input.AccountID, day)
			require.True(t, exists)
			assert.Zero(t, accrual.MinorUnits())

			appendCloseFixtureEntry(t, repository, input.Currency, "restore", 1, day+1, day)
			// A range starting before the canonical cutoff must include this day.
			require.NoError(t, core.ReconcileClosedDays(input.AccountID, 1, day+1, "restore"))
			fees = feeRecords(repository.History())
			require.Len(t, fees, 2)
			assert.Equal(t, domain.EntryFeeRefund, fees[1].Entry.Kind)
			assert.Equal(t, fees[0].Key, fees[1].Entry.OriginalEntryKey)
			requireBalance(t, repository, input.AccountID, day, 0)
			before, appends := repository.History(), repository.appends
			require.NoError(t, core.ReconcileClosedDays(input.AccountID, day, day+1, "restore"))
			require.NoError(t, core.CloseDay(input.AccountID, day, day+1, "ordinary-close"))
			assert.Equal(t, before, repository.History())
			assert.Equal(t, appends, repository.appends)
		})
	}
}

type interruptedClosingRepository struct {
	journal
	failAt  int
	failure error
}

func (r *interruptedClosingRepository) Append(batch domain.Batch) error {
	if r.appends+1 == r.failAt {
		r.appends++
		return r.failure
	}
	return r.journal.Append(batch)
}

func TestReconciliationFailurePreservesEarlierCommittedDays(t *testing.T) {
	currency := fixtureCurrency(t, "AED", 2)
	failure := errors.New("second day append unavailable")
	repository := &interruptedClosingRepository{failure: failure}
	core, err := New(repository, []domain.Account{{ID: "ACC-001", Currency: currency, OpeningBalance: fixtureMoney(t, currency, -1)}},
		map[domain.Currency]domain.Money{currency: fixtureMoney(t, currency, 2500)})
	require.NoError(t, err)
	appendCloseFixtureEntry(t, &repository.journal, currency, "later-credit", 1, 3, 3)
	require.NoError(t, repository.Append(domain.Batch{Records: []domain.Record{
		{Key: "fixture/close/3", Close: &domain.DayClose{AccountID: "ACC-001", Day: 3}},
		{Key: "fixture/close/2", Close: &domain.DayClose{AccountID: "ACC-001", Day: 2}},
	}}))
	before := repository.History()
	repository.failAt = repository.appends + 2
	require.ErrorIs(t, core.ReconcileClosedDays("ACC-001", 2, 5, "correction"), failure)
	after := repository.History()
	assert.Equal(t, before.Records, after.Records[:len(before.Records)])
	require.Len(t, after.Records, len(before.Records)+1)
	fees := feeRecords(after)
	require.Len(t, fees, 1)
	assert.Equal(t, domain.Day(2), fees[0].Entry.ValueDay)
	assert.Nil(t, after.Records[len(after.Records)-1].Diagnostic)
	// Explicitly repeating the same work compares required nets. The committed
	// Day 2 assessment remains intact; only the missing Day 3 fee is appended.
	repository.failAt = 0
	require.NoError(t, core.ReconcileClosedDays("ACC-001", 2, 5, "correction"))
	fees = feeRecords(repository.History())
	require.Len(t, fees, 2)
	assert.Equal(t, domain.Day(3), fees[1].Entry.ValueDay)
	assert.Equal(t, after.Records, repository.History().Records[:len(after.Records)])
	requireBalance(t, repository, "ACC-001", 3, -5000)
}
