package posting

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"inmemoryaccountledgercore/internal/domain"
)

func TestPrepareLeavesHistoryUnchanged(t *testing.T) {
	for _, tc := range []struct {
		name       string
		kind       domain.InputKind
		amount     string
		opening    int64
		status     domain.OutcomeStatus
		diagnostic string
		wantErr    error
	}{
		{"credit", domain.InputCredit, "200.00", 25000, domain.OutcomeAccepted, "", nil},
		{"approved hold", domain.InputAuthorization, "200.00", 25000, domain.OutcomeAccepted, "", nil},
		{"declined hold", domain.InputAuthorization, "0.01", 0, domain.OutcomeDeclined, DiagnosticInsufficientAvailability, nil},
		{"negative hold", domain.InputAuthorization, "-0.01", 25000, domain.OutcomeRejected, DiagnosticNegativeAmount, nil},
		{"overflow", domain.InputCredit, "0.01", math.MaxInt64, "", "", domain.ErrArithmeticOverflow},
	} {
		t.Run(tc.name, func(t *testing.T) {
			history, input := authorizationHistory(t)
			account := *history.Records[0].Account
			account.OpeningBalance = money(t, account.Currency, tc.opening)
			history.Records[0].Account = &account
			input.Kind, input.RawAmount = tc.kind, tc.amount
			before := domain.History{Records: []domain.Record{history.Records[0].Clone()}}

			batch, outcome, err := Prepare(history, account, input)
			assert.Equal(t, before, history, "preparation must preserve the supplied records")
			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)
				assert.Equal(t, domain.Batch{}, batch)
				assert.Equal(t, domain.InputOutcome{}, outcome)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.status, outcome.Status)
			require.NotEmpty(t, batch.Records)
			require.NotNil(t, batch.Records[0].Attempt)
			assert.Equal(t, input, batch.Records[0].Attempt.Input)
			if tc.diagnostic != "" {
				diagnostic := batch.Records[len(batch.Records)-1]
				require.NotNil(t, diagnostic.Diagnostic)
				assert.Equal(t, tc.diagnostic, diagnostic.Diagnostic.Code)
				assert.Equal(t, outcome.DiagnosticKey, diagnostic.Key)
			}
			// Until a caller commits the batch, preparation reserves no IDs,
			// holds, or postings in the supplied history.
			repeated, repeatedOutcome, err := Prepare(history, account, input)
			require.NoError(t, err)
			assert.Equal(t, batch, repeated)
			assert.Equal(t, outcome, repeatedOutcome)
			assert.Equal(t, before, history)
		})
	}
}
