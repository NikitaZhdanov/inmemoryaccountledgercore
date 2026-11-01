package replay_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"inmemoryaccountledgercore/internal/domain"
	"inmemoryaccountledgercore/internal/ledger"
	"inmemoryaccountledgercore/internal/posting"
	"inmemoryaccountledgercore/internal/projection"
	"inmemoryaccountledgercore/internal/repository/memory"
)

// Literal inputs and checkpoints are independent TASK E1–E6 / NUMBERS oracles.
// Explicit closes and historical fee reconciliation are introduced later.
func TestAuthorizationAndFinalSettlement(t *testing.T) {
	aed, err := domain.NewCurrency("AED", 2)
	require.NoError(t, err)
	zero, err := domain.NewMoney(aed, 0)
	require.NoError(t, err)
	fee, err := domain.NewMoney(aed, 2500)
	require.NoError(t, err)
	repository := memory.New()
	core, err := ledger.New(repository,
		[]domain.Account{{ID: "ACC-001", Currency: aed, OpeningBalance: zero}},
		map[domain.Currency]domain.Money{aed: fee})
	require.NoError(t, err)
	var approval domain.Record
	for _, tc := range []struct {
		id            domain.EventID
		kind          domain.InputKind
		raw           string
		day           domain.Day
		order         int
		auth          domain.AuthorizationID
		status        domain.OutcomeStatus
		balance, hold int64
		available     int64
	}{
		{"E1", domain.InputCredit, "1200.00", 1, 1, "", domain.OutcomeAccepted, 120000, 0, 120000},
		{"E2", domain.InputDebit, "950.00", 1, 2, "", domain.OutcomeAccepted, 25000, 0, 25000},
		{"E3", domain.InputAuthorization, "200.00", 2, 3, "Auth-A", domain.OutcomeAccepted, 25000, 20000, 5000},
		{"E4", domain.InputCredit, "400.00", 3, 4, "", domain.OutcomeAccepted, 65000, 20000, 45000},
		{"E5", domain.InputSettlement, "185.00", 4, 5, "Auth-A", domain.OutcomeAccepted, 46500, 0, 46500},
		{"E6", domain.InputSettlement, "180.00", 4, 6, "Auth-Z", domain.OutcomeRejected, 46500, 0, 46500},
	} {
		t.Run(string(tc.id), func(t *testing.T) {
			input := domain.InputEvent{
				ID: tc.id, Kind: tc.kind, AccountID: "ACC-001", Currency: aed, RawAmount: tc.raw,
				BookingDay: tc.day, ValueDay: tc.day, AuthorizationID: tc.auth, Order: tc.order,
			}
			before := repository.History()
			outcome, err := core.Submit(input)
			require.NoError(t, err)
			assert.Equal(t, tc.status, outcome.Status)
			history := repository.History()
			assert.Equal(t, before.Records, history.Records[:len(before.Records)])
			assert.Equal(t, input, history.Records[len(before.Records)].Attempt.Input)
			state, err := projection.Rebuild(history, len(history.Records))
			require.NoError(t, err)
			balance, err := state.Balance(input.AccountID, tc.day, projection.BalanceOptions{})
			require.NoError(t, err)
			assert.Equal(t, tc.balance, balance.MinorUnits())
			hold, err := state.ActiveHolds(input.AccountID)
			require.NoError(t, err)
			assert.Equal(t, tc.hold, hold.MinorUnits())
			available, err := state.AvailableBalance(input.AccountID, tc.day)
			require.NoError(t, err)
			assert.Equal(t, tc.available, available.MinorUnits())
			if tc.id == "E3" {
				approval = history.Records[len(history.Records)-1]
			}
			if tc.id == "E5" {
				records := history.Records[len(before.Records):]
				require.Len(t, records, 4)
				assert.Equal(t, int64(-18500), records[2].Entry.Amount.MinorUnits())
				assert.Equal(t, approval.Key, records[3].Authorization.OriginalHoldKey)
				assert.Equal(t, approval.Authorization.Hold, records[3].Authorization.Hold)
				auth, exists := state.Authorization(input.AccountID, "Auth-A")
				require.True(t, exists)
				assert.Equal(t, domain.AuthorizationSettled, auth.Transition.State)
			}
			if tc.id == "E6" {
				records := history.Records[len(before.Records):]
				require.Len(t, records, 3)
				assert.Equal(t, outcome.DiagnosticKey, records[2].Key)
				assert.Equal(t, posting.DiagnosticUnknownAuthorization, records[2].Diagnostic.Code)
				assert.Equal(t, domain.Day(4), records[2].Diagnostic.ObservationDay)
				_, exists := state.Authorization(input.AccountID, "Auth-Z")
				assert.False(t, exists)
			}
		})
	}
	// A fresh valid event after E6 proves covered rejection does not stop input
	// processing; this one-minor-unit credit is outside the canonical stream.
	outcome, err := core.Submit(domain.InputEvent{
		ID: "continuation", Kind: domain.InputCredit, AccountID: "ACC-001", Currency: aed,
		RawAmount: "0.01", BookingDay: 5, ValueDay: 5,
	})
	require.NoError(t, err)
	assert.Equal(t, domain.OutcomeAccepted, outcome.Status)
	history := repository.History()
	state, err := projection.Rebuild(history, len(history.Records))
	require.NoError(t, err)
	balance, err := state.Balance("ACC-001", 5, projection.BalanceOptions{})
	require.NoError(t, err)
	assert.Equal(t, int64(46501), balance.MinorUnits())
}
