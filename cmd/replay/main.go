// Command replay runs the supplied ledger stream in fresh in-memory storage.
package main

import (
	"fmt"
	"io"
	"os"

	"inmemoryaccountledgercore/internal/domain"
	"inmemoryaccountledgercore/internal/ledger"
	"inmemoryaccountledgercore/internal/projection"
	"inmemoryaccountledgercore/internal/report"
	"inmemoryaccountledgercore/internal/repository/memory"
	"inmemoryaccountledgercore/internal/scenario"
)

func main() {
	if err := run(os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

type configuration struct {
	primary, instalment domain.Account
	charges             map[domain.Currency]domain.Money
}

func canonicalConfiguration() (configuration, error) {
	var accounts []domain.Account
	charges := make(map[domain.Currency]domain.Money)
	for _, spec := range []struct {
		id     domain.AccountID
		code   string
		scale  int
		charge string
	}{
		{"ACC-001", "AED", 2, "25.00"},
		{"ACC-002", "BHD", 3, "0.000"},
	} {
		currency, err := domain.NewCurrency(spec.code, spec.scale)
		if err != nil {
			return configuration{}, err
		}
		opening, err := domain.NewMoney(currency, 0)
		if err != nil {
			return configuration{}, err
		}
		charge, err := domain.ParseMoney(currency, spec.charge)
		if err != nil {
			return configuration{}, err
		}
		accounts = append(accounts, domain.Account{ID: spec.id, Currency: currency, OpeningBalance: opening})
		charges[currency] = charge
	}
	return configuration{primary: accounts[0], instalment: accounts[1], charges: charges}, nil
}

func run(out io.Writer) error {
	config, err := canonicalConfiguration()
	if err != nil {
		return err
	}
	return replay(out, config)
}

func replay(out io.Writer, config configuration) error {
	repository := memory.New()
	accounts := []domain.Account{config.primary, config.instalment}
	core, err := ledger.New(repository, accounts, config.charges)
	if err != nil {
		return fmt.Errorf("initialize replay: %w", err)
	}
	if err := scenario.Run(core, config.primary, config.instalment); err != nil {
		return err
	}
	history := repository.History()
	state, err := projection.Rebuild(history, len(history.Records))
	if err != nil {
		return err
	}
	table, err := report.Format(history, state, []domain.Day{1, 2, 3, 4, 5, 6},
		[]domain.AccountID{config.primary.ID, config.instalment.ID})
	if err != nil {
		return err
	}
	_, err = io.WriteString(out, table)
	return err
}
