package domain_test

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"inmemoryaccountledgercore/internal/domain"
)

func currency(t *testing.T, code string, scale int) domain.Currency {
	t.Helper()
	c, err := domain.NewCurrency(code, scale)
	require.NoError(t, err)
	return c
}

func canonicalCurrencies(t *testing.T) (domain.Currency, domain.Currency) {
	t.Helper()
	return currency(t, "AED", 2), currency(t, "BHD", 3)
}

func money(t *testing.T, currency domain.Currency, minor int64) domain.Money {
	t.Helper()
	m, err := domain.NewMoney(currency, minor)
	require.NoError(t, err)
	return m
}

func TestCurrencyAndConstruction(t *testing.T) {
	aed, bhd := canonicalCurrencies(t)
	for _, tc := range []struct {
		currency domain.Currency
		scale    int
	}{
		{aed, 2},
		{bhd, 3},
	} {
		t.Run(tc.currency.Code(), func(t *testing.T) {
			scale, err := tc.currency.Scale()
			require.NoError(t, err)
			assert.Equal(t, tc.scale, scale)
			for _, minor := range []int64{math.MinInt64, 0, math.MaxInt64} {
				m := money(t, tc.currency, minor)
				assert.Equal(t, tc.currency, m.Currency())
				assert.Equal(t, minor, m.MinorUnits())
			}
		})
	}
	var invalid domain.Currency
	_, err := invalid.Scale()
	require.ErrorIs(t, err, domain.ErrInvalidCurrency)
	m, err := domain.NewMoney(invalid, 0)
	assertMoneyError(t, m, err, domain.ErrInvalidCurrency)
	m, err = domain.ParseMoney(invalid, "0")
	assertMoneyError(t, m, err, domain.ErrInvalidCurrency)
	m, err = domain.Sum(invalid)
	assertMoneyError(t, m, err, domain.ErrInvalidCurrency)
}

// Literal minor-unit oracles come from NUMBERS.md, independently of Format.
func TestParseMoney(t *testing.T) {
	aed, bhd := canonicalCurrencies(t)
	for _, tc := range []struct {
		currency domain.Currency
		raw      string
		minor    int64
	}{
		{aed, "0", 0},
		{aed, "-0.000", 0},
		{bhd, "+0.0000", 0},
		{aed, "1", 100},
		{aed, "1.0", 100},
		{aed, "1.00", 100},
		{aed, "1.000", 100},
		{aed, "1.2", 120},
		{bhd, "1", 1000},
		{bhd, "1.0", 1000},
		{bhd, "1.00", 1000},
		{bhd, "1.000", 1000},
		{bhd, "1.0000", 1000},
		{bhd, "1.2", 1200},
		{bhd, "1.23", 1230},
		{aed, "0.01", 1},
		{aed, "-0.01", -1},
		{bhd, "0.001", 1},
		{bhd, "-0.001", -1},
		{aed, "+0012.30", 1230},
		{aed, "-12.30", -1230},
		{bhd, "12.345", 12345},
		{bhd, "-00012.345000", -12345},
		{aed, "92233720368547758.07", math.MaxInt64},
		{aed, "-92233720368547758.08", math.MinInt64},
		{bhd, "9223372036854775.807", math.MaxInt64},
		{bhd, "-9223372036854775.808", math.MinInt64},
		{aed, "+0092233720368547758.07000", math.MaxInt64},
		{aed, "-0092233720368547758.08000", math.MinInt64},
		{bhd, "+009223372036854775.807000", math.MaxInt64},
		{bhd, "-009223372036854775.808000", math.MinInt64},
	} {
		t.Run(tc.currency.Code()+"/"+tc.raw, func(t *testing.T) {
			got, err := domain.ParseMoney(tc.currency, tc.raw)
			require.NoError(t, err)
			assert.Equal(t, tc.currency, got.Currency())
			assert.Equal(t, tc.minor, got.MinorUnits())
		})
	}
}

