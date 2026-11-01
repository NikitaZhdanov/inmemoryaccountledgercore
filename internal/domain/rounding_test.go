package domain_test

import (
	"fmt"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"inmemoryaccountledgercore/internal/domain"
)

// Literal ratio expectations cover signed ties, odd denominators,
// and endpoint arithmetic independently of the daily interest rate.
func TestRoundHalfEven(t *testing.T) {
	for _, tc := range []struct {
		numerator, denominator, want int64
	}{
		{0, 1, 0},
		{1, 1, 1},
		{-1, 1, -1},
		{1, 2, 0},
		{3, 2, 2},
		{5, 2, 2},
		{-1, 2, 0},
		{-3, 2, -2},
		{-5, 2, -2},
		{1, 3, 0},
		{2, 3, 1},
		{-1, 3, 0},
		{-2, 3, -1},
		{math.MaxInt64, 1, math.MaxInt64},
		{math.MinInt64, 1, math.MinInt64},
		{math.MaxInt64, 2, 4611686018427387904},
		{math.MinInt64, 3, -3074457345618258603},
		{math.MaxInt64, math.MaxInt64, 1},
		{math.MinInt64, math.MaxInt64, -1},
		{4611686018427387903, math.MaxInt64, 0},
		{4611686018427387904, math.MaxInt64, 1},
	} {
		t.Run(fmt.Sprintf("%d/%d", tc.numerator, tc.denominator), func(t *testing.T) {
			got, err := domain.RoundHalfEven(tc.numerator, tc.denominator)
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestRoundHalfEvenInvalidDenominator(t *testing.T) {
	for _, denominator := range []int64{0, -1, math.MinInt64} {
		got, err := domain.RoundHalfEven(1, denominator)
		require.ErrorIs(t, err, domain.ErrInvalidDenominator)
		assert.Zero(t, got)
	}
}
