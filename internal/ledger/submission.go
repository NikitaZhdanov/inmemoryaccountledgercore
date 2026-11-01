// Package ledger coordinates configured ledger workflows and repository commits.
package ledger

import (
	"inmemoryaccountledgercore/internal/domain"
	"inmemoryaccountledgercore/internal/posting"
)

// Submit records an input attempt, outcome, and any posting, transition, or
// diagnostic in one atomic batch. Business rejections return a rejected outcome
// and nil error. Technical failures return a zero outcome and append nothing.
// Historical fee reconciliation remains a separate workflow.
func (c *Core) Submit(input domain.InputEvent) (domain.InputOutcome, error) {
	batch, outcome, err := posting.Prepare(c.repository.History(), c.accounts[input.AccountID], input)
	if err != nil {
		return domain.InputOutcome{}, err
	}
	if err := c.repository.Append(batch); err != nil {
		return domain.InputOutcome{}, err
	}
	return outcome, nil
}
