package main

import (
	"bytes"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"inmemoryaccountledgercore/internal/domain"
	"inmemoryaccountledgercore/internal/ledger"
)

func TestCanonicalConfiguration(t *testing.T) {
	config, err := canonicalConfiguration()
	require.NoError(t, err)
	require.Len(t, config.charges, 2)
	for _, tc := range []struct {
		account domain.Account
		id      domain.AccountID
		code    string
		scale   int
		charge  int64
	}{
		{config.primary, "ACC-001", "AED", 2, 2500},
		{config.instalment, "ACC-002", "BHD", 3, 0},
	} {
		assert.Equal(t, tc.id, tc.account.ID)
		assert.Equal(t, tc.code, tc.account.Currency.Code())
		scale, err := tc.account.Currency.Scale()
		require.NoError(t, err)
		assert.Equal(t, tc.scale, scale)
		assert.Equal(t, int64(0), tc.account.OpeningBalance.MinorUnits())
		assert.Equal(t, tc.account.Currency, tc.account.OpeningBalance.Currency())
		charge, exists := config.charges[tc.account.Currency]
		require.True(t, exists, "zero-charge policy must be present")
		assert.Equal(t, tc.charge, charge.MinorUnits())
		assert.Equal(t, tc.account.Currency, charge.Currency())
	}
}

func TestRunFreshReplayAndOutputFailure(t *testing.T) {
	var first, second bytes.Buffer
	require.NoError(t, run(&first))
	require.NoError(t, run(&second))
	assert.Equal(t, first.String(), second.String())
	assert.Contains(t, first.String(), "6 | ACC-001 | AED | 466.03 | 0.19")
	assert.Contains(t, first.String(), "6 | ACC-002 | BHD | 10.008 | 0.004")
	failure := errors.New("output unavailable")
	require.ErrorIs(t, run(failingWriter{failure}), failure)
}

func TestInvalidCompositionStopsBeforeReplayAndReport(t *testing.T) {
	for _, name := range []string{"missing primary policy", "missing no-fee policy", "invalid descriptor", "conflicting scale", "negative charge", "mismatched charge"} {
		t.Run(name, func(t *testing.T) {
			config, err := canonicalConfiguration()
			require.NoError(t, err)
			switch name {
			case "missing primary policy":
				delete(config.charges, config.primary.Currency)
			case "missing no-fee policy":
				delete(config.charges, config.instalment.Currency)
			case "invalid descriptor":
				config.primary.Currency = domain.Currency{}
			case "conflicting scale":
				config.instalment.Currency, err = domain.NewCurrency("AED", 3)
				require.NoError(t, err)
			case "negative charge":
				config.charges[config.primary.Currency], err = domain.NewMoney(config.primary.Currency, -1)
				require.NoError(t, err)
			case "mismatched charge":
				config.charges[config.primary.Currency] = config.charges[config.instalment.Currency]
			}
			var out bytes.Buffer
			require.ErrorIs(t, replay(&out, config), ledger.ErrInvalidConfiguration)
			assert.Empty(t, out.String())
		})
	}
}

type failingWriter struct{ err error }

func (w failingWriter) Write([]byte) (int, error) { return 0, w.err }
