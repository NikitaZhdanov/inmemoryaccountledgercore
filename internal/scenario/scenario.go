// Package scenario supplies the canonical TASK inputs and explicit close schedule.
// Account identities and currency descriptors come from application composition.
package scenario

import (
	"fmt"

	"inmemoryaccountledgercore/internal/domain"
	"inmemoryaccountledgercore/internal/ledger"
)

// Inputs returns a fresh E1–E10 stream for the configured primary and instalment
// accounts. Amount text, event references, dates, and order preserve TASK.
// It supplies no account, currency, opening-balance, or fee-policy defaults.
func Inputs(primary, instalment domain.Account) []domain.InputEvent {
	return []domain.InputEvent{
		{ID: "E1", Kind: domain.InputCredit, AccountID: primary.ID, Currency: primary.Currency, RawAmount: "1200.00", BookingDay: 1, ValueDay: 1, Order: 1},
		{ID: "E2", Kind: domain.InputDebit, AccountID: primary.ID, Currency: primary.Currency, RawAmount: "950.00", BookingDay: 1, ValueDay: 1, Order: 2},
		{ID: "E3", Kind: domain.InputAuthorization, AccountID: primary.ID, Currency: primary.Currency, RawAmount: "200.00", BookingDay: 2, ValueDay: 2, Order: 3, AuthorizationID: "Auth-A"},
		{ID: "E4", Kind: domain.InputCredit, AccountID: primary.ID, Currency: primary.Currency, RawAmount: "400.00", BookingDay: 3, ValueDay: 3, Order: 4},
		{ID: "E5", Kind: domain.InputSettlement, AccountID: primary.ID, Currency: primary.Currency, RawAmount: "185.00", BookingDay: 4, ValueDay: 4, Order: 5, AuthorizationID: "Auth-A"},
		{ID: "E6", Kind: domain.InputSettlement, AccountID: primary.ID, Currency: primary.Currency, RawAmount: "180.00", BookingDay: 4, ValueDay: 4, Order: 6, AuthorizationID: "Auth-Z"},
		{ID: "E7", Kind: domain.InputDebit, AccountID: primary.ID, Currency: primary.Currency, RawAmount: "620.00", BookingDay: 5, ValueDay: 2, Order: 7},
		{ID: "E8", Kind: domain.InputAuthorization, AccountID: primary.ID, Currency: primary.Currency, RawAmount: "90.00", BookingDay: 5, ValueDay: 5, Order: 8, AuthorizationID: "Auth-B"},
		{ID: "E9", Kind: domain.InputReversal, AccountID: primary.ID, Currency: primary.Currency, BookingDay: 6, ValueDay: 2, Order: 9, ReversesEventID: "E7"},
		{ID: "E10", Kind: domain.InputCredit, AccountID: instalment.ID, Currency: instalment.Currency, RawAmount: "10.000", BookingDay: 5, ValueDay: 5, Order: 10, InstalmentCount: 3},
	}
}

// Run submits the canonical stream to an already initialized core. Business
// rejections remain recorded and replay continues; a technical failure stops
// immediately, preserving earlier committed batches. Close checkpoints follow
// SPEC explicitly, independent of booking-date changes. Day 6 finalization runs
// after E10 and the core's automatic historical corrections. Repeating Run on a
// completed core records duplicate rejections; it is not a fresh replay.
func Run(core *ledger.Core, primary, instalment domain.Account) error {
	accounts := []domain.Account{primary, instalment}
	for _, input := range Inputs(primary, instalment) {
		if _, err := core.Submit(input); err != nil {
			return fmt.Errorf("scenario submit %s: %w", input.ID, err)
		}
		var day domain.Day
		switch input.ID {
		case "E2":
			day = 1
		case "E3":
			day = 2
		case "E4":
			day = 3
		case "E6":
			day = 4
		case "E8":
			day = 5
		default:
			continue
		}
		for _, account := range accounts {
			cause := domain.RecordKey(fmt.Sprintf("scenario/close/%d", day))
			if err := core.CloseDay(account.ID, day, day, cause); err != nil {
				return fmt.Errorf("scenario close %s day %d: %w", account.ID, day, err)
			}
		}
	}
	for _, account := range accounts {
		if err := core.FinalizeDay(account.ID, 6, 6, "scenario/window-end"); err != nil {
			return fmt.Errorf("scenario finalize %s day 6: %w", account.ID, err)
		}
	}
	return nil
}
