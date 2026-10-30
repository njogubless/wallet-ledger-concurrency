# Wallet Ledger: Fix Concurrent Transfers

`/app` contains a small Go module, `ledgertask`, with one package: `ledger`
(in `ledger/ledger.go`). It implements a toy double-entry ledger: a fixed
set of accounts, each holding an integer balance in minor units, and a
`Transfer(from, to string, amount int64) error` method that moves money
between two accounts.

`Transfer` is currently **not safe for concurrent use**. Your job is to fix
`ledger/ledger.go` (you may restructure it however you like — add fields,
helper methods, locks, whatever you need) so that it satisfies all of the
following, even when `Transfer` is called concurrently from many goroutines
at once, against overlapping accounts, in mixed directions (e.g. `A->B` and
`B->A` happening at the same time):

1. **No overdraft.** No account's balance may ever go negative, including
   transiently.
2. **Conservation.** The sum of every account's balance must never change,
   regardless of how many transfers succeed, fail, or run concurrently.
3. **No deadlocks.** `Transfer` must return in a bounded, short amount of
   time for any number of concurrent callers and any mix of account pairs —
   it must never block forever.
4. **Correctness is preserved.** A transfer that *should* succeed (sender has
   sufficient funds throughout) must still succeed and move exactly the
   requested amount. A transfer that should fail (insufficient funds,
   unknown account, non-positive amount, same account on both sides) must
   still return the same error it does today.

## Constraints

- Keep the package's public API compatible: `New(map[string]int64) *Ledger`,
  `(*Ledger) Balance(string) (int64, error)`, `(*Ledger) Total() int64`, and
  `(*Ledger) Transfer(string, string, int64) error` must keep their current
  signatures and behavior for the cases described above.
- Stay within the Go standard library.
- Don't change `go.mod`'s module path.

You do not have network access to look up this exact problem; solve it with
what you know about safe concurrent data structures in Go.
