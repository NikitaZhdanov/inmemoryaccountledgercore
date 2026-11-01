package ledger

import (
	"errors"
	"fmt"
	"sort"

	"inmemoryaccountledgercore/internal/domain"
)

var (
	ErrInvalidObservation = errors.New("record prefix outside history")
	ErrUnknownAccount     = errors.New("account absent from observation")
)

// BalanceOptions selects exclusions from monetary entries, never from opening
// money. The zero value includes every posting. Day zero disables a day filter.
// Fee exclusions cover assessments, refunds, and adjustments together.
type BalanceOptions struct {
	ExcludeAllFees           bool
	ExcludeFeeDay            domain.Day
	ExcludeCapitalizationDay domain.Day
}

// RecordedAuthorization is the last recorded transition at this observation.
// It preserves its record key and decision; rebuilding never reapproves a hold.
type RecordedAuthorization struct {
	Key        domain.RecordKey
	Transition domain.AuthorizationTransition
}

type DailyAccrual struct {
	Day    domain.Day
	Amount domain.Money
}

type accountDay struct {
	account domain.AccountID
	day     domain.Day
}

type authorizationKey struct {
	account domain.AccountID
	id      domain.AuthorizationID
}

// Projection is disposable, read-only state over an owned history prefix.
// Business fields/references are supplied by ledger processing, not validated
// here as new input decisions. Monetary arithmetic remains checked.
type Projection struct {
	history            domain.History
	accounts           map[domain.AccountID]domain.Account
	authorizations     map[authorizationKey]RecordedAuthorization
	authorizationOrder []authorizationKey
	holds              map[domain.AccountID]domain.Money
	fees               map[accountDay]domain.Money
	closed             map[accountDay]bool
	accruals           map[accountDay]domain.Money
	capitalizedThrough map[domain.AccountID]domain.Day
}

// Rebuild observes exactly the first recordCount records in append order;
// len(history.Records) observes all history, and zero observes no records.
// This is a record checkpoint, not an input order or booking-day cutoff: late
// postings can have earlier booking/value dates. No history is changed or appended.
// On any derived failure it returns no partial projection.
func Rebuild(history domain.History, recordCount int) (*Projection, error) {
	if recordCount < 0 || recordCount > len(history.Records) {
		return nil, ErrInvalidObservation
	}

	p, accountOrder, err := initProjection(history, recordCount)
	if err != nil {
		return nil, err
	}
	if err := p.processRecords(); err != nil {
		return nil, err
	}
	if err := p.applyHolds(); err != nil {
		return nil, err
	}
	if err := p.computeAccruals(accountOrder); err != nil {
		return nil, err
	}
	return p, nil
}

func initProjection(history domain.History, recordCount int) (*Projection, []domain.AccountID, error) {
	p := &Projection{
		history:            domain.History{Records: make([]domain.Record, recordCount)},
		accounts:           make(map[domain.AccountID]domain.Account),
		authorizations:     make(map[authorizationKey]RecordedAuthorization),
		holds:              make(map[domain.AccountID]domain.Money),
		fees:               make(map[accountDay]domain.Money),
		closed:             make(map[accountDay]bool),
		accruals:           make(map[accountDay]domain.Money),
		capitalizedThrough: make(map[domain.AccountID]domain.Day),
	}
	var accountOrder []domain.AccountID
	for i, record := range history.Records[:recordCount] {
		p.history.Records[i] = record.Clone()
		if record.Account != nil {
			account := *record.Account
			zero, err := domain.NewMoney(account.Currency, 0)
			if err != nil {
				return nil, nil, err
			}
			if _, err := zero.Add(account.OpeningBalance); err != nil {
				return nil, nil, err
			}
			if _, exists := p.accounts[account.ID]; !exists {
				accountOrder = append(accountOrder, account.ID)
			}
			p.accounts[account.ID] = account
			p.holds[account.ID] = zero
		}
	}
	return p, accountOrder, nil
}

func (p *Projection) processRecords() error {
	for _, record := range p.history.Records {
		if transition := record.Authorization; transition != nil {
			if _, err := p.zero(transition.AccountID); err != nil {
				return err
			}
			key := authorizationKey{transition.AccountID, transition.AuthorizationID}
			if _, exists := p.authorizations[key]; !exists {
				p.authorizationOrder = append(p.authorizationOrder, key)
			}
			p.authorizations[key] = RecordedAuthorization{record.Key, *transition}
		}
		if entry := record.Entry; entry != nil {
			if _, err := p.zero(entry.AccountID); err != nil {
				return err
			}
			if isFee(entry.Kind) {
				key := accountDay{entry.AccountID, entry.ValueDay}
				net, exists := p.fees[key]
				if !exists {
					net, _ = p.zero(entry.AccountID)
				}
				var err error
				p.fees[key], err = net.Add(entry.Amount)
				if err != nil {
					return fmt.Errorf("fee record %q: %w", record.Key, err)
				}
			}
			if entry.Kind == domain.EntryCapitalization && entry.ValueDay > p.capitalizedThrough[entry.AccountID] {
				p.capitalizedThrough[entry.AccountID] = entry.ValueDay
			}
		}
		if close := record.Close; close != nil {
			if _, err := p.zero(close.AccountID); err != nil {
				return err
			}
			p.closed[accountDay{close.AccountID, close.Day}] = true
		}
	}
	return nil
}

