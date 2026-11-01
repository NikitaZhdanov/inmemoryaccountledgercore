package ledger

import (
	"errors"
	"fmt"

	"inmemoryaccountledgercore/internal/domain"
)

func (c *Core) configure(accounts []domain.Account, charges map[domain.Currency]domain.Money) error {
	if err := validateCurrencies(accounts, charges); err != nil {
		return err
	}
	if err := c.configureAccounts(accounts, charges); err != nil {
		return err
	}
	return c.configureCharges(charges)
}

func (c *Core) configureAccounts(accounts []domain.Account, charges map[domain.Currency]domain.Money) error {
	for _, account := range accounts {
		if account.ID == "" {
			return errors.New("empty account ID")
		}
		if _, exists := c.accounts[account.ID]; exists {
			return fmt.Errorf("duplicate account ID %q", account.ID)
		}
		if err := account.OpeningBalance.ValidateCurrency(account.Currency); err != nil {
			return fmt.Errorf("account %q opening balance: %w", account.ID, err)
		}
		if _, exists := charges[account.Currency]; !exists {
			return fmt.Errorf("missing overdraft policy for %s", account.Currency.Code())
		}
		c.accounts[account.ID] = account
	}
	return nil
}

func (c *Core) configureCharges(charges map[domain.Currency]domain.Money) error {
	for currency, charge := range charges {
		if err := charge.ValidateCurrency(currency); err != nil {
			return fmt.Errorf("overdraft policy for %s: %w", currency.Code(), err)
		}
		if charge.MinorUnits() < 0 {
			return fmt.Errorf("negative overdraft charge for %s", currency.Code())
		}
		c.charges[currency] = charge
	}
	return nil
}

func validateCurrencies(accounts []domain.Account, charges map[domain.Currency]domain.Money) error {
	currencies := make(map[domain.Currency]struct{})
	for _, account := range accounts {
		currencies[account.Currency] = struct{}{}
	}
	for currency := range charges {
		currencies[currency] = struct{}{}
	}
	scales := make(map[string]int)
	for currency := range currencies {
		scale, err := currency.Scale()
		if err != nil {
			return err
		}
		if previous, exists := scales[currency.Code()]; exists && previous != scale {
			return fmt.Errorf("%w: conflicting scales for %s", domain.ErrCurrencyMismatch, currency.Code())
		}
		scales[currency.Code()] = scale
	}
	return nil
}
