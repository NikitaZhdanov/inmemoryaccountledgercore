package ledger

import (
	"fmt"
	"slices"

	"inmemoryaccountledgercore/internal/domain"
	"inmemoryaccountledgercore/internal/projection"
)

// FinalizeDay closes the supplied window boundary and capitalizes its corrected
// rounded unpaid accruals through that day. The caller must finish all inputs,
// historical reconciliation, and earlier close checkpoints first. The core has
// no fixed window length. bookingDay and cause date/link the final postings.
// Fee, capitalization (including zero), and any new close commit in one batch.
// The capitalization key identifies the account/window ending on day; repeating
// finalization appends nothing, even with a different booking day or cause.
// This is a terminal checkpoint, not a post-capitalization correction protocol.
func (c *Core) FinalizeDay(id domain.AccountID, day, bookingDay domain.Day, cause domain.RecordKey) error {
	if err := c.validateReconciliation(id, day, bookingDay, cause); err != nil {
		return err
	}
	history := c.repository.History()
	key := domain.RecordKey(fmt.Sprintf("capitalization/%q/%d", id, day))
	for _, record := range history.Records {
		if record.Key == key {
			return nil
		}
	}
	state, err := projection.Rebuild(history, len(history.Records))
	if err != nil {
		return err
	}
	base, err := state.Balance(id, day, projection.BalanceOptions{
		ExcludeFeeDay: day, ExcludeCapitalizationDay: day,
	})
	if err != nil {
		return err
	}
	zero, err := domain.NewMoney(c.accounts[id].Currency, 0)
	if err != nil {
		return err
	}
	prior := zero
	unpaid, err := state.UnpaidAccruals(id)
	if err != nil {
		return err
	}
	for _, accrual := range unpaid {
		if accrual.Day < day {
			prior, err = prior.Add(accrual.Amount)
			if err != nil {
				return err
			}
		}
	}

	// For B < 0, current-day interest is zero regardless of the fee; B+S
	// determines eligibility. For B >= 0, the fee is zero and B supplies the
	// interest base. No provisional fee or iterative calculation is needed.
	required := zero
	if base.MinorUnits() < 0 {
		eligible, err := base.Add(prior)
		if err != nil {
			return err
		}
		if eligible.MinorUnits() < 0 {
			required, err = c.charges[base.Currency()].Neg()
			if err != nil {
				return err
			}
		}
	}
	batch, err := requiredFeeBatch(history, state, id, day, bookingDay, cause, required)
	if err != nil {
		return err
	}
	days, err := state.ClosedDays(id)
	if err != nil {
		return err
	}
	if !slices.Contains(days, day) {
		batch.Records = append(batch.Records, domain.Record{
			Key:   domain.RecordKey(fmt.Sprintf("close/%q/%d", id, day)),
			Close: &domain.DayClose{AccountID: id, Day: day},
		})
	}
	// Derive the rounded current-day accrual through the existing projection,
	// with the final fee in place and no capitalization in its interest base.
	records := make([]domain.Record, len(history.Records), len(history.Records)+len(batch.Records))
	copy(records, history.Records)
	proposed := domain.History{Records: append(records, batch.Records...)}
	closing, err := projection.Rebuild(proposed, len(proposed.Records))
	if err != nil {
		return err
	}
	current, _ := closing.Accrual(id, day) // Existing or proposed close is present.
	capitalization, err := prior.Add(current)
	if err != nil {
		return err
	}
	batch.Records = append(batch.Records, domain.Record{Key: key, Entry: &domain.MonetaryEntry{
		AccountID: id, Amount: capitalization, BookingDay: bookingDay, ValueDay: day,
		Kind: domain.EntryCapitalization, CauseKey: cause,
	}})
	if err := projection.ValidateBatch(history, batch, id, day); err != nil {
		return err
	}
	return c.repository.Append(batch)
}
