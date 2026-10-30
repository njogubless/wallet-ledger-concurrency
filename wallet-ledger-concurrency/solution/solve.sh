#!/bin/bash
# Reference solution: guard the ledger with a single mutex held across the
# whole read-check-mutate sequence of Transfer. This is the simplest
# correct fix — it trivially cannot deadlock (only one lock is ever held at
# a time) and it closes the race between the balance check and the writes.
set -euo pipefail

cat > /app/ledger/ledger.go <<'EOF'
// Package ledger implements a small in-memory double-entry ledger of
// account balances, expressed in integer minor units (e.g. cents).
package ledger

import (
	"errors"
	"sync"
)

var (
	ErrUnknownAccount    = errors.New("unknown account")
	ErrInvalidAmount     = errors.New("amount must be positive")
	ErrInsufficientFunds = errors.New("insufficient funds")
	ErrSameAccount       = errors.New("cannot transfer to the same account")
)

// Ledger holds a fixed set of accounts, keyed by ID, each with an integer
// balance in minor units. A balance must never go negative.
type Ledger struct {
	mu       sync.Mutex
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
	l.mu.Lock()
	defer l.mu.Unlock()
	v, ok := l.balances[id]
	if !ok {
		return 0, ErrUnknownAccount
	}
	return v, nil
}

// Total returns the sum of every account balance. No sequence of calls to
// Transfer, successful or failed, concurrent or not, may change this value.
func (l *Ledger) Total() int64 {
	l.mu.Lock()
	defer l.mu.Unlock()
	var sum int64
	for _, v := range l.balances {
		sum += v
	}
	return sum
}

// Transfer moves amount from the "from" account to the "to" account.
//
// Safe for concurrent use: the whole check-then-mutate sequence runs under
// a single mutex, so no two calls (however many, in whatever direction) can
// interleave their reads and writes. Holding exactly one lock at a time
// also makes deadlock impossible regardless of call order.
func (l *Ledger) Transfer(from, to string, amount int64) error {
	if amount <= 0 {
		return ErrInvalidAmount
	}
	if from == to {
		return ErrSameAccount
	}

	l.mu.Lock()
	defer l.mu.Unlock()

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
	l.balances[from] -= amount
	l.balances[to] += amount
	return nil
}
EOF