func TestParseMoneyRepresentabilityErrors(t *testing.T) {
	aed, bhd := canonicalCurrencies(t)
	for _, tc := range []struct {
		currency domain.Currency
		raw      string
		wantErr  error
	}{
		{aed, "1.005", domain.ErrPrecision},
		{aed, "0.0001", domain.ErrPrecision},
		{bhd, "1.0001", domain.ErrPrecision},
		{bhd, "-0.0001", domain.ErrPrecision},
		{aed, "92233720368547758.08", domain.ErrAmountOutOfRange},
		{aed, "-92233720368547758.09", domain.ErrAmountOutOfRange},
		{bhd, "9223372036854775.808", domain.ErrAmountOutOfRange},
		{bhd, "-9223372036854775.809", domain.ErrAmountOutOfRange},
		{aed, "92233720368547759", domain.ErrAmountOutOfRange},
		{bhd, "9223372036854776", domain.ErrAmountOutOfRange},
		{aed, "9223372036854775808", domain.ErrAmountOutOfRange},
	} {
		t.Run(tc.currency.Code()+"/"+tc.raw, func(t *testing.T) {
			got, err := domain.ParseMoney(tc.currency, tc.raw)
			assertMoneyError(t, got, err, tc.wantErr)
			assert.NotErrorIs(t, err, domain.ErrArithmeticOverflow,
				"input representation failure classified as derived overflow")
		})
	}
}

func TestParseMoneySyntaxErrors(t *testing.T) {
	aed, bhd := canonicalCurrencies(t)
	for _, currency := range []domain.Currency{aed, bhd} {
		for _, raw := range []string{
			"", "+", "-", ".1", "1.", "1.2.3", "1e2", "1,000.00", "1_000",
			" 1", "1 ", "1\n", "1 0", "--1", "++1", "+-1", "NaN", "Inf", "１.００", "−1",
		} {
			t.Run(currency.Code()+"/"+raw, func(t *testing.T) {
				got, err := domain.ParseMoney(currency, raw)
				assertMoneyError(t, got, err, domain.ErrInvalidAmount)
			})
		}
	}
}

