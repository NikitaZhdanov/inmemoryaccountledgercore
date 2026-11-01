package projection_test

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"inmemoryaccountledgercore/internal/domain"
	"inmemoryaccountledgercore/internal/projection"
)

func TestValidateBatchPreservesHistoryBackingSlice(t *testing.T) {
	for _, tc := range []struct {
		name    string
		opening int64
	}{
		{"valid posting", 0},
		{"overflow", math.MaxInt64},
	} {
		t.Run(tc.name, func(t *testing.T) {
			currency := canonicalCurrencies(t)[0]
			history := domain.History{Records: []domain.Record{{
				Key: "account/ACC-001", Account: &domain.Account{
					ID: "ACC-001", Currency: currency, OpeningBalance: money(t, currency, tc.opening),
				},
			}}}
			// Another view owns the record beyond the history prefix. Previewing
			// must preserve it even when append could reuse the spare capacity.
			sentinel := domain.Record{Key: "sentinel", Diagnostic: &domain.Diagnostic{Code: "PRESERVE"}}
			backing := []domain.Record{history.Records[0], sentinel}
			history.Records = backing[:1]
			before := []domain.Record{backing[0].Clone(), backing[1].Clone()}
			posting := domain.Record{Key: "preview", Entry: &domain.MonetaryEntry{
				AccountID: "ACC-001", Amount: money(t, currency, 1), ValueDay: 2,
			}}
			err := projection.ValidateBatch(history, domain.Batch{Records: []domain.Record{posting}}, "ACC-001", 2)
			if tc.opening == math.MaxInt64 {
				require.ErrorIs(t, err, domain.ErrArithmeticOverflow)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, before, backing, "validation must not overwrite records outside the history prefix")
		})
	}
}
