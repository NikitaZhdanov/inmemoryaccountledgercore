package memory_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"inmemoryaccountledgercore/internal/domain"
	"inmemoryaccountledgercore/internal/ledger"
	"inmemoryaccountledgercore/internal/repository/memory"
)

// Composition checks conformance without making the adapter import ledger.
var _ ledger.Repository = (*memory.Repository)(nil)

func currency(t *testing.T, code string, scale int) domain.Currency {
	t.Helper()
	c, err := domain.NewCurrency(code, scale)
	require.NoError(t, err)
	return c
}

func money(t *testing.T, c domain.Currency, minor int64) domain.Money {
	t.Helper()
	m, err := domain.NewMoney(c, minor)
	require.NoError(t, err)
	return m
}

// Literal TASK fields exercise storage only; these fixtures do not execute
// decisions, validate references, or calculate account balances.
func records(t *testing.T) []domain.Record {
	t.Helper()
	aed := currency(t, "AED", 2)
	return []domain.Record{
		{Key: "account/ACC-001", Account: &domain.Account{
			ID: "ACC-001", Currency: aed, OpeningBalance: money(t, aed, 0),
		}},
		{Key: "attempt/1", Attempt: &domain.InputAttempt{ID: "attempt-1", Input: domain.InputEvent{
			ID: "E1", Kind: domain.InputCredit, AccountID: "ACC-001", Currency: aed,
			RawAmount: "1200.00", BookingDay: 1, ValueDay: 1, Order: 1,
		}}},
		{Key: "outcome/1", Outcome: &domain.InputOutcome{
			AttemptID: "attempt-1", Status: domain.OutcomeAccepted,
		}},
		{Key: "entry/E1", Entry: &domain.MonetaryEntry{
			AccountID: "ACC-001", Amount: money(t, aed, 120000), BookingDay: 1, ValueDay: 1,
			Kind: domain.EntryCredit, SourceAttemptID: "attempt-1", CauseKey: "attempt/1",
		}},
		{Key: "attempt/2", Attempt: &domain.InputAttempt{ID: "attempt-2", Input: domain.InputEvent{
			ID: "E3", Kind: domain.InputAuthorization, AccountID: "ACC-001", Currency: aed,
			RawAmount: "200.00", BookingDay: 2, ValueDay: 2, AuthorizationID: "Auth-A", Order: 3,
		}}},
		{Key: "outcome/2", Outcome: &domain.InputOutcome{
			AttemptID: "attempt-2", Status: domain.OutcomeAccepted,
		}},
		{Key: "hold/Auth-A", Authorization: &domain.AuthorizationTransition{
			AccountID: "ACC-001", AuthorizationID: "Auth-A", AttemptID: "attempt-2",
			BookingDay: 2, ValueDay: 2, State: domain.AuthorizationApproved, Hold: money(t, aed, 20000),
		}},
		{Key: "attempt/3", Attempt: &domain.InputAttempt{ID: "attempt-3", Input: domain.InputEvent{
			ID: "E6", Kind: domain.InputSettlement, AccountID: "ACC-001", Currency: aed,
			RawAmount: "180.00", BookingDay: 4, ValueDay: 4, AuthorizationID: "Auth-Z", Order: 6,
		}}},
		{Key: "outcome/3", Outcome: &domain.InputOutcome{
			AttemptID: "attempt-3", Status: domain.OutcomeRejected, DiagnosticKey: "diagnostic/3",
		}},
		{Key: "diagnostic/3", Diagnostic: &domain.Diagnostic{
			Code: "unknown_authorization", Reason: "authorization ID is absent",
			AccountID: "ACC-001", ObservationDay: 4, AttemptID: "attempt-3", CauseKey: "attempt/3",
		}},
		{Key: "close/ACC-001/4", Close: &domain.DayClose{AccountID: "ACC-001", Day: 4}},
	}
}

