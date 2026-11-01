// Package report formats final restated finances alongside recorded decisions.
// It has no repository or workflow dependencies and never changes history.
package report

import (
	"fmt"
	"strings"

	"inmemoryaccountledgercore/internal/domain"
	"inmemoryaccountledgercore/internal/projection"
)

// Format returns one table in supplied day/account order. History and state must
// describe the same finalized snapshot. Financial values use all value-dated
// postings; authorization transitions use booking days and diagnostics use their
// original observation days. Presentation never reevaluates a recorded decision.
// Errors return no partial report.
func Format(history domain.History, state *projection.Projection, days []domain.Day, accounts []domain.AccountID) (string, error) {
	if state == nil {
		return "", fmt.Errorf("report requires a history projection")
	}
	var out strings.Builder
	out.WriteString("Restated after the entire event stream E1–E10 and Day 6 finalization\n")
	out.WriteString("Finances are restated; authorization states and reasons are recorded decisions.\n\n")
	out.WriteString("Day | Account | Currency | Closing ledger | Daily interest | Fee history (assessed/value day) | Recorded authorizations | Errors/decision reasons\n")
	for _, day := range days {
		for _, account := range accounts {
			balance, err := state.Balance(account, day, projection.BalanceOptions{})
			if err != nil {
				return "", err
			}
			closing, err := balance.Format()
			if err != nil {
				return "", err
			}
			accrual, exists := state.Accrual(account, day)
			if !exists {
				return "", fmt.Errorf("report requires closed account %s day %d", account, day)
			}
			interest, err := accrual.Format()
			if err != nil {
				return "", err
			}
			fees, err := feeHistory(history, account, day)
			if err != nil {
				return "", err
			}
			authorizations, err := recordedAuthorizations(history, account, day)
			if err != nil {
				return "", err
			}
			fmt.Fprintf(&out, "%d | %s | %s | %s | %s | %s | %s | %s\n",
				day, account, balance.Currency().Code(), closing, interest, fees, authorizations,
				decisionReasons(history, account, day))
		}
	}
	return out.String(), nil
}

func feeHistory(history domain.History, account domain.AccountID, day domain.Day) (string, error) {
	var values []string
	for _, record := range history.Records {
		entry := record.Entry
		if entry == nil || entry.AccountID != account || entry.ValueDay != day {
			continue
		}
		switch entry.Kind {
		case domain.EntryFeeAssessment, domain.EntryFeeRefund, domain.EntryFeeAdjustment:
			amount, err := entry.Amount.Format()
			if err != nil {
				return "", err
			}
			if entry.Amount.MinorUnits() > 0 {
				amount = "+" + amount
			}
			value := fmt.Sprintf("%s %s [key=%s; obligation=%s; booked=%d; value=%d",
				entry.Kind, amount, record.Key, entry.FeeObligationID, entry.BookingDay, entry.ValueDay)
			if entry.OriginalEntryKey != "" {
				value += fmt.Sprintf("; original=%s", entry.OriginalEntryKey)
			}
			value += fmt.Sprintf("; cause=%s]", entry.CauseKey)
			values = append(values, value)
		}
	}
	return cell(values), nil
}

func recordedAuthorizations(history domain.History, account domain.AccountID, day domain.Day) (string, error) {
	var order []domain.AuthorizationID
	latest := make(map[domain.AuthorizationID]domain.AuthorizationTransition)
	for _, record := range history.Records {
		transition := record.Authorization
		if transition == nil || transition.AccountID != account || transition.BookingDay > day {
			continue
		}
		if _, exists := latest[transition.AuthorizationID]; !exists {
			order = append(order, transition.AuthorizationID)
		}
		latest[transition.AuthorizationID] = *transition
	}
	var values []string
	for _, id := range order {
		transition := latest[id]
		value := fmt.Sprintf("%s %s", id, transition.State)
		if transition.State == domain.AuthorizationApproved {
			hold, err := transition.Hold.Format()
			if err != nil {
				return "", err
			}
			value = fmt.Sprintf("%s ACTIVE (hold %s)", id, hold)
		}
		values = append(values, value)
	}
	return cell(values), nil
}

func decisionReasons(history domain.History, account domain.AccountID, day domain.Day) string {
	inputs := make(map[domain.AttemptID]domain.EventID)
	outcomes := make(map[domain.AttemptID]domain.OutcomeStatus)
	for _, record := range history.Records {
		if record.Attempt != nil {
			inputs[record.Attempt.ID] = record.Attempt.Input.ID
		}
		if record.Outcome != nil {
			outcomes[record.Outcome.AttemptID] = record.Outcome.Status
		}
	}
	var values []string
	for _, record := range history.Records {
		diagnostic := record.Diagnostic
		if diagnostic == nil || diagnostic.AccountID != account || diagnostic.ObservationDay != day {
			continue
		}
		label := string(record.Key)
		if event, exists := inputs[diagnostic.AttemptID]; exists {
			label = fmt.Sprintf("%s %s", event, outcomes[diagnostic.AttemptID])
		}
		values = append(values, fmt.Sprintf("%s [%s]: %s", label, diagnostic.Code, diagnostic.Reason))
	}
	return cell(values)
}

func cell(values []string) string {
	if len(values) == 0 {
		return "-"
	}
	return strings.Join(values, "; ")
}
