package scenario_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"inmemoryaccountledgercore/internal/domain"
	"inmemoryaccountledgercore/internal/ledger"
	"inmemoryaccountledgercore/internal/scenario"
)

func TestInputsPreserveEveryTaskField(t *testing.T) {
	primary, instalment := configuredAccounts(t)
	// This table is transcribed from TASK, independently of scenario.Inputs.
	want := []domain.InputEvent{
		{ID: "E1", Kind: "CREDIT", AccountID: "ACC-001", Currency: primary.Currency, RawAmount: "1200.00", BookingDay: 1, ValueDay: 1, Order: 1},
		{ID: "E2", Kind: "DEBIT", AccountID: "ACC-001", Currency: primary.Currency, RawAmount: "950.00", BookingDay: 1, ValueDay: 1, Order: 2},
		{ID: "E3", Kind: "AUTHORIZATION", AccountID: "ACC-001", Currency: primary.Currency, RawAmount: "200.00", BookingDay: 2, ValueDay: 2, Order: 3, AuthorizationID: "Auth-A"},
		{ID: "E4", Kind: "CREDIT", AccountID: "ACC-001", Currency: primary.Currency, RawAmount: "400.00", BookingDay: 3, ValueDay: 3, Order: 4},
		{ID: "E5", Kind: "SETTLEMENT", AccountID: "ACC-001", Currency: primary.Currency, RawAmount: "185.00", BookingDay: 4, ValueDay: 4, Order: 5, AuthorizationID: "Auth-A"},
		{ID: "E6", Kind: "SETTLEMENT", AccountID: "ACC-001", Currency: primary.Currency, RawAmount: "180.00", BookingDay: 4, ValueDay: 4, Order: 6, AuthorizationID: "Auth-Z"},
		{ID: "E7", Kind: "DEBIT", AccountID: "ACC-001", Currency: primary.Currency, RawAmount: "620.00", BookingDay: 5, ValueDay: 2, Order: 7},
		{ID: "E8", Kind: "AUTHORIZATION", AccountID: "ACC-001", Currency: primary.Currency, RawAmount: "90.00", BookingDay: 5, ValueDay: 5, Order: 8, AuthorizationID: "Auth-B"},
		{ID: "E9", Kind: "REVERSAL", AccountID: "ACC-001", Currency: primary.Currency, BookingDay: 6, ValueDay: 2, Order: 9, ReversesEventID: "E7"},
		{ID: "E10", Kind: "CREDIT", AccountID: "ACC-002", Currency: instalment.Currency, RawAmount: "10.000", BookingDay: 5, ValueDay: 5, Order: 10, InstalmentCount: 3},
	}
	assert.Equal(t, want, scenario.Inputs(primary, instalment))
	modified := scenario.Inputs(primary, instalment)
	modified[0].RawAmount = "0.00"
	assert.Equal(t, want, scenario.Inputs(primary, instalment), "each stream is independently owned")
}

func TestInputsUseInjectedIdentitiesAndDescriptors(t *testing.T) {
	primary, instalment := configuredAccounts(t)
	primary.ID, instalment.ID = "primary", "instalment"
	var err error
	primary.Currency, err = domain.NewCurrency("XYZ", 4)
	require.NoError(t, err)
	instalment.Currency, err = domain.NewCurrency("ABC", 18)
	require.NoError(t, err)
	inputs := scenario.Inputs(primary, instalment)
	require.Len(t, inputs, 10)
	for _, input := range inputs[:9] {
		assert.Equal(t, primary.ID, input.AccountID)
		assert.Equal(t, primary.Currency, input.Currency)
	}
	assert.Equal(t, instalment.ID, inputs[9].AccountID)
	assert.Equal(t, instalment.Currency, inputs[9].Currency)
}

