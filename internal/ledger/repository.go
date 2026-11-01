package ledger

import "inmemoryaccountledgercore/internal/domain"

// Repository stores authoritative append-only history. History returns an
// ordered, detached snapshot. Append copies its input and commits the entire
// batch or none of it, rejecting empty or conflicting record keys and malformed
// record envelopes. Supplied event IDs are not storage keys: input duplication
// is a business decision, recorded under a fresh attempt identity.
// Implementations need not support concurrent use.
type Repository interface {
	History() domain.History
	Append(domain.Batch) error
}
