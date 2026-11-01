// Package posting prepares validated input batches without reading or writing storage.
package posting

import (
	"errors"
	"fmt"

	"inmemoryaccountledgercore/internal/domain"
	"inmemoryaccountledgercore/internal/projection"
)

// ErrUnsupportedInput identifies a workflow not implemented by this core yet.
// It is a technical failure, with no stored attempt or business rejection.
var ErrUnsupportedInput = errors.New("input workflow not implemented")

const (
	DiagnosticDuplicateID      = "DUPLICATE_EVENT_ID"
	DiagnosticInvalidEventID   = "INVALID_EVENT_ID"
	DiagnosticUnknownAccount   = "UNKNOWN_ACCOUNT"
	DiagnosticInvalidCurrency  = "INVALID_CURRENCY"
	DiagnosticCurrencyMismatch = "CURRENCY_MISMATCH"
	DiagnosticInvalidAmount    = "INVALID_AMOUNT"
	DiagnosticAmountPrecision  = "AMOUNT_PRECISION"
	DiagnosticAmountRange      = "AMOUNT_OUT_OF_RANGE"
	DiagnosticNegativeAmount   = "NEGATIVE_AMOUNT"
	negativeAmountReason       = "amount must be nonnegative"
)

// Prepare builds an input attempt, outcome, and its postings, transitions, or
// diagnostics against committed history and a configured account value. A zero
// account denotes an unknown input account. It changes no history or configuration
// and has no repository dependency.
// Business rejections return a batch and nil error; technical failures return
// no batch or outcome. Duplicate checks precede dispatch and amount validation.
func Prepare(history domain.History, account domain.Account, input domain.InputEvent) (domain.Batch, domain.InputOutcome, error) {
	attempt, duplicate := prepareAttempt(history, input)
	if duplicate {
		return buildRejectionBatch(attempt, DiagnosticDuplicateID, "event ID has already been submitted")
	}
	if input.ID == "" {
		return buildRejectionBatch(attempt, DiagnosticInvalidEventID, "event ID is empty")
	}
	if input.InstalmentCount < 0 || (input.InstalmentCount != 0 && input.Kind != domain.InputCredit) {
		return domain.Batch{}, domain.InputOutcome{}, fmt.Errorf("%w: %s (instalments %d)", ErrUnsupportedInput, input.Kind, input.InstalmentCount)
	}
	switch input.Kind {
	case domain.InputReversal:
		if code, err := validateAccount(account, input); err != nil {
			return buildRejectionBatch(attempt, code, err.Error())
		}
		batch, outcome, err := reversalBatch(history, attempt)
		if err != nil {
			return domain.Batch{}, domain.InputOutcome{}, err
		}
		if err := projection.ValidateBatch(history, batch, input.AccountID, input.ValueDay); err != nil {
			return domain.Batch{}, domain.InputOutcome{}, err
		}
		return batch, outcome, nil
	case domain.InputCredit, domain.InputDebit, domain.InputAuthorization, domain.InputSettlement:
	default:
		return domain.Batch{}, domain.InputOutcome{}, fmt.Errorf("%w: %s", ErrUnsupportedInput, input.Kind)
	}
	amount, code, err := parseInputAmount(account, input)
	if err != nil {
		return buildRejectionBatch(attempt, code, err.Error())
	}
	batch, outcome, err := submissionBatch(history, attempt, amount)
	if err != nil {
		return domain.Batch{}, domain.InputOutcome{}, err
	}
	if err := projection.ValidateBatch(history, batch, input.AccountID, input.ValueDay); err != nil {
		return domain.Batch{}, domain.InputOutcome{}, err
	}
	return batch, outcome, nil
}

func submissionBatch(history domain.History, attempt domain.InputAttempt, amount domain.Money) (domain.Batch, domain.InputOutcome, error) {
	switch attempt.Input.Kind {
	case domain.InputAuthorization:
		return authorizationBatch(history, attempt, amount)
	case domain.InputSettlement:
		return settlementBatch(history, attempt, amount)
	case domain.InputCredit:
		if attempt.Input.InstalmentCount > 0 {
			return instalmentBatch(attempt, amount)
		}
		fallthrough
	default:
		posting, err := ordinaryPosting(attempt, amount)
		if err != nil {
			return domain.Batch{}, domain.InputOutcome{}, err
		}
		outcome := domain.InputOutcome{AttemptID: attempt.ID, Status: domain.OutcomeAccepted}
		batch := attemptBatch(attempt, outcome)
		batch.Records = append(batch.Records, posting)
		return batch, outcome, nil
	}
}