func TestAppendOrderAndSnapshots(t *testing.T) {
	var repository memory.Repository // No constructor or defaults are required.
	assert.Empty(t, repository.History().Records)
	require.NoError(t, repository.Append(domain.Batch{}))
	assert.Empty(t, repository.History().Records)

	want := records(t)
	for _, record := range records(t) {
		require.NoError(t, repository.Append(domain.Batch{Records: []domain.Record{record}}))
	}
	snapshot := repository.History()
	assert.Equal(t, want, snapshot.Records)

	// Booking dates cannot reorder the supplied E9/E10 stream positions.
	bhd := currency(t, "BHD", 3)
	later := []domain.Record{
		{Key: "attempt/4", Attempt: &domain.InputAttempt{ID: "attempt-4", Input: domain.InputEvent{
			ID: "E9", Kind: domain.InputReversal, AccountID: "ACC-001", Currency: currency(t, "AED", 2),
			BookingDay: 6, ValueDay: 2, ReversesEventID: "E7", Order: 9,
		}}},
		{Key: "attempt/5", Attempt: &domain.InputAttempt{ID: "attempt-5", Input: domain.InputEvent{
			ID: "E10", Kind: domain.InputCredit, AccountID: "ACC-002", Currency: bhd,
			RawAmount: "10.000", BookingDay: 5, ValueDay: 5, Order: 10, InstalmentCount: 3,
		}}},
	}
	require.NoError(t, repository.Append(domain.Batch{Records: later}))
	assert.Equal(t, append(want, later...), repository.History().Records)
	assert.Equal(t, records(t), snapshot.Records, "an earlier snapshot cannot grow or change")
}

func TestHistoryOwnership(t *testing.T) {
	repository := memory.New()
	submitted := domain.Batch{Records: records(t)}
	want := records(t) // Independent values, not a copy through production Clone.
	require.NoError(t, repository.Append(submitted))

	mutate(submitted.Records)
	submitted.Records[0] = domain.Record{Key: "replaced"}
	submitted.Records = append(submitted.Records, domain.Record{Key: "added"})
	assert.Equal(t, want, repository.History().Records)

	returned := repository.History()
	mutate(returned.Records)
	returned.Records[0] = domain.Record{Key: "replaced again"}
	returned.Records = append(returned.Records, domain.Record{Key: "added again"})
	assert.Equal(t, want, repository.History().Records)

	// Two simultaneous readers also own independent payload pointers.
	first, second := repository.History(), repository.History()
	mutate(first.Records)
	assert.Equal(t, want, second.Records)
}

func mutate(records []domain.Record) {
	for i := range records {
		r := &records[i]
		r.Key = "changed"
		if r.Account != nil {
			*r.Account = domain.Account{}
		}
		if r.Attempt != nil {
			r.Attempt.Input.RawAmount = "changed raw amount"
			r.Attempt.ID = "changed attempt"
		}
		if r.Outcome != nil {
			*r.Outcome = domain.InputOutcome{}
		}
		if r.Entry != nil {
			*r.Entry = domain.MonetaryEntry{}
		}
		if r.Authorization != nil {
			*r.Authorization = domain.AuthorizationTransition{}
		}
		if r.Diagnostic != nil {
			*r.Diagnostic = domain.Diagnostic{}
		}
		if r.Close != nil {
			*r.Close = domain.DayClose{}
		}
	}
}

func closeRecord(key domain.RecordKey) domain.Record {
	return domain.Record{Key: key, Close: &domain.DayClose{AccountID: "ACC-001", Day: 1}}
}

