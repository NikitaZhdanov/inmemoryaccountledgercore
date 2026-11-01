package ledger

import (
	"fmt"

	"inmemoryaccountledgercore/internal/domain"
)

const (
	DiagnosticInsufficientAvailability = "INSUFFICIENT_AVAILABILITY"
	DiagnosticUnknownAuthorization     = "UNKNOWN_AUTHORIZATION"
)

// authorizationBatch uses the current record checkpoint, including posted fees
// and active holds. Derived unpaid accruals are never spendable money.
func authorizationBatch(history domain.History, attempt domain.InputAttempt, amount domain.Money) (domain.Batch, domain.InputOutcome, error) {
	if amount.MinorUnits() < 0 {
		batch, outcome := buildRejectionBatch(attempt, DiagnosticNegativeAmount, negativeAmountReason)
		return batch, outcome, nil
	}
	input := attempt.Input
	if input.AuthorizationID == "" || input.BookingDay != input.ValueDay {
		return domain.Batch{}, domain.InputOutcome{}, fmt.Errorf("%w: authorization requires an ID and matching booking/value days", ErrUnsupportedInput)
	}
	projection, err := Rebuild(history, len(history.Records))
	if err != nil {
		return domain.Batch{}, domain.InputOutcome{}, err
	}
	if _, exists := projection.Authorization(input.AccountID, input.AuthorizationID); exists {
		return domain.Batch{}, domain.InputOutcome{}, fmt.Errorf("%w: authorization ID already recorded", ErrUnsupportedInput)
	}
	available, err := projection.AvailableBalance(input.AccountID, input.ValueDay)
	if err != nil {
		return domain.Batch{}, domain.InputOutcome{}, err
	}
	proposed, err := available.Sub(amount)
	if err != nil {
		return domain.Batch{}, domain.InputOutcome{}, err
	}
	outcome := domain.InputOutcome{AttemptID: attempt.ID, Status: domain.OutcomeAccepted}
	state := domain.AuthorizationApproved
	if proposed.MinorUnits() < 0 {
		outcome.Status = domain.OutcomeDeclined
		outcome.DiagnosticKey = domain.RecordKey(string(attempt.ID) + "/diagnostic")
		state = domain.AuthorizationDeclined
	}
	batch := attemptBatch(attempt, outcome)
	batch.Records = append(batch.Records, domain.Record{
		Key: domain.RecordKey(fmt.Sprintf("authorization/%q", input.ID)),
		Authorization: &domain.AuthorizationTransition{
			AccountID: input.AccountID, AuthorizationID: input.AuthorizationID,
			AttemptID: attempt.ID, BookingDay: input.BookingDay, ValueDay: input.ValueDay,
			State: state, Hold: amount,
		},
	})
	if outcome.Status == domain.OutcomeDeclined {
		batch.Records = append(batch.Records, domain.Record{Key: outcome.DiagnosticKey, Diagnostic: &domain.Diagnostic{
			Code: DiagnosticInsufficientAvailability, Reason: "hold would make available balance negative",
			AccountID: input.AccountID, ObservationDay: input.BookingDay, AttemptID: attempt.ID,
		}})
	}
	return batch, outcome, nil
}

func settlementBatch(history domain.History, attempt domain.InputAttempt, amount domain.Money) (domain.Batch, domain.InputOutcome, error) {
	if amount.MinorUnits() < 0 {
		batch, outcome := buildRejectionBatch(attempt, DiagnosticNegativeAmount, negativeAmountReason)
		return batch, outcome, nil
	}
	input := attempt.Input
	if input.AuthorizationID == "" {
		return domain.Batch{}, domain.InputOutcome{}, fmt.Errorf("%w: settlement requires an authorization ID", ErrUnsupportedInput)
	}
	projection, err := Rebuild(history, len(history.Records))
	if err != nil {
		return domain.Batch{}, domain.InputOutcome{}, err
	}
	original, exists := projection.Authorization(input.AccountID, input.AuthorizationID)
	if !exists {
		batch, outcome := buildRejectionBatch(attempt, DiagnosticUnknownAuthorization, "authorization ID is absent from the account history")
		return batch, outcome, nil
	}
	if original.Transition.State != domain.AuthorizationApproved || amount.MinorUnits() > original.Transition.Hold.MinorUnits() {
		return domain.Batch{}, domain.InputOutcome{}, fmt.Errorf("%w: inactive authorization or over-capture", ErrUnsupportedInput)
	}
	debit, err := amount.Neg()
	if err != nil {
		return domain.Batch{}, domain.InputOutcome{}, err
	}
	outcome := domain.InputOutcome{AttemptID: attempt.ID, Status: domain.OutcomeAccepted}
	batch := attemptBatch(attempt, outcome)
	batch.Records = append(batch.Records,
		domain.Record{
			Key: domain.RecordKey(fmt.Sprintf("entry/%q", input.ID)),
			Entry: &domain.MonetaryEntry{
				AccountID: input.AccountID, Amount: debit, BookingDay: input.BookingDay,
				ValueDay: input.ValueDay, Kind: domain.EntrySettlement, SourceAttemptID: attempt.ID,
			},
		},
		domain.Record{
			Key: domain.RecordKey(fmt.Sprintf("authorization/%q", input.ID)),
			Authorization: &domain.AuthorizationTransition{
				AccountID: input.AccountID, AuthorizationID: input.AuthorizationID,
				AttemptID: attempt.ID, BookingDay: input.BookingDay, ValueDay: input.ValueDay,
				State: domain.AuthorizationSettled, Hold: original.Transition.Hold, OriginalHoldKey: original.Key,
			},
		},
	)
	return batch, outcome, nil
}
