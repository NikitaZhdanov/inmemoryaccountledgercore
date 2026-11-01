package domain

import "fmt"

// MaxCurrencyScale keeps the decimal unit factor within signed int64.
const MaxCurrencyScale = 18

const currencyCodeLength = 3

// Currency identifies a unit by code and decimal scale. It validates syntax,
// not membership in a currency registry. Its Go zero value is invalid.
type Currency struct {
	code  string
	scale int
}

func NewCurrency(code string, scale int) (Currency, error) {
	c := Currency{code: code, scale: scale}
	if _, err := c.Scale(); err != nil {
		return Currency{}, err
	}
	return c, nil
}

func (c Currency) Code() string { return c.code }

// Scale validates the descriptor and returns its fractional decimal places.
func (c Currency) Scale() (int, error) {
	if len(c.code) != currencyCodeLength {
		return 0, fmt.Errorf("%w: code must contain %d uppercase ASCII letters", ErrInvalidCurrency, currencyCodeLength)
	}
	for _, letter := range c.code {
		if letter < 'A' || letter > 'Z' {
			return 0, fmt.Errorf("%w: code must contain uppercase ASCII letters", ErrInvalidCurrency)
		}
	}
	if c.scale < 0 || c.scale > MaxCurrencyScale {
		return 0, fmt.Errorf("%w: scale must be between 0 and %d", ErrInvalidCurrency, MaxCurrencyScale)
	}
	return c.scale, nil
}
