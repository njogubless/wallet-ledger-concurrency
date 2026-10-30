# wallet-ledger-concurrency

A [Harbor](https://github.com/harbor-framework/harbor) evaluation task: can an
AI coding agent find and correctly fix a real concurrency bug in a wallet
ledger, under a verifier that actually proves the fix holds rather than just
checking that the code "looks" fixed?

Built while learning Go, drawing on patterns from my own
[wallet](https://github.com/njogubless/wallet) backend — a double-entry
ledger service where a concurrency bug doesn't just crash, it loses or
creates money.

**Verified results:**

| Agent | Mean reward |
| --- | --- |
| `oracle` (reference fix) | `1.000` |
| `nop` (does nothing) | `0.000` |

## What the agent has to do

The repo ships a small, self-contained Go module — package `ledger` — with
one method: `Transfer(from, to string, amount int64) error`. As shipped, it's
**not safe for concurrent use**: it reads the sender's balance, checks it,
then mutates both balances as two separate, unsynchronized steps. Race two
calls against each other and both can read the same stale balance, both pass
the check, and both succeed — overdrawing an account that should have been
declined.

The agent must fix `ledger/ledger.go` so that, under any number of concurrent,
overlapping, mixed-direction `Transfer` calls:

1. **No overdraft** — no account balance ever goes negative.
2. **Conservation** — the total money in the system never changes.
3. **No deadlocks** — `Transfer` always returns in bounded time.
4. **Correctness preserved** — valid transfers still succeed, invalid ones
   still return the same errors.

Full brief: [`instruction.md`](./instruction.md).

## How it's scored

The verifier runs in a **separate environment** from the agent — a fresh
copy of the agent's container, with only the agent's edited `ledger/` and
`go.mod` restored into it. The hidden test file never touches the agent's
own container, so there's nothing to peek at mid-task.

`tests/ledger_concurrency_test.go` then runs under `go test -race` and checks:

- **`TestConcurrentTransfersPreserveInvariants`** — 64 goroutines × 300
  rounds of transfers around a ring of 4 accounts; no negative balance, total
  unchanged, 15s deadlock timeout.
- **`TestCrossDirectionNoDeadlock`** — hammers `A→B` and `B→A` at the same
  time, the case that deadlocks a naive two-lock fix with no lock ordering.
- **`TestNeverOverdraws`** — 1000 concurrent 1-unit transfers out of a
  100-unit account; a racy check-then-mutate lets far more than 100 succeed.
- **`TestCorrectnessPreserved`** — plain sequential calls confirm the fix
  didn't change documented success/error behavior.
- `go test -race` independently flags the raw unsynchronized memory access.

Single pass/fail reward, written to `/logs/verifier/reward.txt`.

## Layout