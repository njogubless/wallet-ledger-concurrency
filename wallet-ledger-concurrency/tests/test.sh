#!/bin/bash
# Verifier for wallet-ledger-concurrency. Runs in the same (shared)
# container as the agent, so it sees the agent's edited /app/ledger. This
# script is only uploaded to the container at verify time, after the agent's
# turn ends, so the agent never sees these tests while solving the task.
set -u

mkdir -p /logs/verifier

APP_DIR=/app
LEDGER_DIR="$APP_DIR/ledger"

if [ ! -d "$LEDGER_DIR" ]; then
  echo "FAIL: $LEDGER_DIR does not exist" | tee /logs/verifier/output.txt
  echo 0 > /logs/verifier/reward.txt
  exit 0
fi

# Drop the hidden tests into the same package as the agent's fixed code.
cp /tests/ledger_concurrency_test.go "$LEDGER_DIR/ledger_concurrency_test.go"

cd "$APP_DIR"

go test -race -count=1 -timeout=120s ./ledger/... > /logs/verifier/output.txt 2>&1
STATUS=$?

cat /logs/verifier/output.txt

if [ "$STATUS" -eq 0 ]; then
  echo 1 > /logs/verifier/reward.txt
else
  echo 0 > /logs/verifier/reward.txt
fi

exit 0
