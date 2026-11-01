// Package ledger owns ledger business policy independently of storage.
package ledger

import (
	"errors"

	"inmemoryaccountledgercore/internal/domain"
)

// 0.04% = 4/10000 = 1/2500 per day.
const dailyInterestDenominator int64 = 2500

var ErrInvalidInstalmentCount = errors.New("instalment count must be positive")

// DailyInterest returns the positive base's daily interest, rounded HALF_EVEN
// to its currency's minor unit. Nonpositive bases accrue zero. Invalid money
// returns domain.ErrInvalidCurrency and no monetary result.
func DailyInterest(base domain.Money) (domain.Money, error) {
	var rounded int64
	if minor := base.MinorUnits(); minor > 0 {
		var err error
		rounded, err = domain.RoundHalfEven(minor, dailyInterestDenominator)
		if err != nil {
			return domain.Money{}, err
		}
	}
	return domain.NewMoney(base.Currency(), rounded)
}

// AllocateInstalments splits total into count ordered amounts, using integer
// division toward zero and adding the full remainder to the last instalment.
// It preserves currency and the exact signed total, including at int64 bounds.
// A nonpositive count or invalid money returns an error and no allocation.
// This calculation does not validate whether an input is an allowed posting.
func AllocateInstalments(total domain.Money, count int) ([]domain.Money, error) {
	if count <= 0 {
		return nil, ErrInvalidInstalmentCount
	}
	part, err := domain.NewMoney(total.Currency(), total.MinorUnits()/int64(count))
	if err != nil {
		return nil, err
	}
	remainder, err := domain.NewMoney(total.Currency(), total.MinorUnits()%int64(count))
	if err != nil {
		return nil, err
	}
	last, err := part.Add(remainder)
	if err != nil {
		return nil, err
	}
	parts := make([]domain.Money, count)
	for i := range parts {
		parts[i] = part
	}
	parts[count-1] = last
	return parts, nil
}
