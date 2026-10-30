# njogubless/wallet-ledger-concurrency

## What the agent does

The agent is handed a small Go module (`ledgertask`, package `ledger`) that
implements an in-memory double-entry wallet ledger: a fixed set of accounts
with integer balances and a `Transfer(from, to, amount) error` method. The
shipped `Transfer` is not safe for concurrent use — it reads the sender's
balance, checks it, and then mutates both balances as two separate,
unsynchronized steps, so concurrent calls can race past the check and
overdraw an account. The agent must fix `ledger/ledger.go` so the ledger
never overdraws, never loses or creates money, and never deadlocks under
concurrent `Transfer` calls, while keeping the package's public API and its
documented errors intact. Full brief: `instruction.md`.

This mirrors a real failure mode in production wallet/ledger systems (the
author's own [wallet](https://github.com/njogubless/wallet) project guards
the equivalent step with per-account row locks taken in a fixed order), cut
down to a self-contained, dependency-free Go package so it can be graded
deterministically with the race detector instead of a database.

## Environment

- Base image: `golang:1.23-bookworm`.
- No extra packages installed; only the Go toolchain already in the base
  image.
- Starter code is copied into `/app` at image build time from
  `environment/starter/` (`go.mod` + `ledger/ledger.go`).
- No GPU. Network is disabled (`no-network`) — the fix only needs the Go
  standard library (`sync`). 2 CPUs, 2048 MB RAM.
- Agent timeout: 7200s (2 hours), sized for a long-horizon task even though
  a correct fix is small.

## Verifier

- Separate verifier environment (`[verifier] environment_mode = "separate"`):
  grading runs in a fresh copy of the agent image, not the container the
  agent used, so the hidden test file is never present in the agent's
  container at any point.
- The agent's edited `/app/ledger` and `/app/go.mod` are declared as
  `artifacts` and restored into that fresh container before grading.
- `tests/ledger_concurrency_test.go` is uploaded to `/tests/` and copied by
  `tests/test.sh` into `/app/ledger/` (same package as the agent's code) so
  it compiles against the agent's fix.
- `test.sh` runs `go test -race -count=1 -timeout=120s ./ledger/...` and
  writes `1` to `/logs/verifier/reward.txt` on success, `0` otherwise.
  Single pass/fail reward — no partial credit.
- What the hidden tests check:
  - `TestConcurrentTransfersPreserveInvariants` — 64 goroutines x 300 rounds
    of transfers around a ring of 4 accounts; asserts no negative balance
    and the total is unchanged, with a 15s deadlock timeout.
  - `TestCrossDirectionNoDeadlock` — hammers `A->B` and `B->A` at the same
    time (the case that deadlocks a naive two-lock fix that doesn't order
    its locks), with the same 15s timeout and conservation check.
  - `TestNeverOverdraws` — 1000 concurrent 1-unit transfers out of a
    100-unit account; a racy check-then-mutate implementation lets far more
    than 100 succeed.
  - `TestCorrectnessPreserved` — plain sequential calls confirm the fix
    didn't change the documented success/error behavior.
  - `go test -race` also catches the unsynchronized concurrent map access
    directly, independent of the invariant checks above.

## Layout

```
wallet-ledger-concurrency/
├── instruction.md                       # Task prompt (no mention of tests)
├── task.toml                            # separate verifier, no-network, keywords/metadata
├── README.md
├── environment/
│   ├── Dockerfile                       # golang:1.23-bookworm, COPY starter/ -> /app
│   └── starter/
│       ├── go.mod                       # module ledgertask
│       └── ledger/ledger.go             # buggy Transfer the agent must fix
├── solution/
│   └── solve.sh                         # reference fix: single sync.Mutex
└── tests/
    ├── test.sh                          # builds + runs the hidden test, writes reward.txt
    └── ledger_concurrency_test.go        # hidden concurrency/invariant tests
```

## Running

```bash
# Sanity-check the reference solution scores 1.0
harbor run -p wallet-ledger-concurrency -a oracle

# Confirm doing nothing scores 0.0
harbor run -p wallet-ledger-concurrency -a nop

# Try a real agent
harbor run -p wallet-ledger-concurrency -a terminus-2 -m anthropic/claude-sonnet-4-6
```
