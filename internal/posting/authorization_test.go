package posting

import (
	"fmt"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"inmemoryaccountledgercore/internal/domain"
	"inmemoryaccountledgercore/internal/projection"
)

func authorizationHistory(t *testing.T) (domain.History, domain.InputEvent) {
	t.Helper()
	currency := canonicalCurrencies(t)[0]
	history := domain.History{Records: []domain.Record{{
		Key: "account/ACC-001", Account: &domain.Account{
			ID: "ACC-001", Currency: currency, OpeningBalance: money(t, currency, 25000),
		},
	}}}
	input := domain.InputEvent{
		ID: "authorization", Kind: domain.InputAuthorization, AccountID: "ACC-001", Currency: currency,
		AuthorizationID: "Auth-A", BookingDay: 2, ValueDay: 2, RawAmount: "200.00",
	}
	return history, input
}

func approvedHold(t *testing.T, input domain.InputEvent) domain.Record {
	t.Helper()
	return domain.Record{Key: "hold", Authorization: &domain.AuthorizationTransition{
		AccountID: input.AccountID, AuthorizationID: input.AuthorizationID,
		State: domain.AuthorizationApproved, Hold: money(t, input.Currency, 20000),
		BookingDay: 2, ValueDay: 2,
	}}
}

func TestAuthorizationBuildersRejectNegativeAmounts(t *testing.T) {
	for _, kind := range []domain.InputKind{domain.InputAuthorization, domain.InputSettlement} {
		for _, minor := range []int64{-1, math.MinInt64} {
			t.Run(fmt.Sprintf("%s/%d", kind, minor), func(t *testing.T) {
				history, input := authorizationHistory(t)
				var holds int64
				if kind == domain.InputSettlement {
					history.Records = append(history.Records, approvedHold(t, input))
					input.ID = "settlement"
					holds = 20000
				}
				input.Kind = kind
				before := domain.History{Records: make([]domain.Record, len(history.Records))}
				for i, record := range history.Records {
					before.Records[i] = record.Clone()
				}
				attempt, _ := prepareAttempt(before, input)
				amount := money(t, input.Currency, minor)
				builder := authorizationBatch
				if kind == domain.InputSettlement {
					builder = settlementBatch
				}
				// Bypass Prepare's parser to exercise each workflow's own guard.
				batch, outcome, err := builder(before, attempt, amount)
				require.NoError(t, err)
				assert.Equal(t, domain.OutcomeRejected, outcome.Status)
				require.Len(t, batch.Records, 3)
				assert.Equal(t, attempt, *batch.Records[0].Attempt)
				assert.Equal(t, outcome, *batch.Records[1].Outcome)
				assert.Equal(t, DiagnosticNegativeAmount, batch.Records[2].Diagnostic.Code)
				assert.Equal(t, outcome.DiagnosticKey, batch.Records[2].Key)
				for _, record := range batch.Records {
					assert.Nil(t, record.Entry, "negative settlement must not produce a credit")
					assert.Nil(t, record.Authorization, "negative request must not create or release a hold")
				}
				assert.Equal(t, before, history, "building a batch cannot change history")
				state, err := projection.Rebuild(history, len(history.Records))
				require.NoError(t, err)
				active, err := state.ActiveHolds(input.AccountID)
				require.NoError(t, err)
				assert.Equal(t, holds, active.MinorUnits())
				available, err := state.AvailableBalance(input.AccountID, 2)
				require.NoError(t, err)
				assert.Equal(t, 25000-holds, available.MinorUnits())
			})
		}
	}
}