// Construct money from tabulated minor units to verify formatting independently
// of parsing.
func TestMoneyFormat(t *testing.T) {
	aed, bhd := canonicalCurrencies(t)
	for _, tc := range []struct {
		currency domain.Currency
		minor    int64
		want     string
	}{
		{aed, 0, "0.00"},
		{bhd, 0, "0.000"},
		{aed, 1, "0.01"},
		{aed, -1, "-0.01"},
		{bhd, 1, "0.001"},
		{bhd, -1, "-0.001"},
		{aed, 100, "1.00"},
		{bhd, 1000, "1.000"},
		{aed, 1230, "12.30"},
		{aed, -1230, "-12.30"},
		{bhd, 12345, "12.345"},
		{bhd, -12345, "-12.345"},
		{aed, math.MaxInt64, "92233720368547758.07"},
		{aed, math.MinInt64, "-92233720368547758.08"},
		{bhd, math.MaxInt64, "9223372036854775.807"},
		{bhd, math.MinInt64, "-9223372036854775.808"},
	} {
		t.Run(tc.currency.Code()+"/"+tc.want, func(t *testing.T) {
			got, err := money(t, tc.currency, tc.minor).Format()
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
	var invalid domain.Money
	got, err := invalid.Format()
	require.ErrorIs(t, err, domain.ErrInvalidCurrency)
	assert.Empty(t, got)
}

func TestMoneyValidateCurrency(t *testing.T) {
	aed, bhd := canonicalCurrencies(t)
	for _, tc := range []struct {
		name     string
		amount   domain.Money
		expected domain.Currency
		wantErr  error
	}{
		{"matching", money(t, aed, 0), aed, nil},
		{"equal descriptor", money(t, aed, 1), currency(t, "AED", 2), nil},
		{"minimum amount", money(t, aed, math.MinInt64), aed, nil},
		{"maximum amount", money(t, aed, math.MaxInt64), aed, nil},
		{"different code", money(t, bhd, 0), aed, domain.ErrCurrencyMismatch},
		{"different scale", money(t, aed, 0), currency(t, "AED", 3), domain.ErrCurrencyMismatch},
		{"invalid money", domain.Money{}, aed, domain.ErrInvalidCurrency},
		{"invalid expected descriptor", money(t, aed, 0), domain.Currency{}, domain.ErrInvalidCurrency},
		{"both invalid", domain.Money{}, domain.Currency{}, domain.ErrInvalidCurrency},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.amount.ValidateCurrency(tc.expected)
			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestCheckedArithmetic(t *testing.T) {
	aed, bhd := canonicalCurrencies(t)
	for _, currency := range []domain.Currency{aed, bhd} {
		t.Run(currency.Code(), func(t *testing.T) {
			for _, tc := range []struct {
				name     string
				op       func(domain.Money, domain.Money) (domain.Money, error)
				a, b     int64
				want     int64
				overflow bool
			}{
				{"add upper overflow", domain.Money.Add, math.MaxInt64, 1, 0, true},
				{"add lower overflow", domain.Money.Add, math.MinInt64, -1, 0, true},
				{"add to maximum", domain.Money.Add, math.MaxInt64 - 1, 1, math.MaxInt64, false},
				{"add to minimum", domain.Money.Add, math.MinInt64 + 1, -1, math.MinInt64, false},
				{"add mixed endpoints", domain.Money.Add, math.MaxInt64, math.MinInt64, -1, false},
				{"add reversed endpoints", domain.Money.Add, math.MinInt64, math.MaxInt64, -1, false},
				{"add zero to maximum", domain.Money.Add, math.MaxInt64, 0, math.MaxInt64, false},
				{"add zero to minimum", domain.Money.Add, math.MinInt64, 0, math.MinInt64, false},
				{"subtract upper overflow", domain.Money.Sub, math.MaxInt64, -1, 0, true},
				{"subtract lower overflow", domain.Money.Sub, math.MinInt64, 1, 0, true},
				{"subtract minimum from zero", domain.Money.Sub, 0, math.MinInt64, 0, true},
				{"subtract maximum from minimum", domain.Money.Sub, math.MinInt64, math.MaxInt64, 0, true},
				{"subtract minimum from maximum", domain.Money.Sub, math.MaxInt64, math.MinInt64, 0, true},
				{"subtract to minimum", domain.Money.Sub, math.MinInt64 + 1, 1, math.MinInt64, false},
				{"subtract to maximum", domain.Money.Sub, math.MaxInt64 - 1, -1, math.MaxInt64, false},
				{"subtract equal minima", domain.Money.Sub, math.MinInt64, math.MinInt64, 0, false},
				{"subtract equal maxima", domain.Money.Sub, math.MaxInt64, math.MaxInt64, 0, false},
				{"subtract minimum from negative unit", domain.Money.Sub, -1, math.MinInt64, math.MaxInt64, false},
				{"subtract maximum from negative unit", domain.Money.Sub, -1, math.MaxInt64, math.MinInt64, false},
				{"subtract zero from minimum", domain.Money.Sub, math.MinInt64, 0, math.MinInt64, false},
				{"subtract zero from maximum", domain.Money.Sub, math.MaxInt64, 0, math.MaxInt64, false},
			} {
				t.Run(tc.name, func(t *testing.T) {
					got, err := tc.op(money(t, currency, tc.a), money(t, currency, tc.b))
					if tc.overflow {
						assertMoneyError(t, got, err, domain.ErrArithmeticOverflow)
						assert.NotErrorIs(t, err, domain.ErrAmountOutOfRange,
							"derived overflow classified as invalid input")
						return
					}
					require.NoError(t, err)
					assert.Equal(t, money(t, currency, tc.want), got)
				})
			}
			for _, tc := range []struct{ input, want int64 }{
				{0, 0}, {1, -1}, {-1, 1},
				{math.MaxInt64, math.MinInt64 + 1}, {math.MinInt64 + 1, math.MaxInt64},
			} {
				got, err := money(t, currency, tc.input).Neg()
				require.NoError(t, err, "Neg(%d)", tc.input)
				assert.Equal(t, money(t, currency, tc.want), got, "Neg(%d)", tc.input)
			}
			got, err := money(t, currency, math.MinInt64).Neg()
			assertMoneyError(t, got, err, domain.ErrArithmeticOverflow)
		})
	}
}

func TestSum(t *testing.T) {
	aed, bhd := canonicalCurrencies(t)
	for _, currency := range []domain.Currency{aed, bhd} {
		for _, tc := range []struct {
			name     string
			terms    []int64
			want     int64
			overflow bool
		}{
			{"empty", nil, 0, false},
			{"positive intermediate overflow", []int64{math.MaxInt64, 1, -1}, 0, true},
			{"negative intermediate overflow", []int64{math.MinInt64, -1, 1}, 0, true},
			{"safe upper order", []int64{math.MaxInt64, -1, 1}, math.MaxInt64, false},
			{"safe lower order", []int64{math.MinInt64, 1, -1}, math.MinInt64, false},
			{"endpoint cancellation", []int64{math.MaxInt64, math.MinInt64}, -1, false},
		} {
			t.Run(currency.Code()+"/"+tc.name, func(t *testing.T) {
				var terms []domain.Money
				for _, minor := range tc.terms {
					terms = append(terms, money(t, currency, minor))
				}
				got, err := domain.Sum(currency, terms...)
				if tc.overflow {
					assertMoneyError(t, got, err, domain.ErrArithmeticOverflow)
					return
				}
				require.NoError(t, err)
				assert.Equal(t, money(t, currency, tc.want), got)
			})
		}
	}
}

func TestCurrencyChecksOnOperations(t *testing.T) {
	aedCurrency, bhdCurrency := canonicalCurrencies(t)
	aed, bhd := money(t, aedCurrency, 0), money(t, bhdCurrency, 0)
	var invalid domain.Money
	for _, tc := range []struct {
		name    string
		a, b    domain.Money
		wantErr error
	}{
		{"AED and BHD", aed, bhd, domain.ErrCurrencyMismatch},
		{"BHD and AED", bhd, aed, domain.ErrCurrencyMismatch},
		{"invalid left", invalid, aed, domain.ErrInvalidCurrency},
		{"invalid right", aed, invalid, domain.ErrInvalidCurrency},
		{"both invalid", invalid, invalid, domain.ErrInvalidCurrency},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tc.a.Add(tc.b)
			assertMoneyError(t, got, err, tc.wantErr)
			got, err = tc.a.Sub(tc.b)
			assertMoneyError(t, got, err, tc.wantErr)
		})
	}
	got, err := invalid.Neg()
	assertMoneyError(t, got, err, domain.ErrInvalidCurrency)
	got, err = domain.Sum(aedCurrency, aed, bhd)
	assertMoneyError(t, got, err, domain.ErrCurrencyMismatch)
	got, err = domain.Sum(bhdCurrency, bhd, aed)
	assertMoneyError(t, got, err, domain.ErrCurrencyMismatch)
	got, err = domain.Sum(aedCurrency, aed, invalid)
	assertMoneyError(t, got, err, domain.ErrInvalidCurrency)
}

func assertMoneyError(t *testing.T, got domain.Money, err, wantErr error) {
	t.Helper()
	require.ErrorIs(t, err, wantErr)
	assert.Equal(t, domain.Money{}, got, "error returned a partial monetary result")
}
