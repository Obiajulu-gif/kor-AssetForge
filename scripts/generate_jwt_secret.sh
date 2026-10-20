#!/usr/bin/env bash
# Generate 32 random bytes as 64 hexadecimal characters. Prints only the secret.
set -euo pipefail

if ! command -v openssl >/dev/null 2>&1; then
    echo "openssl is required to generate a JWT secret" >&2
    exit 1
fi

openssl rand -hex 32