func TestRunStopsOnTechnicalFailure(t *testing.T) {
	failure := errors.New("injected append failure")
	for _, tc := range []struct {
		name          string
		matches       func(domain.Record) bool
		lastEvent     domain.EventID
		capitalizedID domain.AccountID
	}{
		{"submission", func(r domain.Record) bool { return r.Attempt != nil && r.Attempt.Input.ID == "E1" }, "", ""},
		{"primary close", func(r domain.Record) bool {
			return r.Close != nil && r.Close.Day == 1 && r.Close.AccountID == "ACC-001"
		}, "E2", ""},
		{"instalment close", func(r domain.Record) bool {
			return r.Close != nil && r.Close.Day == 1 && r.Close.AccountID == "ACC-002"
		}, "E2", ""},
		{"historical correction", func(r domain.Record) bool { return r.Entry != nil && r.Entry.Kind == domain.EntryFeeAssessment }, "E7", ""},
		{"primary finalization", func(r domain.Record) bool {
			return r.Entry != nil && r.Entry.Kind == domain.EntryCapitalization && r.Entry.AccountID == "ACC-001"
		}, "E10", ""},
		{"instalment finalization", func(r domain.Record) bool {
			return r.Entry != nil && r.Entry.Kind == domain.EntryCapitalization && r.Entry.AccountID == "ACC-002"
		}, "E10", "ACC-001"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			primary, instalment := configuredAccounts(t)
			repository := &testRepository{}
			fee, err := domain.ParseMoney(primary.Currency, "25.00")
			require.NoError(t, err)
			core, err := ledger.New(repository, []domain.Account{primary, instalment}, map[domain.Currency]domain.Money{
				primary.Currency: fee, instalment.Currency: instalment.OpeningBalance,
			})
			require.NoError(t, err)
			var beforeFailure domain.History
			repository.reject = func(batch domain.Batch) error {
				for _, record := range batch.Records {
					if tc.matches(record) {
						beforeFailure = repository.History()
						return failure
					}
				}
				return nil
			}
			require.ErrorIs(t, scenario.Run(core, primary, instalment), failure)
			assert.Equal(t, beforeFailure, repository.History(), "no later operation follows the failed batch")
			var lastEvent domain.EventID
			var capitalizedID domain.AccountID
			for _, record := range repository.History().Records {
				if record.Attempt != nil {
					lastEvent = record.Attempt.Input.ID
				}
				if record.Entry != nil && record.Entry.Kind == domain.EntryCapitalization {
					capitalizedID = record.Entry.AccountID
				}
			}
			assert.Equal(t, tc.lastEvent, lastEvent)
			assert.Equal(t, tc.capitalizedID, capitalizedID)
		})
	}
}

func configuredAccounts(t *testing.T) (domain.Account, domain.Account) {
	t.Helper()
	aed, err := domain.NewCurrency("AED", 2)
	require.NoError(t, err)
	bhd, err := domain.NewCurrency("BHD", 3)
	require.NoError(t, err)
	aedZero, err := domain.NewMoney(aed, 0)
	require.NoError(t, err)
	bhdZero, err := domain.NewMoney(bhd, 0)
	require.NoError(t, err)
	return domain.Account{ID: "ACC-001", Currency: aed, OpeningBalance: aedZero},
		domain.Account{ID: "ACC-002", Currency: bhd, OpeningBalance: bhdZero}
}

// The unit-test double only stores detached batches and injects append failures;
// repository contracts are tested beside memory, and real wiring in tests/replay.
type testRepository struct {
	records []domain.Record
	reject  func(domain.Batch) error
}

func (r *testRepository) History() domain.History {
	history := domain.History{Records: make([]domain.Record, len(r.records))}
	for i, record := range r.records {
		history.Records[i] = record.Clone()
	}
	return history
}

func (r *testRepository) Append(batch domain.Batch) error {
	if r.reject != nil {
		if err := r.reject(batch); err != nil {
			return err
		}
	}
	for _, record := range batch.Records {
		r.records = append(r.records, record.Clone())
	}
	return nil
}