func (p *Projection) applyHolds() error {
	for _, key := range p.authorizationOrder {
		transition := p.authorizations[key].Transition
		if transition.State == domain.AuthorizationApproved {
			total, err := p.holds[key.account].Add(transition.Hold)
			if err != nil {
				return fmt.Errorf("active hold %q: %w", key.id, err)
			}
			p.holds[key.account] = total
		}
	}
	return nil
}

func (p *Projection) computeAccruals(accountOrder []domain.AccountID) error {
	for _, id := range accountOrder {
		days, _ := p.ClosedDays(id)
		for _, day := range days {
			// Capitalization pays the window's accruals; it does not earn
			// interest on its own value day. Earlier capitalizations can earn
			// interest on later days.
			base, err := p.Balance(id, day, BalanceOptions{ExcludeCapitalizationDay: day})
			if err != nil {
				return err
			}
			accrual, err := DailyInterest(base)
			if err != nil {
				return err
			}
			p.accruals[accountDay{id, day}] = accrual
		}
	}
	return nil
}

// Balance starts with opening money and adds selected postings whose value day
// is on/before day, in original append order. Booking day does not filter them.
// Every intermediate must fit; an error returns no partial monetary result.
func (p *Projection) Balance(id domain.AccountID, day domain.Day, options BalanceOptions) (domain.Money, error) {
	if _, err := p.zero(id); err != nil {
		return domain.Money{}, err
	}
	total := p.accounts[id].OpeningBalance
	for _, record := range p.history.Records {
		entry := record.Entry
		if entry == nil || entry.AccountID != id || entry.ValueDay > day {
			continue
		}
		if isFee(entry.Kind) && (options.ExcludeAllFees || (options.ExcludeFeeDay != 0 && entry.ValueDay == options.ExcludeFeeDay)) {
			continue
		}
		if entry.Kind == domain.EntryCapitalization && options.ExcludeCapitalizationDay != 0 && entry.ValueDay == options.ExcludeCapitalizationDay {
			continue
		}
		var err error
		total, err = total.Add(entry.Amount)
		if err != nil {
			return domain.Money{}, fmt.Errorf("balance record %q: %w", record.Key, err)
		}
	}
	return total, nil
}

// AvailableBalance uses the observation's current active holds and the requested
// value-date balance. Historical authorization decisions require an earlier
// record prefix, not reevaluation against a restated monetary balance.
func (p *Projection) AvailableBalance(id domain.AccountID, day domain.Day) (domain.Money, error) {
	balance, err := p.Balance(id, day, BalanceOptions{})
	if err != nil {
		return domain.Money{}, err
	}
	return balance.Sub(p.holds[id])
}

func (p *Projection) ActiveHolds(id domain.AccountID) (domain.Money, error) {
	if _, exists := p.accounts[id]; !exists {
		return domain.Money{}, ErrUnknownAccount
	}
	return p.holds[id], nil
}

func (p *Projection) Authorization(id domain.AccountID, auth domain.AuthorizationID) (RecordedAuthorization, bool) {
	value, exists := p.authorizations[authorizationKey{id, auth}]
	return value, exists
}

// FeeNet is the signed sum of all fee effects for an account/assessed value day:
// charges are negative and refunds positive. A missing obligation returns zero.
func (p *Projection) FeeNet(id domain.AccountID, day domain.Day) (domain.Money, error) {
	account, exists := p.accounts[id]
	if !exists {
		return domain.Money{}, ErrUnknownAccount
	}
	if net, exists := p.fees[accountDay{id, day}]; exists {
		return net, nil
	}
	return domain.NewMoney(account.Currency, 0)
}

// ClosedDays returns a fresh ascending list of the recorded close checkpoints.
func (p *Projection) ClosedDays(id domain.AccountID) ([]domain.Day, error) {
	if _, exists := p.accounts[id]; !exists {
		return nil, ErrUnknownAccount
	}
	var days []domain.Day
	for key := range p.closed {
		if key.account == id {
			days = append(days, key.day)
		}
	}
	sort.Slice(days, func(i, j int) bool { return days[i] < days[j] })
	return days, nil
}

// Accrual returns the corrected rounded accrual for a closed day, including
// days already paid by capitalization. An open/absent day returns false.
func (p *Projection) Accrual(id domain.AccountID, day domain.Day) (domain.Money, bool) {
	value, exists := p.accruals[accountDay{id, day}]
	return value, exists
}

// UnpaidAccruals returns a fresh chronological list. In the single replay window,
// a capitalization pays all closed-day accruals through its value day. The daily
// values remain queryable via Accrual and never change balances or holds.
func (p *Projection) UnpaidAccruals(id domain.AccountID) ([]DailyAccrual, error) {
	days, err := p.ClosedDays(id)
	if err != nil {
		return nil, err
	}
	var accruals []DailyAccrual
	for _, day := range days {
		if day > p.capitalizedThrough[id] {
			accruals = append(accruals, DailyAccrual{day, p.accruals[accountDay{id, day}]})
		}
	}
	return accruals, nil
}

func (p *Projection) zero(id domain.AccountID) (domain.Money, error) {
	account, exists := p.accounts[id]
	if !exists {
		return domain.Money{}, fmt.Errorf("%w: %q", ErrUnknownAccount, id)
	}
	return domain.NewMoney(account.Currency, 0)
}

func isFee(kind domain.EntryKind) bool {
	return kind == domain.EntryFeeAssessment || kind == domain.EntryFeeRefund || kind == domain.EntryFeeAdjustment
}
