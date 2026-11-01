package domain_test

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

func TestCurrencyAndConstruction(t *testing.T) {
	for _, tc := range []struct {
		currency domain.Currency
		scale    int
	}{
		{domain.AED, 2},
		{domain.BHD, 3},
	} {
		t.Run(string(tc.currency), func(t *testing.T) {
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
	for _, currency := range []domain.Currency{"", "USD"} {
		_, err := currency.Scale()
		require.ErrorIs(t, err, domain.ErrInvalidCurrency)
		m, err := domain.NewMoney(currency, 0)
		assertMoneyError(t, m, err, domain.ErrInvalidCurrency)
		m, err = domain.ParseMoney(currency, "0")
		assertMoneyError(t, m, err, domain.ErrInvalidCurrency)
		m, err = domain.Sum(currency)
		assertMoneyError(t, m, err, domain.ErrInvalidCurrency)
	}
}

// Literal minor-unit oracles come from NUMBERS.md, independently of Format.
func TestParseMoney(t *testing.T) {
	for _, tc := range []struct {
		currency domain.Currency
		raw      string
		minor    int64
	}{
		{domain.AED, "0", 0},
		{domain.AED, "-0.000", 0},
		{domain.BHD, "+0.0000", 0},
		{domain.AED, "1", 100},
		{domain.AED, "1.0", 100},
		{domain.AED, "1.00", 100},
		{domain.AED, "1.000", 100},
		{domain.AED, "1.2", 120},
		{domain.BHD, "1", 1000},
		{domain.BHD, "1.0", 1000},
		{domain.BHD, "1.00", 1000},
		{domain.BHD, "1.000", 1000},
		{domain.BHD, "1.0000", 1000},
		{domain.BHD, "1.2", 1200},
		{domain.BHD, "1.23", 1230},
		{domain.AED, "0.01", 1},
		{domain.AED, "-0.01", -1},
		{domain.BHD, "0.001", 1},
		{domain.BHD, "-0.001", -1},
		{domain.AED, "+0012.30", 1230},
		{domain.AED, "-12.30", -1230},
		{domain.BHD, "12.345", 12345},
		{domain.BHD, "-00012.345000", -12345},
		{domain.AED, "92233720368547758.07", math.MaxInt64},
		{domain.AED, "-92233720368547758.08", math.MinInt64},
		{domain.BHD, "9223372036854775.807", math.MaxInt64},
		{domain.BHD, "-9223372036854775.808", math.MinInt64},
		{domain.AED, "+0092233720368547758.07000", math.MaxInt64},
		{domain.AED, "-0092233720368547758.08000", math.MinInt64},
		{domain.BHD, "+009223372036854775.807000", math.MaxInt64},
		{domain.BHD, "-009223372036854775.808000", math.MinInt64},
	} {
		t.Run(string(tc.currency)+"/"+tc.raw, func(t *testing.T) {
			got, err := domain.ParseMoney(tc.currency, tc.raw)
			require.NoError(t, err)
			assert.Equal(t, tc.currency, got.Currency())
			assert.Equal(t, tc.minor, got.MinorUnits())
		})
	}
}

func TestParseMoneyRepresentabilityErrors(t *testing.T) {
	for _, tc := range []struct {
		currency domain.Currency
		raw      string
		wantErr  error
	}{
		{domain.AED, "1.005", domain.ErrPrecision},
		{domain.AED, "0.0001", domain.ErrPrecision},
		{domain.BHD, "1.0001", domain.ErrPrecision},
		{domain.BHD, "-0.0001", domain.ErrPrecision},
		{domain.AED, "92233720368547758.08", domain.ErrAmountOutOfRange},
		{domain.AED, "-92233720368547758.09", domain.ErrAmountOutOfRange},
		{domain.BHD, "9223372036854775.808", domain.ErrAmountOutOfRange},
		{domain.BHD, "-9223372036854775.809", domain.ErrAmountOutOfRange},
		{domain.AED, "92233720368547759", domain.ErrAmountOutOfRange},
		{domain.BHD, "9223372036854776", domain.ErrAmountOutOfRange},
		{domain.AED, "9223372036854775808", domain.ErrAmountOutOfRange},
	} {
		t.Run(string(tc.currency)+"/"+tc.raw, func(t *testing.T) {
			got, err := domain.ParseMoney(tc.currency, tc.raw)
			assertMoneyError(t, got, err, tc.wantErr)
			assert.NotErrorIs(t, err, domain.ErrArithmeticOverflow,
				"input representation failure classified as derived overflow")
		})
	}
}

func TestParseMoneySyntaxErrors(t *testing.T) {
	for _, currency := range []domain.Currency{domain.AED, domain.BHD} {
		for _, raw := range []string{
			"", "+", "-", ".1", "1.", "1.2.3", "1e2", "1,000.00", "1_000",
			" 1", "1 ", "1\n", "1 0", "--1", "++1", "+-1", "NaN", "Inf", "１.００", "−1",
		} {
			t.Run(string(currency)+"/"+raw, func(t *testing.T) {
				got, err := domain.ParseMoney(currency, raw)
				assertMoneyError(t, got, err, domain.ErrInvalidAmount)
			})
		}
	}
}

// Construct directly from independently tabulated minor units: a parser and
// formatter with matching mistakes must not be able to validate each other.
func TestMoneyFormat(t *testing.T) {
	for _, tc := range []struct {
		currency domain.Currency
		minor    int64
		want     string
	}{
		{domain.AED, 0, "0.00"},
		{domain.BHD, 0, "0.000"},
		{domain.AED, 1, "0.01"},
		{domain.AED, -1, "-0.01"},
		{domain.BHD, 1, "0.001"},
		{domain.BHD, -1, "-0.001"},
		{domain.AED, 100, "1.00"},
		{domain.BHD, 1000, "1.000"},
		{domain.AED, 1230, "12.30"},
		{domain.AED, -1230, "-12.30"},
		{domain.BHD, 12345, "12.345"},
		{domain.BHD, -12345, "-12.345"},
		{domain.AED, math.MaxInt64, "92233720368547758.07"},
		{domain.AED, math.MinInt64, "-92233720368547758.08"},
		{domain.BHD, math.MaxInt64, "9223372036854775.807"},
		{domain.BHD, math.MinInt64, "-9223372036854775.808"},
	} {
		t.Run(string(tc.currency)+"/"+tc.want, func(t *testing.T) {
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

func TestCheckedArithmetic(t *testing.T) {
	for _, currency := range []domain.Currency{domain.AED, domain.BHD} {
		t.Run(string(currency), func(t *testing.T) {
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
	for _, currency := range []domain.Currency{domain.AED, domain.BHD} {
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
			t.Run(string(currency)+"/"+tc.name, func(t *testing.T) {
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
	aed, bhd := money(t, domain.AED, 0), money(t, domain.BHD, 0)
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
	got, err = domain.Sum(domain.AED, aed, bhd)
	assertMoneyError(t, got, err, domain.ErrCurrencyMismatch)
	got, err = domain.Sum(domain.BHD, bhd, aed)
	assertMoneyError(t, got, err, domain.ErrCurrencyMismatch)
	got, err = domain.Sum(domain.AED, aed, invalid)
	assertMoneyError(t, got, err, domain.ErrInvalidCurrency)
}

func assertMoneyError(t *testing.T, got domain.Money, err, wantErr error) {
	t.Helper()
	require.ErrorIs(t, err, wantErr)
	assert.Equal(t, domain.Money{}, got, "error returned a partial monetary result")
}
