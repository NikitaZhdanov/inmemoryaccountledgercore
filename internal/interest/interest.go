// Package interest calculates rounded daily accruals using ledger interest policy.
package interest

import "inmemoryaccountledgercore/internal/domain"

// 0.04% = 4/10000 = 1/2500 per day.
const dailyInterestDenominator int64 = 2500

// Daily returns the positive base's daily interest, rounded HALF_EVEN
// to its currency's minor unit. Nonpositive bases accrue zero. Invalid money
// returns domain.ErrInvalidCurrency and no monetary result.
func Daily(base domain.Money) (domain.Money, error) {
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
