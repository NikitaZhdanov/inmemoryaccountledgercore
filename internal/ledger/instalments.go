package ledger

import (
	"fmt"

	"inmemoryaccountledgercore/internal/domain"
)

func instalmentBatch(attempt domain.InputAttempt, amount domain.Money) (domain.Batch, domain.InputOutcome, error) {
	parts, err := AllocateInstalments(amount, attempt.Input.InstalmentCount)
	if err != nil {
		return domain.Batch{}, domain.InputOutcome{}, err
	}
	outcome := domain.InputOutcome{AttemptID: attempt.ID, Status: domain.OutcomeAccepted}
	batch := attemptBatch(attempt, outcome)
	for i, part := range parts {
		batch.Records = append(batch.Records, domain.Record{
			Key: domain.RecordKey(fmt.Sprintf("entry/%q/instalment/%d", attempt.Input.ID, i+1)),
			Entry: &domain.MonetaryEntry{
				AccountID: attempt.Input.AccountID, Amount: part,
				BookingDay: attempt.Input.BookingDay, ValueDay: attempt.Input.ValueDay,
				Kind: domain.EntryInstalment, SourceAttemptID: attempt.ID,
				CauseKey: domain.RecordKey(attempt.ID), Instalment: i + 1, InstalmentCount: len(parts),
			},
		})
	}
	return batch, outcome, nil
}
