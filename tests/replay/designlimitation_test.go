//go:build designlimitation

package replay_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"inmemoryaccountledgercore/internal/domain"
	"inmemoryaccountledgercore/internal/ledger"
	"inmemoryaccountledgercore/internal/projection"
	"inmemoryaccountledgercore/internal/repository/memory"
)

func TestDesignLimitationFailedSubmissionIsFinanciallyAtomic(t *testing.T) {
	aed, err := domain.NewCurrency("AED", 2)
	require.NoError(t, err)
	zero, err := domain.NewMoney(aed, 0)
	require.NoError(t, err)
	charge, err := domain.NewMoney(aed, 2500)
	require.NoError(t, err)
	interruption := errors.New("interrupted Day 3 fee append")
	repository := &interruptedFeeRepository{Repository: memory.New(), failure: interruption}
	core, err := ledger.New(repository, []domain.Account{
		{ID: "ACC-001", Currency: aed, OpeningBalance: zero},
	}, map[domain.Currency]domain.Money{aed: charge})
	require.NoError(t, err)
	require.NoError(t, core.CloseDay("ACC-001", 2, 2, "checkpoint/2"))
	require.NoError(t, core.CloseDay("ACC-001", 3, 3, "checkpoint/3"))
	before := repository.History()
	state := replayProjection(t, before)
	beforeBalance, err := state.Balance("ACC-001", 3, projection.BalanceOptions{})
	require.NoError(t, err)
	require.Zero(t, beforeBalance.MinorUnits())
	input := domain.InputEvent{
		ID: "late-debit", Kind: domain.InputDebit, AccountID: "ACC-001",
		Currency: aed, RawAmount: "0.01", BookingDay: 5, ValueDay: 2,
	}

	// Fail before the second fee batch reaches real memory storage. The input
	// and Day 2 fee have already committed; the repository is then healthy.
	repository.armed = true
	outcome, err := core.Submit(input)
	require.ErrorIs(t, err, interruption)
	require.Equal(t, domain.InputOutcome{}, outcome)
	require.False(t, repository.armed, "the transient failure was exercised")
	interrupted := repository.History()
	require.Equal(t, before.Records, interrupted.Records[:len(before.Records)])
	state = replayProjection(t, interrupted)
	afterBalance, err := state.Balance(input.AccountID, 3, projection.BalanceOptions{})
	require.NoError(t, err)

	// Deliberately failing design probe: financial atomicity of one public
	// Submit call would leave the prior zero balance unchanged on technical
	// failure. This useful guarantee lets callers handle an error without
	// observing principal with only partially reconciled historical fees.
	// The repository commits each batch separately: -0.01 principal and the
	// -25.00 Day 2 fee survive the failed Day 3 append, leaving -25.01 AED.
	// Whole-submission atomicity exceeds the current contract; this exposes
	// that engineering trade-off without changing any business rule. There is
	// no duplicate submission or recovery before this assertion. Explicit
	// recovery and its idempotence are covered by normal backdating regressions.
	// See docs/DESIGN.md "Repository boundary" and NUMBERS
	// "Interrupted reconciliation probe".
	assert.Equal(t, int64(0), afterBalance.MinorUnits(),
		"failed Submit should leave the pre-submission financial balance unchanged")
}

// All successful appends and history reads use the real memory adapter. Only
// the selected append is interrupted, atomically, before storage is called.
type interruptedFeeRepository struct {
	*memory.Repository
	failure error
	armed   bool
}

func (r *interruptedFeeRepository) Append(batch domain.Batch) error {
	if r.armed {
		for _, record := range batch.Records {
			if entry := record.Entry; entry != nil && entry.Kind == domain.EntryFeeAssessment && entry.ValueDay == 3 {
				r.armed = false
				return r.failure
			}
		}
	}
	return r.Repository.Append(batch)
}