func TestAppendAtomicValidation(t *testing.T) {
	for _, tc := range []struct {
		name string
		bad  []domain.Record
		err  error
	}{
		{"stored key conflict across payload types", []domain.Record{closeRecord("account/ACC-001")}, memory.ErrDuplicateKey},
		{"within batch conflict", []domain.Record{closeRecord("collision"), closeRecord("collision")}, memory.ErrDuplicateKey},
		{"empty key", []domain.Record{closeRecord("")}, memory.ErrInvalidKey},
		{"no payload", []domain.Record{{Key: "invalid"}}, memory.ErrInvalidRecord},
		{"multiple payloads", []domain.Record{{Key: "invalid", Close: &domain.DayClose{}, Account: &domain.Account{}}}, memory.ErrInvalidRecord},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repository := memory.New()
			initial := records(t)
			require.NoError(t, repository.Append(domain.Batch{Records: initial}))
			before := repository.History()

			proposed := []domain.Record{closeRecord("before-conflict")}
			proposed = append(proposed, tc.bad...)
			proposed = append(proposed, closeRecord("after-conflict"))
			require.ErrorIs(t, repository.Append(domain.Batch{Records: proposed}), tc.err)
			assert.Equal(t, before, repository.History(), "failure appends none of the batch")

			// Failure must not reserve keys from the valid prefix or suffix.
			recovered := []domain.Record{closeRecord("before-conflict"), closeRecord("after-conflict"), closeRecord("collision"), closeRecord("invalid")}
			require.NoError(t, repository.Append(domain.Batch{Records: recovered}))
			assert.Equal(t, append(initial, recovered...), repository.History().Records)
		})
	}
}

func TestRecordKeysAreIndependentOfSuppliedIDs(t *testing.T) {
	repository := memory.New()
	original := records(t)
	require.NoError(t, repository.Append(domain.Batch{Records: original}))

	// These are already-decided rejection records, not a test of the later
	// duplicate-input policy. Both attempts refer to E1 using fresh identities.
	duplicates := []domain.Record{
		{Key: "duplicate-attempt/1", Attempt: &domain.InputAttempt{ID: "duplicate-1", Input: original[1].Attempt.Input}},
		{Key: "duplicate-outcome/1", Outcome: &domain.InputOutcome{
			AttemptID: "duplicate-1", Status: domain.OutcomeRejected, DiagnosticKey: "duplicate-diagnostic/1",
		}},
		{Key: "duplicate-diagnostic/1", Diagnostic: &domain.Diagnostic{
			Code: "duplicate_input", Reason: "input ID was already submitted", AccountID: "ACC-001",
			ObservationDay: 1, AttemptID: "duplicate-1", CauseKey: "duplicate-attempt/1",
		}},
		{Key: "duplicate-attempt/2", Attempt: &domain.InputAttempt{ID: "duplicate-2", Input: original[1].Attempt.Input}},
		{Key: "duplicate-outcome/2", Outcome: &domain.InputOutcome{
			AttemptID: "duplicate-2", Status: domain.OutcomeRejected, DiagnosticKey: "duplicate-diagnostic/2",
		}},
		{Key: "duplicate-diagnostic/2", Diagnostic: &domain.Diagnostic{
			Code: "duplicate_input", Reason: "input ID was already submitted", AccountID: "ACC-001",
			ObservationDay: 1, AttemptID: "duplicate-2", CauseKey: "duplicate-attempt/2",
		}},
	}
	duplicates[3].Attempt.Input.RawAmount = "950.00"
	require.NoError(t, repository.Append(domain.Batch{Records: duplicates}))
	assert.Equal(t, append(original, duplicates...), repository.History().Records)
}

func TestStorageDoesNotApplyBusinessValidation(t *testing.T) {
	repository := memory.New()
	// Missing configuration, unparsed principal, and unresolved links must stay
	// recordable. Their validation/outcomes belong to ledger, not the adapter.
	batch := domain.Batch{Records: []domain.Record{
		{Key: "account", Account: &domain.Account{}},
		{Key: "attempt", Attempt: &domain.InputAttempt{ID: "attempt", Input: domain.InputEvent{
			ID: "bad-input", RawAmount: "invalid principal", AccountID: "absent-account",
		}}},
		{Key: "entry", Entry: &domain.MonetaryEntry{OriginalEntryKey: "absent-entry"}},
	}}
	require.NoError(t, repository.Append(batch))
	assert.Equal(t, batch.Records, repository.History().Records)
}
