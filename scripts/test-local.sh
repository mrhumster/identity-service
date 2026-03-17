#!/bin/bash

set -e

NAMESPACE="go-app"
LOCAL_DB_PORT=5432

echo "🔌 Setting up port forwarding to Kubernetes PostgreSQL..."

kubectl port-forward -n $NAMESPACE svc/postgresql $LOCAL_DB_PORT:5432 >/dev/null 2>&1 &
PF_PID=$!

sleep 3

if ! ps -p $PF_PID >/dev/null; then
  echo "❌ Failed to establish port forwarding"
  exit 1
fi

echo "✅ Port forwarding established (PID: $PF_PID)"

cleanup() {
  echo "🧹 Cleaning up..."
  kill $PF_PID 2>/dev/null || true
  wait $PF_PID 2>/dev/null || true
}

trap cleanup EXIT

export TEST_DB_HOST="localhost"
export TEST_DB_PORT="$LOCAL_DB_PORT"
export DB_USER="postgres"
export DB_PASS="Master1234"
export TEST_DB_NAME="test_database1"

echo "🧪 Running tests with database on localhost:$LOCAL_DB_PORT..."

go test ./...

TEST_EXIT_CODE=$?

if [ $TEST_EXIT_CODE -eq 0 ]; then
  echo "✅ All tests passed!"
else
  echo "❌ Tests failed with exit code: $TEST_EXIT_CODE"
fi

exit $TEST_EXIT_CODE
