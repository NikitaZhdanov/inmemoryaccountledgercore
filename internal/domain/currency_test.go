package domain_test

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"inmemoryaccountledgercore/internal/domain"
)

func TestCurrencyValidation(t *testing.T) {
	for _, tc := range []struct {
		code  string
		scale int
	}{
		{"USD", 2}, {"PTS", 0}, {"TOK", 4}, {"MAX", 18},
	} {
		t.Run(tc.code, func(t *testing.T) {
			c, err := domain.NewCurrency(tc.code, tc.scale)
			require.NoError(t, err)
			assert.Equal(t, tc.code, c.Code())
			scale, err := c.Scale()
			require.NoError(t, err)
			assert.Equal(t, tc.scale, scale)
		})
	}
	for _, tc := range []struct {
		name, code string
		scale      int
	}{
		{"missing code", "", 2},
		{"short code", "US", 2},
		{"long code", "USDD", 2},
		{"lowercase code", "usd", 2},
		{"digit in code", "US1", 2},
		{"leading space", " US", 2},
		{"trailing space", "US ", 2},
		{"unicode code", "ＡＥＤ", 2},
		{"negative scale", "TOK", -1},
		{"scale beyond limit", "TOK", 19},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, err := domain.NewCurrency(tc.code, tc.scale)
			require.ErrorIs(t, err, domain.ErrInvalidCurrency)
			assert.Equal(t, domain.Currency{}, c)
		})
	}
}

// Literal decimal/minor-unit pairs exercise scales outside the supplied replay.
func TestMoneyConfigurablePrecision(t *testing.T) {
	for _, tc := range []struct {
		name, code string
		scale      int
		raw, want  string
		minor      int64
	}{
		{"integer zero", "PTS", 0, "0", "0", 0},
		{"integer trailing zeros", "PTS", 0, "+0012.000", "12", 12},
		{"integer negative", "PTS", 0, "-12", "-12", -12},
		{"integer maximum", "PTS", 0, "9223372036854775807", "9223372036854775807", math.MaxInt64},
		{"integer minimum", "PTS", 0, "-9223372036854775808", "-9223372036854775808", math.MinInt64},
		{"four-place padding", "TOK", 4, "1.2", "1.2000", 12000},
		{"four-place trailing zeros", "TOK", 4, "-12.345600", "-12.3456", -123456},
		{"four-place unit", "TOK", 4, "0.0001", "0.0001", 1},
		{"maximum scale zero", "MAX", 18, "0", "0.000000000000000000", 0},
		{"maximum scale unit", "MAX", 18, "0.000000000000000001", "0.000000000000000001", 1},
		{"maximum scale positive bound", "MAX", 18, "9.2233720368547758070", "9.223372036854775807", math.MaxInt64},
		{"maximum scale negative bound", "MAX", 18, "-9.2233720368547758080", "-9.223372036854775808", math.MinInt64},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := currency(t, tc.code, tc.scale)
			got, err := domain.ParseMoney(c, tc.raw)
			require.NoError(t, err)
			assert.Equal(t, c, got.Currency())
			assert.Equal(t, tc.minor, got.MinorUnits())
			// Format an independent minor-unit fixture, not the parser result.
			formatted, err := money(t, c, tc.minor).Format()
			require.NoError(t, err)
			assert.Equal(t, tc.want, formatted)
		})
	}
	for _, tc := range []struct {
		code  string
		scale int
		raw   string
		err   error
	}{
		{"PTS", 0, "1.1", domain.ErrPrecision},
		{"TOK", 4, "1.00001", domain.ErrPrecision},
		{"MAX", 18, "0.0000000000000000001", domain.ErrPrecision},
		{"PTS", 0, "9223372036854775808", domain.ErrAmountOutOfRange},
		{"PTS", 0, "-9223372036854775809", domain.ErrAmountOutOfRange},
		{"MAX", 18, "9.223372036854775808", domain.ErrAmountOutOfRange},
		{"MAX", 18, "-9.223372036854775809", domain.ErrAmountOutOfRange},
	} {
		t.Run(tc.code+"/"+tc.raw, func(t *testing.T) {
			got, err := domain.ParseMoney(currency(t, tc.code, tc.scale), tc.raw)
			assertMoneyError(t, got, err, tc.err)
		})
	}
}

func TestCurrencyCompatibility(t *testing.T) {
	base := currency(t, "TOK", 4)
	equivalent := currency(t, "TOK", 4)
	for _, tc := range []struct {
		name  string
		other domain.Currency
	}{
		{"same code different scale", currency(t, "TOK", 3)},
		{"different code same scale", currency(t, "PTS", 4)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			left, right := money(t, base, 1), money(t, tc.other, 1)
			got, err := left.Add(right)
			assertMoneyError(t, got, err, domain.ErrCurrencyMismatch)
			got, err = left.Sub(right)
			assertMoneyError(t, got, err, domain.ErrCurrencyMismatch)
			got, err = domain.Sum(base, left, right)
			assertMoneyError(t, got, err, domain.ErrCurrencyMismatch)
		})
	}
	left, right := money(t, base, 1), money(t, equivalent, 1)
	got, err := left.Add(right)
	require.NoError(t, err)
	assert.Equal(t, money(t, base, 2), got)
	got, err = left.Sub(right)
	require.NoError(t, err)
	assert.Equal(t, money(t, base, 0), got)
	got, err = domain.Sum(equivalent, left, right)
	require.NoError(t, err)
	assert.Equal(t, money(t, base, 2), got)
}
