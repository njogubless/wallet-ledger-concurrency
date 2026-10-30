// Package ledger implements a small in-memory double-entry ledger of
// account balances, expressed in integer minor units (e.g. cents).
package ledger

import "errors"

var (
	ErrUnknownAccount    = errors.New("unknown account")
	ErrInvalidAmount     = errors.New("amount must be positive")
	ErrInsufficientFunds = errors.New("insufficient funds")
	ErrSameAccount       = errors.New("cannot transfer to the same account")
)

// Ledger holds a fixed set of accounts, keyed by ID, each with an integer
// balance in minor units. A balance must never go negative.
type Ledger struct {
	balances map[string]int64
}

// New creates a Ledger seeded with the given opening balances.
func New(opening map[string]int64) *Ledger {
	b := make(map[string]int64, len(opening))
	for id, v := range opening {
		b[id] = v
	}
	return &Ledger{balances: b}
}

// Balance returns the current balance of an account.
func (l *Ledger) Balance(id string) (int64, error) {
	v, ok := l.balances[id]
	if !ok {
		return 0, ErrUnknownAccount
	}
	return v, nil
}

// Total returns the sum of every account balance. No sequence of calls to
// Transfer, successful or failed, concurrent or not, may change this value.
func (l *Ledger) Total() int64 {
	var sum int64
	for _, v := range l.balances {
		sum += v
	}
	return sum
}

// Transfer moves amount from the "from" account to the "to" account.
//
// Required behavior, including when Transfer is called concurrently from
// many goroutines against overlapping accounts and in mixed directions
// (e.g. "A->B" and "B->A" at the same time):
//   - an account's balance must never go negative;
//   - Total() must never change, whatever mix of successful and failed
//     transfers occurs;
//   - Transfer must never deadlock, for any number of concurrent callers and
//     any set of account pairs.
//
// THIS IMPLEMENTATION IS NOT SAFE FOR CONCURRENT USE: it reads the sender's
// balance, decides whether there are sufficient funds, and then mutates both
// balances as separate, unsynchronized steps. Two concurrent transfers can
// both read the same stale balance, both pass the check, and both debit the
// account, driving it negative (and corrupting Total() along the way).
func (l *Ledger) Transfer(from, to string, amount int64) error {
	if amount <= 0 {
		return ErrInvalidAmount
	}
	if from == to {
		return ErrSameAccount
	}
	fromBal, ok := l.balances[from]
	if !ok {
		return ErrUnknownAccount
	}
	if _, ok := l.balances[to]; !ok {
		return ErrUnknownAccount
	}
	if fromBal < amount {
		return ErrInsufficientFunds
	}
	// BUG: no synchronization between the check above and the writes below,
	// and no synchronization between concurrent calls at all.
	l.balances[from] = l.balances[from] - amount
	l.balances[to] = l.balances[to] + amount
	return nil
}
