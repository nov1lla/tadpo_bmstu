#!/usr/bin/env bash
set -euo pipefail

until pg_isready -h "${POSTGRES_HOST}" -U "${POSTGRES_USER}" -d "${POSTGRES_DB}"; do
  sleep 1
done

psql "postgresql://${POSTGRES_USER}:${POSTGRES_PASSWORD}@${POSTGRES_HOST}:5432/${POSTGRES_DB}?sslmode=disable" -f /migrations/001_init.sql
psql "postgresql://${POSTGRES_USER}:${POSTGRES_PASSWORD}@${POSTGRES_HOST}:5432/${POSTGRES_DB}?sslmode=disable" -f /migrations/002_add_user_credentials.sql
