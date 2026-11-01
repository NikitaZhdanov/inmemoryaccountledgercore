package projection

import (
	"sort"

	"inmemoryaccountledgercore/internal/domain"
)

// ValidateBatch checks the proposed history before append, including every affected value
// checkpoint. A backdated posting may fit at its own day but overflow at a later
// posting. Rebuild also checks derived closed-day accruals and recorded holds.
func ValidateBatch(history domain.History, batch domain.Batch, id domain.AccountID, fromDay domain.Day) error {
	records := make([]domain.Record, len(history.Records), len(history.Records)+len(batch.Records))
	copy(records, history.Records)
	history.Records = append(records, batch.Records...)
	projection, err := Rebuild(history, len(history.Records))
	if err != nil {
		return err
	}
	days := []domain.Day{fromDay}
	seen := map[domain.Day]bool{fromDay: true}
	for _, record := range history.Records {
		entry := record.Entry
		if entry != nil && entry.AccountID == id && entry.ValueDay >= fromDay && !seen[entry.ValueDay] {
			days = append(days, entry.ValueDay)
			seen[entry.ValueDay] = true
		}
	}
	sort.Slice(days, func(i, j int) bool { return days[i] < days[j] })
	for _, day := range days {
		if _, err := projection.AvailableBalance(id, day); err != nil {
			return err
		}
	}
	return nil
}
