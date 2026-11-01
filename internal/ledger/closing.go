package ledger

import (
	"errors"
	"fmt"
	"sort"

	"inmemoryaccountledgercore/internal/domain"
)

var ErrInvalidReconciliation = errors.New("invalid close or reconciliation request")

// CloseDay explicitly closes one account's supplied positive day using ordinary
// fee reconciliation. It reconciles all recorded closes, including the requested
// day, in chronological order. The caller owns the close schedule and chooses
// capitalization-aware finalization separately at its window boundary.
// Each day's fee and any new close checkpoint form one validated atomic batch;
// a later failure does not roll back earlier committed days. bookingDay dates
// new fee effects; cause identifies the close or recorded financial correction.
// Both must be supplied (positive booking day, nonempty cause).
// Repeating an unchanged close appends nothing. Submissions do not call this API.
func (c *Core) CloseDay(id domain.AccountID, day, bookingDay domain.Day, cause domain.RecordKey) error {
	if err := c.validateReconciliation(id, day, bookingDay, cause); err != nil {
		return err
	}
	return c.reconcileDays(id, 1, day, bookingDay, cause)
}

// ReconcileClosedDays corrects recorded closes on/after the positive fromDay,
// in chronological order. It never closes an open day. Callers supply the booking
// day and stable cause of the correction; unchanged required nets append nothing.
// Unpaid accruals are rebuilt from corrected history, with no accrual postings.
// Each day commits separately; technical errors propagate without diagnostics.
func (c *Core) ReconcileClosedDays(id domain.AccountID, fromDay, bookingDay domain.Day, cause domain.RecordKey) error {
	if err := c.validateReconciliation(id, fromDay, bookingDay, cause); err != nil {
		return err
	}
	return c.reconcileDays(id, fromDay, 0, bookingDay, cause)
}

func (c *Core) validateReconciliation(id domain.AccountID, day, bookingDay domain.Day, cause domain.RecordKey) error {
	if _, exists := c.accounts[id]; !exists {
		return fmt.Errorf("%w: %q", ErrUnknownAccount, id)
	}
	if day < 1 || bookingDay < 1 || cause == "" {
		return ErrInvalidReconciliation
	}
	return nil
}

func (c *Core) reconcileDays(id domain.AccountID, fromDay, closeDay, bookingDay domain.Day, cause domain.RecordKey) error {
	history := c.repository.History()
	projection, err := Rebuild(history, len(history.Records))
	if err != nil {
		return err
	}
	days, err := projection.ClosedDays(id)
	if err != nil {
		return err
	}
	newClose := closeDay != 0 && !projection.closed[accountDay{id, closeDay}]
	if newClose {
		days = append(days, closeDay)
		sort.Slice(days, func(i, j int) bool { return days[i] < days[j] })
	}
	for _, day := range days {
		if day < fromDay {
			continue
		}
		// Every day observes earlier committed fee corrections.
		history = c.repository.History()
		projection, err = Rebuild(history, len(history.Records))
		if err != nil {
			return err
		}
		batch, err := c.feeBatch(history, projection, id, day, bookingDay, cause)
		if err != nil {
			return err
		}
		if newClose && day == closeDay {
			batch.Records = append(batch.Records, domain.Record{
				Key:   domain.RecordKey(fmt.Sprintf("close/%q/%d", id, day)),
				Close: &domain.DayClose{AccountID: id, Day: day},
			})
		}
		if len(batch.Records) == 0 {
			continue
		}
		if err := validatePosting(history, batch, domain.InputEvent{AccountID: id, ValueDay: day}); err != nil {
			return err
		}
		if err := c.repository.Append(batch); err != nil {
			return err
		}
	}
	return nil
}

func (c *Core) feeBatch(history domain.History, projection *Projection, id domain.AccountID, day, bookingDay domain.Day, cause domain.RecordKey) (domain.Batch, error) {
	base, err := projection.Balance(id, day, BalanceOptions{ExcludeFeeDay: day})
	if err != nil {
		return domain.Batch{}, err
	}
	required, err := domain.NewMoney(c.accounts[id].Currency, 0)
	if err != nil {
		return domain.Batch{}, err
	}
	if base.MinorUnits() < 0 {
		required, err = c.charges[required.Currency()].Neg()
		if err != nil {
			return domain.Batch{}, err
		}
	}
	net, err := projection.FeeNet(id, day)
	if err != nil {
		return domain.Batch{}, err
	}
	difference, err := required.Sub(net)
	if err != nil {
		return domain.Batch{}, err
	}
	if difference.MinorUnits() == 0 {
		return domain.Batch{}, nil
	}

	obligation := domain.FeeObligationID(fmt.Sprintf("fee/%q/%d", id, day))
	var original domain.RecordKey
	for _, record := range history.Records {
		if entry := record.Entry; entry != nil && entry.Kind == domain.EntryFeeAssessment && entry.FeeObligationID == obligation {
			original = record.Key
			break
		}
	}
	kind := domain.EntryFeeAssessment
	key := domain.RecordKey(string(obligation) + "/assessment")
	if original != "" {
		kind = domain.EntryFeeAdjustment
		if difference.MinorUnits() > 0 {
			kind = domain.EntryFeeRefund
		}
		key = domain.RecordKey(fmt.Sprintf("%s/cause/%q", obligation, cause))
	}
	return domain.Batch{Records: []domain.Record{{Key: key, Entry: &domain.MonetaryEntry{
		AccountID: id, Amount: difference, BookingDay: bookingDay, ValueDay: day,
		Kind: kind, CauseKey: cause, OriginalEntryKey: original, FeeObligationID: obligation,
	}}}}, nil
}
