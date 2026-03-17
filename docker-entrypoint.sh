#!/bin/sh
#set -e

#DB_HOST="${DB_HOST:-}"
#DB_PORT="${DB_PORT:-5432}"

#for i in $(seq 1 30); do
#  if [ -n "$DB_HOST" ] && nc -z "$DB_HOST" "$DB_PORT"; then
#    break
#  fi
#  sleep 2
#done

#for i in $(seq 1 5); do
#  migrate -path /app/migrations -database "$DATABASE_URL" up && break || sleep $((i * 5))
#done
echo "🐳 Stubs entrypoint"
