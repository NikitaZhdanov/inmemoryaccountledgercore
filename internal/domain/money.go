// Package domain owns shared ledger values without storage or business policy.
package domain

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
)

// Input representation errors are distinct from derived arithmetic overflow.
var (
	ErrInvalidCurrency    = errors.New("unsupported currency")
	ErrInvalidAmount      = errors.New("invalid fixed-point amount")
	ErrPrecision          = errors.New("amount exceeds currency precision")
	ErrAmountOutOfRange   = errors.New("input amount outside int64 minor-unit range")
	ErrCurrencyMismatch   = errors.New("currency mismatch")
	ErrArithmeticOverflow = errors.New("derived monetary arithmetic overflow")
)

// Currency identifies supported units; no implicit conversion is provided.
type Currency string

const (
	AED Currency = "AED"
	BHD Currency = "BHD"
)

// Scale is the number of fractional decimal places required by the currency.
func (c Currency) Scale() (int, error) {
	switch c {
	case AED:
		return 2, nil
	case BHD:
		return 3, nil
	default:
		return 0, fmt.Errorf("%w: %q", ErrInvalidCurrency, c)
	}
}

// Money is an exact signed minor-unit value. Its fields are private so only
// supported currencies can be constructed. The Go zero value is invalid.
type Money struct {
	currency Currency
	minor    int64
}

func NewMoney(currency Currency, minor int64) (Money, error) {
	if _, err := currency.Scale(); err != nil {
		return Money{}, err
	}
	return Money{currency: currency, minor: minor}, nil
}

func (m Money) Currency() Currency { return m.currency }
func (m Money) MinorUnits() int64  { return m.minor }

// ParseMoney accepts [+-]?[0-9]+(\.[0-9]+)? without rounding. Excess
// fractional zeros are allowed; nonzero excess digits reject. The caller keeps
// the original text if needed for an immutable input record.
func ParseMoney(currency Currency, raw string) (Money, error) {
	scale, err := currency.Scale()
	if err != nil {
		return Money{}, err
	}
	digits := raw
	sign := ""
	if len(digits) > 0 && (digits[0] == '+' || digits[0] == '-') {
		sign, digits = digits[:1], digits[1:]
	}
	whole, fraction, hasPoint := strings.Cut(digits, ".")
	if !decimalDigits(whole) || (hasPoint && !decimalDigits(fraction)) {
		return Money{}, ErrInvalidAmount
	}
	if len(fraction) > scale {
		for _, digit := range fraction[scale:] {
			if digit != '0' {
				return Money{}, ErrPrecision
			}
		}
		fraction = fraction[:scale]
	}
	// Parse the complete signed minor-unit text, so MinInt64 never needs a
	// positive int64 magnitude or a potentially overflowing scaling operation.
	minorText := sign + whole + fraction + strings.Repeat("0", scale-len(fraction))
	minor, err := strconv.ParseInt(minorText, 10, 64)
	if err != nil {
		return Money{}, ErrAmountOutOfRange
	}
	return Money{currency: currency, minor: minor}, nil
}

func decimalDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, digit := range s {
		if digit < '0' || digit > '9' {
			return false
		}
	}
	return true
}

// Format returns fixed-point amount text at the currency's scale, without a
// currency prefix. It formats MinInt64 without negating the numeric value.
func (m Money) Format() (string, error) {
	scale, err := m.currency.Scale()
	if err != nil {
		return "", err
	}
	digits := strconv.FormatInt(m.minor, 10)
	sign := ""
	if digits[0] == '-' {
		sign, digits = "-", digits[1:]
	}
	if len(digits) <= scale {
		digits = strings.Repeat("0", scale+1-len(digits)) + digits
	}
	point := len(digits) - scale
	return sign + digits[:point] + "." + digits[point:], nil
}

func (m Money) checkCurrency(other Money) error {
	if _, err := m.currency.Scale(); err != nil {
		return err
	}
	if _, err := other.currency.Scale(); err != nil {
		return err
	}
	if m.currency != other.currency {
		return fmt.Errorf("%w: %s and %s", ErrCurrencyMismatch, m.currency, other.currency)
	}
	return nil
}

func (m Money) Add(other Money) (Money, error) {
	if err := m.checkCurrency(other); err != nil {
		return Money{}, err
	}
	if (other.minor > 0 && m.minor > math.MaxInt64-other.minor) ||
		(other.minor < 0 && m.minor < math.MinInt64-other.minor) {
		return Money{}, fmt.Errorf("add: %w", ErrArithmeticOverflow)
	}
	return Money{currency: m.currency, minor: m.minor + other.minor}, nil
}

func (m Money) Sub(other Money) (Money, error) {
	if err := m.checkCurrency(other); err != nil {
		return Money{}, err
	}
	// Check subtraction directly: negating MinInt64 would reject valid
	// differences such as MinInt64 - MinInt64.
	if (other.minor > 0 && m.minor < math.MinInt64+other.minor) ||
		(other.minor < 0 && m.minor > math.MaxInt64+other.minor) {
		return Money{}, fmt.Errorf("subtract: %w", ErrArithmeticOverflow)
	}
	return Money{currency: m.currency, minor: m.minor - other.minor}, nil
}

func (m Money) Neg() (Money, error) {
	if _, err := m.currency.Scale(); err != nil {
		return Money{}, err
	}
	if m.minor == math.MinInt64 {
		return Money{}, fmt.Errorf("negate: %w", ErrArithmeticOverflow)
	}
	return Money{currency: m.currency, minor: -m.minor}, nil
}

// Sum checks every intermediate addition in supplied order. An empty sum is
// zero in the requested currency. Errors return no partial result.
func Sum(currency Currency, amounts ...Money) (Money, error) {
	total, err := NewMoney(currency, 0)
	if err != nil {
		return Money{}, err
	}
	for i, amount := range amounts {
		total, err = total.Add(amount)
		if err != nil {
			return Money{}, fmt.Errorf("sum term %d: %w", i, err)
		}
	}
	return total, nil
}
