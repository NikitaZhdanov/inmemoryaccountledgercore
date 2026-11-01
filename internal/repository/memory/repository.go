// Package memory provides sequential, append-only in-memory record storage.
package memory

import (
	"errors"
	"fmt"

	"inmemoryaccountledgercore/internal/domain"
)

var (
	ErrInvalidKey    = errors.New("empty record key")
	ErrDuplicateKey  = errors.New("duplicate record key")
	ErrInvalidRecord = errors.New("record must contain exactly one payload")
)

// Repository stores records in append order. Its zero value is ready for use.
// It validates record structure and keys, not business policy or references.
// It does not support concurrent use.
type Repository struct {
	records []domain.Record
	keys    map[domain.RecordKey]struct{}
}

func New() *Repository { return &Repository{} }

// History returns detached containers and payloads, preserving append order.
func (r *Repository) History() domain.History {
	return domain.History{Records: cloneRecords(r.records)}
}

// Append validates all proposed keys and envelopes before changing storage.
// Empty batches are no-ops. Distinct keys may carry the same supplied event ID.
func (r *Repository) Append(batch domain.Batch) error {
	proposed := make(map[domain.RecordKey]struct{}, len(batch.Records))
	for i, record := range batch.Records {
		if record.Key == "" {
			return fmt.Errorf("record %d: %w", i, ErrInvalidKey)
		}
		if _, exists := r.keys[record.Key]; exists {
			return fmt.Errorf("record %q: %w", record.Key, ErrDuplicateKey)
		}
		if _, exists := proposed[record.Key]; exists {
			return fmt.Errorf("record %q: %w", record.Key, ErrDuplicateKey)
		}
		if !hasOnePayload(record) {
			return fmt.Errorf("record %q: %w", record.Key, ErrInvalidRecord)
		}
		proposed[record.Key] = struct{}{}
	}

	copies := cloneRecords(batch.Records)
	if r.keys == nil && len(copies) != 0 {
		r.keys = make(map[domain.RecordKey]struct{}, len(copies))
	}
	r.records = append(r.records, copies...)
	for key := range proposed {
		r.keys[key] = struct{}{}
	}
	return nil
}

func cloneRecords(records []domain.Record) []domain.Record {
	if records == nil {
		return nil
	}
	copies := make([]domain.Record, len(records))
	for i, record := range records {
		copies[i] = record.Clone()
	}
	return copies
}

func hasOnePayload(record domain.Record) bool {
	count := 0
	for _, present := range [...]bool{
		record.Account != nil, record.Attempt != nil, record.Outcome != nil,
		record.Entry != nil, record.Authorization != nil,
		record.Diagnostic != nil, record.Close != nil,
	} {
		if present {
			count++
		}
	}
	return count == 1
}
