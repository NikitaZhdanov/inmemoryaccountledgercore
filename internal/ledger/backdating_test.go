package ledger

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"inmemoryaccountledgercore/internal/domain"
	"inmemoryaccountledgercore/internal/posting"
)

func TestSubmissionReconciliationAppendBoundaries(t *testing.T) {
	for failOffset := 1; failOffset <= 3; failOffset++ {
		t.Run(fmt.Sprintf("append %d", failOffset), func(t *testing.T) {
			core, journal, input := ordinaryFixture(t, 0)
			failure := errors.New("submission or fee append unavailable")
			repository := &interruptedClosingRepository{journal: *journal, failure: failure}
			core.repository = repository
			for day := domain.Day(1); day <= 3; day++ {
				require.NoError(t, core.CloseDay(input.AccountID, day, day, domain.RecordKey(fmt.Sprintf("close/%d", day))))
			}
			before := repository.History()
			input.Kind, input.RawAmount = domain.InputDebit, "0.01"
			repository.failAt = repository.appends + failOffset
			outcome, err := core.Submit(input)
			require.ErrorIs(t, err, failure)
			assert.Equal(t, domain.InputOutcome{}, outcome)
			after := repository.History()
			assert.Equal(t, before.Records, after.Records[:len(before.Records)])
			if failOffset == 1 {
				assert.Equal(t, before, after, "failed input batch leaves no records")
				return
			}
			require.Len(t, after.Records, len(before.Records)+3+failOffset-2)
			assert.Equal(t, domain.OutcomeAccepted, after.Records[len(before.Records)+1].Outcome.Status)
			fees := feeRecords(after)
			require.Len(t, fees, failOffset-2)
			for _, record := range after.Records {
				assert.Nil(t, record.Diagnostic, "technical failure is not a rejection")
			}
			cause := domain.RecordKey(after.Records[len(before.Records)].Attempt.ID)
			if failOffset == 3 {
				assert.Equal(t, domain.Day(2), fees[0].Entry.ValueDay)
				assert.Equal(t, cause, fees[0].Entry.CauseKey)
				requireBalance(t, repository, input.AccountID, 3, -2501)
			} else {
				requireBalance(t, repository, input.AccountID, 3, -1)
			}
			// Even while a correction is incomplete, duplicate and invalid
			// submissions must not perform outstanding financial work.
			repository.failAt = 0
			for _, changed := range []bool{false, true} {
				duplicate := input
				if changed {
					duplicate.AccountID, duplicate.BookingDay = "unknown", 0
					duplicate.Kind, duplicate.RawAmount = "unsupported", "invalid"
				}
				start, appends := len(repository.records), repository.appends
				rejected, err := core.Submit(duplicate)
				require.NoError(t, err)
				assert.Equal(t, domain.OutcomeRejected, rejected.Status)
				requireAttempt(t, repository, start, duplicate, rejected)
				assert.Equal(t, posting.DiagnosticDuplicateID, repository.records[start+2].Diagnostic.Code)
				assert.Equal(t, appends+1, repository.appends)
				assert.Equal(t, fees, feeRecords(repository.History()))
			}
			invalid := input
			invalid.ID, invalid.RawAmount = "invalid", "1.005"
			start := len(repository.records)
			rejected, err := core.Submit(invalid)
			require.NoError(t, err)
			assert.Equal(t, domain.OutcomeRejected, rejected.Status)
			requireAttempt(t, repository, start, invalid, rejected)
			assert.Equal(t, fees, feeRecords(repository.History()))
			// Explicit reconciliation completes only the missing fees; the
			// original accepted attempt and any committed fee remain intact.
			require.NoError(t, core.ReconcileClosedDays(input.AccountID, 2, 5, cause))
			assert.Equal(t, after.Records, repository.History().Records[:len(after.Records)])
			require.Len(t, feeRecords(repository.History()), 2)
			requireBalance(t, repository, input.AccountID, 1, 0)
			requireBalance(t, repository, input.AccountID, 2, -2501)
			requireBalance(t, repository, input.AccountID, 3, -5001)
			complete, appends := repository.History(), repository.appends
			require.NoError(t, core.ReconcileClosedDays(input.AccountID, 2, 5, cause))
			assert.Equal(t, complete, repository.History())
			assert.Equal(t, appends, repository.appends)
		})
	}
}

func TestSubmissionFeeOverflowRetainsCommittedBatches(t *testing.T) {
	for _, tc := range []struct {
		name, amount string
		wantFees     int
	}{
		{"first fee underflows", "92233720368547758.07", 0},
		{"second fee underflows", "92233720368547733.08", 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			core, repository, input := ordinaryFixture(t, 0)
			for _, day := range []domain.Day{2, 3} {
				require.NoError(t, core.CloseDay(input.AccountID, day, day, domain.RecordKey(fmt.Sprintf("close/%d", day))))
			}
			before := repository.History()
			input.Kind, input.RawAmount = domain.InputDebit, tc.amount
			outcome, err := core.Submit(input)
			require.ErrorIs(t, err, domain.ErrArithmeticOverflow)
			assert.Equal(t, domain.InputOutcome{}, outcome)
			after := repository.History()
			require.Len(t, after.Records, len(before.Records)+3+tc.wantFees)
			assert.Equal(t, before.Records, after.Records[:len(before.Records)])
			assert.Equal(t, domain.OutcomeAccepted, after.Records[len(before.Records)+1].Outcome.Status)
			require.Len(t, feeRecords(after), tc.wantFees)
			for _, record := range after.Records {
				assert.Nil(t, record.Diagnostic)
			}
		})
	}
}
