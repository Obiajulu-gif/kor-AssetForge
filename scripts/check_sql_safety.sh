#!/usr/bin/env bash
# Scan production and test Go code; also verify the rule catches unsafe fixtures.
set -euo pipefail

repo_root="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"
semgrep --test --config .semgrep/sql-safety.yml .semgrep/sql-safety.go
semgrep scan --error --metrics=off --disable-version-check --config .semgrep/sql-safety.yml backend
