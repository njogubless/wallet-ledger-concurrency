package ledger

import (
	"sync"
	"testing"
	"time"
)

// waitWithTimeout waits for wg, failing t if it doesn't finish within d —
// this is how a deadlocked Transfer is detected deterministically rather
// than by the test run simply hanging forever.
func waitWithTimeout(t *testing.T, wg *sync.WaitGroup, d time.Duration, msg string) {
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(d):
		t.Fatal(msg)
	}
}

// TestConcurrentTransfersPreserveInvariants hammers a ring of four accounts
// with many concurrent transfers and checks the two invariants that a
// racy, unsynchronized Transfer violates: no account goes negative, and the
// total money in the system never changes.
func TestConcurrentTransfersPreserveInvariants(t *testing.T) {
	opening := map[string]int64{"A": 1000, "B": 1000, "C": 1000, "D": 1000}
	l := New(opening)
	total := l.Total()
	accounts := []string{"A", "B", "C", "D"}

	const workers = 64
	const rounds = 300

	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(seed int) {
			defer wg.Done()
			for r := 0; r < rounds; r++ {
				from := accounts[(seed+r)%len(accounts)]
				to := accounts[(seed+r+1)%len(accounts)]
				// Errors (e.g. insufficient funds) are an expected, fine
				// outcome; a negative balance or a changed total is not.
				_ = l.Transfer(from, to, 10)
			}
		}(i)
	}
	waitWithTimeout(t, &wg, 15*time.Second, "deadlock: concurrent transfers around a ring of accounts did not complete within 15s")

	if got := l.Total(); got != total {
		t.Fatalf("conservation violated: total went from %d to %d", total, got)
	}
	for _, id := range accounts {
		b, err := l.Balance(id)
		if err != nil {
			t.Fatalf("Balance(%s): %v", id, err)
		}
		if b < 0 {
			t.Fatalf("account %s went negative: %d", id, b)
		}
	}
}

// TestCrossDirectionNoDeadlock specifically hammers A->B and B->A at the
// same time, which deadlocks a naive "lock both accounts, in call order"
// fix that doesn't order its locks consistently.
func TestCrossDirectionNoDeadlock(t *testing.T) {
	l := New(map[string]int64{"A": 1_000_000, "B": 1_000_000})

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			for j := 0; j < 200; j++ {
				_ = l.Transfer("A", "B", 1)
			}
		}()
		go func() {
			defer wg.Done()
			for j := 0; j < 200; j++ {
				_ = l.Transfer("B", "A", 1)
			}
		}()
	}
	waitWithTimeout(t, &wg, 15*time.Second, "deadlock: concurrent A->B and B->A transfers did not complete within 15s")

	if got := l.Total(); got != 2_000_000 {
		t.Fatalf("conservation violated: total = %d", got)
	}
}

// TestNeverOverdraws drives 1000 concurrent 1-unit transfers out of an
// account that only has 100 units, and checks that the account never goes
// negative and the system-wide total is unchanged. A version that checks
// the balance and then mutates it as two separate steps will let far more
// than 100 of these succeed.
func TestNeverOverdraws(t *testing.T) {
	l := New(map[string]int64{"A": 100, "B": 0})

	var wg sync.WaitGroup
	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = l.Transfer("A", "B", 1)
		}()
	}
	waitWithTimeout(t, &wg, 15*time.Second, "deadlock: 1000 concurrent transfers from a single account did not complete within 15s")

	a, err := l.Balance("A")
	if err != nil {
		t.Fatalf("Balance(A): %v", err)
	}
	if a < 0 {
		t.Fatalf("account A went negative: %d", a)
	}
	if got := l.Total(); got != 100 {
		t.Fatalf("conservation violated: total = %d", got)
	}
}

// TestCorrectnessPreserved checks that fixing concurrency safety didn't
// break ordinary sequential behavior: valid transfers still move money,
// and invalid ones still return the documented errors.
func TestCorrectnessPreserved(t *testing.T) {
	l := New(map[string]int64{"A": 500, "B": 10})

	if err := l.Transfer("A", "B", 200); err != nil {
		t.Fatalf("expected a valid transfer to succeed, got: %v", err)
	}
	if b, _ := l.Balance("A"); b != 300 {
		t.Fatalf("A balance = %d, want 300", b)
	}
	if b, _ := l.Balance("B"); b != 210 {
		t.Fatalf("B balance = %d, want 210", b)
	}

	if err := l.Transfer("A", "B", 100000); err != ErrInsufficientFunds {
		t.Fatalf("expected ErrInsufficientFunds, got: %v", err)
	}
	if err := l.Transfer("A", "A", 1); err != ErrSameAccount {
		t.Fatalf("expected ErrSameAccount, got: %v", err)
	}
	if err := l.Transfer("A", "nope", 1); err != ErrUnknownAccount {
		t.Fatalf("expected ErrUnknownAccount, got: %v", err)
	}
	if err := l.Transfer("A", "B", 0); err != ErrInvalidAmount {
		t.Fatalf("expected ErrInvalidAmount, got: %v", err)
	}
}
