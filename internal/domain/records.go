package domain

// RecordKey identifies a stored record independently of any supplied event ID.
type RecordKey string

// AttemptID identifies one submission, including a rejected duplicate input.
type AttemptID string

type EventID string
type AccountID string
type AuthorizationID string
type FeeObligationID string

// Day is a value or booking day in the configured replay window. Storage does
// not impose a window; zero denotes an inapplicable optional day field.
type Day int

// Account carries explicit configuration; validation across accounts and
// policies belongs to core initialization, not storage.
type Account struct {
	ID             AccountID
	Currency       Currency
	OpeningBalance Money
}

type InputKind string

const (
	InputCredit        InputKind = "CREDIT"
	InputDebit         InputKind = "DEBIT"
	InputAuthorization InputKind = "AUTHORIZATION"
	InputSettlement    InputKind = "SETTLEMENT"
	InputReversal      InputKind = "REVERSAL"
)

// InputEvent preserves supplied fields before monetary parsing or validation.
// Order is the supplied stream position, independent of dates and attempt IDs.
// InstalmentCount is set only when the input requests instalment posting.
type InputEvent struct {
	ID              EventID
	Kind            InputKind
	AccountID       AccountID
	Currency        Currency
	RawAmount       string
	BookingDay      Day
	ValueDay        Day
	AuthorizationID AuthorizationID
	ReversesEventID EventID
	Order           int
	InstalmentCount int
}

type InputAttempt struct {
	ID    AttemptID
	Input InputEvent
}

type OutcomeStatus string

const (
	OutcomeAccepted OutcomeStatus = "ACCEPTED"
	OutcomeDeclined OutcomeStatus = "DECLINED"
	OutcomeRejected OutcomeStatus = "REJECTED"
)

// InputOutcome refers to one attempt rather than a potentially reused event ID.
type InputOutcome struct {
	AttemptID     AttemptID
	Status        OutcomeStatus
	DiagnosticKey RecordKey
}

type EntryKind string

const (
	EntryCredit         EntryKind = "CREDIT"
	EntryDebit          EntryKind = "DEBIT"
	EntrySettlement     EntryKind = "SETTLEMENT"
	EntryReversal       EntryKind = "REVERSAL"
	EntryInstalment     EntryKind = "INSTALMENT"
	EntryFeeAssessment  EntryKind = "FEE_ASSESSMENT"
	EntryFeeRefund      EntryKind = "FEE_REFUND"
	EntryFeeAdjustment  EntryKind = "FEE_ADJUSTMENT"
	EntryCapitalization EntryKind = "CAPITALIZATION"
)

// MonetaryEntry contains a signed posting. SourceAttemptID links an originating
// input; CauseKey links the record triggering an internal posting/correction.
// OriginalEntryKey links a reversal/refund to its original entry, while
// FeeObligationID remains stable across all adjustments for an account/day.
// Instalment positions are one-based; zero means inapplicable.
type MonetaryEntry struct {
	AccountID        AccountID
	Amount           Money
	BookingDay       Day
	ValueDay         Day
	Kind             EntryKind
	SourceAttemptID  AttemptID
	CauseKey         RecordKey
	OriginalEntryKey RecordKey
	FeeObligationID  FeeObligationID
	Instalment       int
	InstalmentCount  int
}

type AuthorizationState string

const (
	AuthorizationApproved AuthorizationState = "APPROVED"
	AuthorizationDeclined AuthorizationState = "DECLINED"
	AuthorizationSettled  AuthorizationState = "SETTLED"
)

// AuthorizationTransition records the original decision or final settlement.
// Hold is the approved/declined request amount; OriginalHoldKey links a
// settlement to the recorded approval whose full hold is released.
type AuthorizationTransition struct {
	AccountID       AccountID
	AuthorizationID AuthorizationID
	AttemptID       AttemptID
	BookingDay      Day
	ValueDay        Day
	State           AuthorizationState
	Hold            Money
	OriginalHoldKey RecordKey
}

// Diagnostic retains a reason at its original observation checkpoint.
// AssessedDay is optional for fee-related diagnostics; CauseKey can identify
// an internal correction independently of an input attempt.
type Diagnostic struct {
	Code           string
	Reason         string
	AccountID      AccountID
	ObservationDay Day
	AttemptID      AttemptID
	CauseKey       RecordKey
	AssessedDay    Day
}

// DayClose records that an account/day has been explicitly closed. Rounded
// unpaid accruals are derived projections, not stored monetary entries.
type DayClose struct {
	AccountID AccountID
	Day       Day
}

// Record is a keyed envelope with exactly one non-nil payload. Payloads contain
// only value fields. Callers may edit their own records; repository ownership
// begins with a detached copy and every history read returns another copy.
type Record struct {
	Key           RecordKey
	Account       *Account
	Attempt       *InputAttempt
	Outcome       *InputOutcome
	Entry         *MonetaryEntry
	Authorization *AuthorizationTransition
	Diagnostic    *Diagnostic
	Close         *DayClose
}

// Clone detaches every payload pointer. Any future mutable payload fields must
// also be copied here to preserve repository ownership.
func (r Record) Clone() Record {
	r.Account = cloneValue(r.Account)
	r.Attempt = cloneValue(r.Attempt)
	r.Outcome = cloneValue(r.Outcome)
	r.Entry = cloneValue(r.Entry)
	r.Authorization = cloneValue(r.Authorization)
	r.Diagnostic = cloneValue(r.Diagnostic)
	r.Close = cloneValue(r.Close)
	return r
}

func cloneValue[T any](value *T) *T {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

// Batch is one ordered, atomic append unit. It is not a whole-operation
// transaction when processing requires several appends.
type Batch struct {
	Records []Record
}

// History is an ordered, detached snapshot of all committed records.
type History struct {
	Records []Record
}
