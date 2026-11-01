package posting

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"inmemoryaccountledgercore/internal/domain"
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
				got, err := allocateInstalments(total, tc.count)
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

func TestAllocationErrors(t *testing.T) {
	parts, err := allocateInstalments(domain.Money{}, 3)
	require.ErrorIs(t, err, domain.ErrInvalidCurrency)
	assert.Nil(t, parts)
	for _, count := range []int{0, -1} {
		parts, err := allocateInstalments(money(t, canonicalCurrencies(t)[0], 10000), count)
		require.ErrorIs(t, err, errInvalidInstalmentCount)
		assert.Nil(t, parts)
	}
}
