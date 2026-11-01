// Package ledger coordinates configured ledger workflows and repository commits.
package ledger

import (
	"fmt"

	"inmemoryaccountledgercore/internal/domain"
	"inmemoryaccountledgercore/internal/posting"
)

// Submit records an input attempt, outcome, and any posting, transition, or
// diagnostic in one atomic batch, then reconciles affected closed days before
// returning. It never closes an open day. Business rejections return a rejected
// outcome and nil error without reconciliation. Technical failures return a zero
// outcome; a failed input batch appends nothing, but a later reconciliation
// failure retains the input and any earlier committed corrections.
func (c *Core) Submit(input domain.InputEvent) (domain.InputOutcome, error) {
	batch, outcome, err := posting.Prepare(c.repository.History(), c.accounts[input.AccountID], input)
	if err != nil {
		return domain.InputOutcome{}, err
	}
	if err := c.repository.Append(batch); err != nil {
		return domain.InputOutcome{}, err
	}
	// Supported input batches post to one account on one value day. Inspect
	// actual postings so holds, declines, and rejections trigger no corrections.
	for _, record := range batch.Records {
		if entry := record.Entry; entry != nil {
			if err := c.ReconcileClosedDays(entry.AccountID, entry.ValueDay, entry.BookingDay, domain.RecordKey(outcome.AttemptID)); err != nil {
				return domain.InputOutcome{}, fmt.Errorf("reconcile committed attempt %q: %w", outcome.AttemptID, err)
			}
			break
		}
	}
	return outcome, nil
}
