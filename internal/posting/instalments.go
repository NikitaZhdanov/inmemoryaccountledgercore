package posting

import (
	"errors"
	"fmt"

	"inmemoryaccountledgercore/internal/domain"
)

var errInvalidInstalmentCount = errors.New("instalment count must be positive")

func instalmentBatch(attempt domain.InputAttempt, amount domain.Money) (domain.Batch, domain.InputOutcome, error) {
	parts, err := allocateInstalments(amount, attempt.Input.InstalmentCount)
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

// allocateInstalments splits total into count ordered amounts, using integer
// division toward zero and adding the full remainder to the last instalment.
// It preserves currency and the exact signed total, including at int64 bounds.
// A nonpositive count or invalid money returns an error and no allocation.
// This calculation does not validate whether an input is an allowed posting.
func allocateInstalments(total domain.Money, count int) ([]domain.Money, error) {
	if count <= 0 {
		return nil, errInvalidInstalmentCount
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
