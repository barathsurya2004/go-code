#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SERVICE_DIR="${SCRIPT_DIR}/../src/penne-service"

echo "🔄 Rebasing local database from production..."
cd "${SERVICE_DIR}"
go run ./cmd/dbshadow "$@"
