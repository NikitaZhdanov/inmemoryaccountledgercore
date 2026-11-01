package interest_test

import (
	"math"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"inmemoryaccountledgercore/internal/domain"
	"inmemoryaccountledgercore/internal/interest"
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

// Literal quotient/remainder expectations in minor units apply at either
// currency scale. The expectations do not call policy code.
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
				got, err := interest.Daily(base)
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
			got, err := interest.Daily(base)
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
				accrual, err := interest.Daily(money(t, currency, base))
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

func TestInvalidInterestBase(t *testing.T) {
	got, err := interest.Daily(domain.Money{})
	require.ErrorIs(t, err, domain.ErrInvalidCurrency)
	assert.Equal(t, domain.Money{}, got)
}
