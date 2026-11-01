package ledger

import (
	"errors"
	"fmt"
	"sort"

	"inmemoryaccountledgercore/internal/domain"
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
)

// Submit records a credit/debit attempt, its outcome, and any posting or
// diagnostic in one atomic batch. Covered business rejections return a rejected
// outcome and nil error; diagnostics are linked by Outcome.DiagnosticKey.
// Technical failures return a zero outcome and append nothing from this batch.
// Supplied days/order and raw text are retained; explicit closing and historical
// fee reconciliation are separate workflows. Authorization, settlement, reversal,
// and instalment processing remain unsupported until their planned tasks.
func (c *Core) Submit(input domain.InputEvent) (domain.InputOutcome, error) {
	history := c.repository.History()
	attempt, duplicate := prepareAttempt(history, input)
	if duplicate {
		return c.reject(attempt, DiagnosticDuplicateID, "event ID has already been submitted")
	}
	if input.ID == "" {
		return c.reject(attempt, DiagnosticInvalidEventID, "event ID is empty")
	}
	if input.Kind != domain.InputCredit && input.Kind != domain.InputDebit || input.InstalmentCount != 0 {
		return domain.InputOutcome{}, fmt.Errorf("%w: %s (instalments %d)", ErrUnsupportedInput, input.Kind, input.InstalmentCount)
	}
	amount, code, err := c.parseOrdinaryAmount(input)
	if err != nil {
		return c.reject(attempt, code, err.Error())
	}
	posting, err := ordinaryPosting(attempt, amount)
	if err != nil {
		return domain.InputOutcome{}, err
	}
	outcome := domain.InputOutcome{AttemptID: attempt.ID, Status: domain.OutcomeAccepted}
	batch := attemptBatch(attempt, outcome)
	batch.Records = append(batch.Records, posting)
	if err := validatePosting(history, batch, input); err != nil {
		return domain.InputOutcome{}, err
	}
	if err := c.repository.Append(batch); err != nil {
		return domain.InputOutcome{}, err
	}
	return outcome, nil
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

func (c *Core) parseOrdinaryAmount(input domain.InputEvent) (domain.Money, string, error) {
	account, exists := c.accounts[input.AccountID]
	if !exists {
		return domain.Money{}, DiagnosticUnknownAccount, fmt.Errorf("%w: %q", ErrUnknownAccount, input.AccountID)
	}
	if _, err := input.Currency.Scale(); err != nil {
		return domain.Money{}, DiagnosticInvalidCurrency, err
	}
	if input.Currency != account.Currency {
		return domain.Money{}, DiagnosticCurrencyMismatch, domain.ErrCurrencyMismatch
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
		return domain.Money{}, DiagnosticNegativeAmount, errors.New("credit/debit amount must be nonnegative")
	}
	return amount, "", nil
}

func attemptBatch(attempt domain.InputAttempt, outcome domain.InputOutcome) domain.Batch {
	return domain.Batch{Records: []domain.Record{
		{Key: domain.RecordKey(attempt.ID), Attempt: &attempt},
		{Key: domain.RecordKey(string(attempt.ID) + "/outcome"), Outcome: &outcome},
	}}
}

func (c *Core) reject(attempt domain.InputAttempt, code, reason string) (domain.InputOutcome, error) {
	key := domain.RecordKey(string(attempt.ID) + "/diagnostic")
	outcome := domain.InputOutcome{AttemptID: attempt.ID, Status: domain.OutcomeRejected, DiagnosticKey: key}
	batch := attemptBatch(attempt, outcome)
	batch.Records = append(batch.Records, domain.Record{Key: key, Diagnostic: &domain.Diagnostic{
		Code: code, Reason: reason, AccountID: attempt.Input.AccountID,
		ObservationDay: attempt.Input.BookingDay, AttemptID: attempt.ID,
	}})
	if err := c.repository.Append(batch); err != nil {
		return domain.InputOutcome{}, err
	}
	return outcome, nil
}

// Validate the proposed history before append, including every affected value
// checkpoint. A backdated posting may fit at its own day but overflow at a later
// posting. Rebuild also checks derived closed-day accruals and recorded holds.
func validatePosting(history domain.History, batch domain.Batch, input domain.InputEvent) error {
	history.Records = append(history.Records, batch.Records...)
	projection, err := Rebuild(history, len(history.Records))
	if err != nil {
		return err
	}
	days := []domain.Day{input.ValueDay}
	seen := map[domain.Day]bool{input.ValueDay: true}
	for _, record := range history.Records {
		entry := record.Entry
		if entry != nil && entry.AccountID == input.AccountID && entry.ValueDay >= input.ValueDay && !seen[entry.ValueDay] {
			days = append(days, entry.ValueDay)
			seen[entry.ValueDay] = true
		}
	}
	sort.Slice(days, func(i, j int) bool { return days[i] < days[j] })
	for _, day := range days {
		if _, err := projection.AvailableBalance(input.AccountID, day); err != nil {
			return err
		}
	}
	return nil
}