// prepareAttempt counts committed attempts and detects reuse without changing
// history. Rejected duplicates receive their own sequential attempt identity.
func prepareAttempt(history domain.History, input domain.InputEvent) (domain.InputAttempt, bool) {
	var attempts int
	duplicate := false
	for _, record := range history.Records {
		if record.Attempt != nil {
			attempts++
			if record.Attempt.Input.ID == input.ID {
				duplicate = true
			}
		}
	}
	attempt := domain.InputAttempt{ID: domain.AttemptID(fmt.Sprintf("attempt/%d", attempts+1)), Input: input}
	return attempt, duplicate
}

func ordinaryPosting(attempt domain.InputAttempt, amount domain.Money) (domain.Record, error) {
	input := attempt.Input
	kind := domain.EntryCredit
	if input.Kind == domain.InputDebit {
		kind = domain.EntryDebit
		var err error
		amount, err = amount.Neg()
		if err != nil {
			return domain.Record{}, err
		}
	}
	return domain.Record{
		Key: domain.RecordKey(fmt.Sprintf("entry/%q", input.ID)),
		Entry: &domain.MonetaryEntry{
			AccountID: input.AccountID, Amount: amount, BookingDay: input.BookingDay,
			ValueDay: input.ValueDay, Kind: kind, SourceAttemptID: attempt.ID,
		},
	}, nil
}

func validateAccount(account domain.Account, input domain.InputEvent) (string, error) {
	if account.ID == "" || account.ID != input.AccountID {
		return DiagnosticUnknownAccount, fmt.Errorf("%w: %q", projection.ErrUnknownAccount, input.AccountID)
	}
	if _, err := input.Currency.Scale(); err != nil {
		return DiagnosticInvalidCurrency, err
	}
	if input.Currency != account.Currency {
		return DiagnosticCurrencyMismatch, domain.ErrCurrencyMismatch
	}
	return "", nil
}

func parseInputAmount(account domain.Account, input domain.InputEvent) (domain.Money, string, error) {
	if code, err := validateAccount(account, input); err != nil {
		return domain.Money{}, code, err
	}
	amount, err := domain.ParseMoney(account.Currency, input.RawAmount)
	if err != nil {
		code := DiagnosticInvalidAmount
		switch {
		case errors.Is(err, domain.ErrPrecision):
			code = DiagnosticAmountPrecision
		case errors.Is(err, domain.ErrAmountOutOfRange):
			code = DiagnosticAmountRange
		}
		return domain.Money{}, code, err
	}
	if amount.MinorUnits() < 0 {
		return domain.Money{}, DiagnosticNegativeAmount, errors.New(negativeAmountReason)
	}
	return amount, "", nil
}

func attemptBatch(attempt domain.InputAttempt, outcome domain.InputOutcome) domain.Batch {
	return domain.Batch{Records: []domain.Record{
		{Key: domain.RecordKey(attempt.ID), Attempt: &attempt},
		{Key: domain.RecordKey(string(attempt.ID) + "/outcome"), Outcome: &outcome},
	}}
}

func buildRejectionBatch(attempt domain.InputAttempt, code, reason string) (domain.Batch, domain.InputOutcome, error) {
	key := domain.RecordKey(string(attempt.ID) + "/diagnostic")
	outcome := domain.InputOutcome{AttemptID: attempt.ID, Status: domain.OutcomeRejected, DiagnosticKey: key}
	batch := attemptBatch(attempt, outcome)
	batch.Records = append(batch.Records, domain.Record{Key: key, Diagnostic: &domain.Diagnostic{
		Code: code, Reason: reason, AccountID: attempt.Input.AccountID,
		ObservationDay: attempt.Input.BookingDay, AttemptID: attempt.ID,
	}})
	return batch, outcome, nil
}
