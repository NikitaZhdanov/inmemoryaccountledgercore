package ledger_test

import (
	"math"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"inmemoryaccountledgercore/internal/domain"
	"inmemoryaccountledgercore/internal/ledger"
)

func money(t *testing.T, currency domain.Currency, minor int64) domain.Money {
	t.Helper()
	m, err := domain.NewMoney(currency, minor)
	require.NoError(t, err)
	return m
}

func canonicalCurrencies(t *testing.T) []domain.Currency {
	t.Helper()
	aed, err := domain.NewCurrency("AED", 2)
	require.NoError(t, err)
	bhd, err := domain.NewCurrency("BHD", 3)
	require.NoError(t, err)
	return []domain.Currency{aed, bhd}
}

// NUMBERS.md tabulates exact quotient/remainder oracles in minor units, which
// apply at either currency scale. The expectations do not call policy code.
func TestDailyInterest(t *testing.T) {
	for _, currency := range canonicalCurrencies(t) {
		for _, tc := range []struct {
			base, want int64
		}{
			{math.MinInt64, 0},
			{-1, 0},
			{0, 0},
			{1, 0},
			{1249, 0},
			{1250, 0}, // 0.5 ties to even zero.
			{1251, 1},
			{2500, 1},
			{3749, 1},
			{3750, 2}, // 1.5 ties to even two.
			{3751, 2},
			{6250, 2}, // 2.5 ties to an even nonzero quotient.
			{math.MaxInt64, 3689348814741910},
		} {
			t.Run(currency.Code()+"/"+strconv.FormatInt(tc.base, 10), func(t *testing.T) {
				base := money(t, currency, tc.base)
				got, err := ledger.DailyInterest(base)
				require.NoError(t, err)
				assert.Equal(t, money(t, currency, tc.want), got)
				assert.Equal(t, tc.base, base.MinorUnits())
			})
		}
	}
}

func TestReplayInterest(t *testing.T) {
	for _, tc := range []struct {
		code       string
		scale      int
		base, want string
	}{
		{"AED", 2, "250.00", "0.10"},
		{"AED", 2, "650.00", "0.26"},
		{"AED", 2, "465.00", "0.19"},
		{"BHD", 3, "10.000", "0.004"},
	} {
		t.Run(tc.code+"/"+tc.base, func(t *testing.T) {
			currency, err := domain.NewCurrency(tc.code, tc.scale)
			require.NoError(t, err)
			base, err := domain.ParseMoney(currency, tc.base)
			require.NoError(t, err)
			got, err := ledger.DailyInterest(base)
			require.NoError(t, err)
			formatted, err := got.Format()
			require.NoError(t, err)
			assert.Equal(t, tc.want, formatted)
			assert.Equal(t, currency, got.Currency())
		})
	}
}

func TestRoundedAccrualSums(t *testing.T) {
	for _, tc := range []struct {
		code           string
		scale          int
		bases, daily   []int64
		capitalization int64
	}{
		{"AED", 2, []int64{25000, 25000, 65000, 46500, 46500, 46500}, []int64{10, 10, 26, 19, 19, 19}, 103},
		{"BHD", 3, []int64{0, 0, 0, 0, 10000, 10000}, []int64{0, 0, 0, 0, 4, 4}, 8},
	} {
		t.Run(tc.code, func(t *testing.T) {
			currency, err := domain.NewCurrency(tc.code, tc.scale)
			require.NoError(t, err)
			var accruals []domain.Money
			for day, base := range tc.bases {
				accrual, err := ledger.DailyInterest(money(t, currency, base))
				require.NoError(t, err)
				assert.Equal(t, money(t, currency, tc.daily[day]), accrual, "day %d", day+1)
				accruals = append(accruals, accrual)
			}
			total, err := domain.Sum(currency, accruals...)
			require.NoError(t, err)
			assert.Equal(t, money(t, currency, tc.capitalization), total)
		})
	}
}

func TestAllocateInstalments(t *testing.T) {
	for _, currency := range canonicalCurrencies(t) {
		for _, tc := range []struct {
			name  string
			total int64
			count int
			want  []int64
		}{
			{"E10", 10000, 3, []int64{3333, 3333, 3334}},
			{"divisible", 9999, 3, []int64{3333, 3333, 3333}},
			{"multiple residual units", 10001, 3, []int64{3333, 3333, 3335}},
			{"smaller than count", 2, 3, []int64{0, 0, 2}},
			{"zero", 0, 3, []int64{0, 0, 0}},
			{"signed total", -10000, 3, []int64{-3333, -3333, -3334}},
			{"maximum", math.MaxInt64, 3, []int64{3074457345618258602, 3074457345618258602, 3074457345618258603}},
			{"minimum", math.MinInt64, 3, []int64{-3074457345618258602, -3074457345618258602, -3074457345618258604}},
			{"single maximum", math.MaxInt64, 1, []int64{math.MaxInt64}},
			{"single minimum", math.MinInt64, 1, []int64{math.MinInt64}},
		} {
			t.Run(currency.Code()+"/"+tc.name, func(t *testing.T) {
				total := money(t, currency, tc.total)
				got, err := ledger.AllocateInstalments(total, tc.count)
				require.NoError(t, err)
				want := make([]domain.Money, len(tc.want))
				for i, minor := range tc.want {
					want[i] = money(t, currency, minor)
				}
				assert.Equal(t, want, got)
				sum, err := domain.Sum(currency, got...)
				require.NoError(t, err)
				assert.Equal(t, total, sum, "allocation must conserve principal")
			})
		}
	}
}

func TestCalculationErrors(t *testing.T) {
	got, err := ledger.DailyInterest(domain.Money{})
	require.ErrorIs(t, err, domain.ErrInvalidCurrency)
	assert.Equal(t, domain.Money{}, got)

	parts, err := ledger.AllocateInstalments(domain.Money{}, 3)
	require.ErrorIs(t, err, domain.ErrInvalidCurrency)
	assert.Nil(t, parts)
	for _, count := range []int{0, -1} {
		parts, err := ledger.AllocateInstalments(money(t, canonicalCurrencies(t)[0], 10000), count)
		require.ErrorIs(t, err, ledger.ErrInvalidInstalmentCount)
		assert.Nil(t, parts)
	}
}
