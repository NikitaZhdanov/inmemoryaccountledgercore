package replay_test

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"inmemoryaccountledgercore/internal/domain"
	"inmemoryaccountledgercore/internal/report"
	"inmemoryaccountledgercore/internal/scenario"
)

func TestCanonicalReport(t *testing.T) {
	// This literal oracle combines NUMBERS' six-day balances/accruals with
	// SPEC's recorded decisions and assessed-day grouping. It is not generated
	// by production calculations or by capturing the executable's output.
	want, err := os.ReadFile("testdata/canonical_report.txt")
	require.NoError(t, err)
	var first string
	for i := 0; i < 2; i++ {
		repository, core, primary, instalment := canonicalComponents(t)
		require.NoError(t, scenario.Run(core, primary, instalment))
		history := repository.History()
		state := replayProjection(t, history)
		got, err := report.Format(history, state, []domain.Day{1, 2, 3, 4, 5, 6}, []domain.AccountID{primary.ID, instalment.ID})
		require.NoError(t, err)
		assert.Equal(t, string(want), got)
		assert.Equal(t, history, repository.History(), "report must not append or mutate records")
		if i == 0 {
			first = got
		} else {
			assert.Equal(t, first, got, "fresh runs must print identically")
		}
	}
}

func TestReportRetainsAmountValidationReason(t *testing.T) {
	repository, core, primary, instalment := canonicalComponents(t)
	_, err := core.Submit(domain.InputEvent{
		ID: "invalid-precision", Kind: domain.InputCredit, AccountID: primary.ID,
		Currency: primary.Currency, RawAmount: "1.005", BookingDay: 1, ValueDay: 1,
	})
	require.NoError(t, err)
	require.NoError(t, scenario.Run(core, primary, instalment))
	history := repository.History()
	got, err := report.Format(history, replayProjection(t, history), []domain.Day{1, 2}, []domain.AccountID{primary.ID})
	require.NoError(t, err)
	assert.Contains(t, got, "1 | ACC-001 | AED | 250.00 | 0.10")
	assert.Contains(t, got, "invalid-precision REJECTED [AMOUNT_PRECISION]: amount exceeds currency precision")
	assert.Contains(t, got, "2 | ACC-001 | AED | 250.00 | 0.10")
	assert.Equal(t, 1, strings.Count(got, "invalid-precision REJECTED"))
}
