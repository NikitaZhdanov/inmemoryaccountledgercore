package posting

import (
	"errors"
	"fmt"

	"inmemoryaccountledgercore/internal/domain"
)

const DiagnosticAlreadyReversed = "ALREADY_REVERSED"

// errAlreadyReversed is a business error translated to a recorded rejection by
// reversalBatch. Storage and arithmetic failures remain technical errors.
var errAlreadyReversed = errors.New("debit has already been reversed by another attempt")

// reversalBatch resolves a supplied event reference to its stored ordinary debit.
// The source supplies the amount; E9 has no separately supplied principal amount.
// Other reversal workflows have no agreed business policy and remain unsupported.
func reversalBatch(history domain.History, attempt domain.InputAttempt) (domain.Batch, domain.InputOutcome, error) {
	input := attempt.Input
	var source domain.AttemptID
	for _, record := range history.Records {
		if record.Attempt != nil && record.Attempt.Input.ID == input.ReversesEventID {
			source = record.Attempt.ID
			break // A later rejected duplicate must not replace the original source.
		}
	}
	var original domain.Record
	if source != "" {
		for _, record := range history.Records {
			if record.Entry != nil && record.Entry.SourceAttemptID == source {
				if original.Entry != nil {
					return domain.Batch{}, domain.InputOutcome{}, fmt.Errorf("%w: reversal requires a single debit entry", ErrUnsupportedInput)
				}
				original = record
			}
		}
	}
	if original.Entry == nil || original.Entry.Kind != domain.EntryDebit ||
		original.Entry.AccountID != input.AccountID || original.Entry.Amount.Currency() != input.Currency ||
		original.Entry.ValueDay != input.ValueDay {
		return domain.Batch{}, domain.InputOutcome{}, fmt.Errorf("%w: reversal requires an ordinary debit on the same account, currency, and value day", ErrUnsupportedInput)
	}
	correction, err := principalCorrection(history, attempt, original)
	if errors.Is(err, errAlreadyReversed) {
		return buildRejectionBatch(attempt, DiagnosticAlreadyReversed, err.Error())
	}
	if err != nil {
		return domain.Batch{}, domain.InputOutcome{}, err
	}
	outcome := domain.InputOutcome{AttemptID: attempt.ID, Status: domain.OutcomeAccepted}
	batch := attemptBatch(attempt, outcome)
	batch.Records = append(batch.Records, correction.Records...)
	return batch, outcome, nil
}

// principalCorrection constructs only the financial effect. Repeating the same
// recorded cause returns no records; submitting the input again is handled by
// Prepare's duplicate rejection before this helper runs. Identity derives from
// the original entry, independently of the correction's input ID. A different
// attempt for an already reversed debit returns the business errAlreadyReversed.
func principalCorrection(history domain.History, attempt domain.InputAttempt, original domain.Record) (domain.Batch, error) {
	cause := domain.RecordKey(attempt.ID)
	for _, record := range history.Records {
		if record.Entry != nil && record.Entry.Kind == domain.EntryReversal && record.Entry.OriginalEntryKey == original.Key {
			if record.Entry.CauseKey == cause {
				return domain.Batch{}, nil
			}
			return domain.Batch{}, errAlreadyReversed
		}
	}
	amount, err := original.Entry.Amount.Neg()
	if err != nil {
		return domain.Batch{}, err
	}
	return domain.Batch{Records: []domain.Record{{
		Key: domain.RecordKey(fmt.Sprintf("reversal/%q", original.Key)),
		Entry: &domain.MonetaryEntry{
			AccountID: original.Entry.AccountID, Amount: amount,
			BookingDay: attempt.Input.BookingDay, ValueDay: original.Entry.ValueDay,
			Kind: domain.EntryReversal, SourceAttemptID: attempt.ID,
			CauseKey: cause, OriginalEntryKey: original.Key,
		},
	}}}, nil
}
