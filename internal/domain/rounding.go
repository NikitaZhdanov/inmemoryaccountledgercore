package domain

import "errors"

var ErrInvalidDenominator = errors.New("rounding denominator must be positive")

// RoundHalfEven rounds the exact ratio numerator/denominator to the nearest
// integer, choosing the even integer at a tie. It supports every signed int64
// numerator without overflowing intermediates. A nonpositive denominator returns
// zero and ErrInvalidDenominator. The caller supplies units and decides when
// rounding applies.
func RoundHalfEven(numerator, denominator int64) (int64, error) {
	if denominator <= 0 {
		return 0, ErrInvalidDenominator
	}
	quotient, remainder := numerator/denominator, numerator%denominator
	if remainder < 0 {
		// The remainder's magnitude is less than the positive denominator,
		// so negating it is safe even when numerator is MinInt64.
		remainder = -remainder
	}
	// Compare distances to the adjacent integers without doubling the remainder.
	complement := denominator - remainder
	if remainder > complement || (remainder == complement && quotient%2 != 0) {
		// A nonzero remainder requires denominator > 1, leaving room to adjust.
		if numerator < 0 {
			quotient--
		} else {
			quotient++
		}
	}
	return quotient, nil
}
