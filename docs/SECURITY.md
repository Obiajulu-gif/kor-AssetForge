# Backend security configuration

For vulnerability reporting, see the root [security policy](../SECURITY.md).

## JWT signing keys

`JWT_SECRET` is mandatory in every environment. The backend exits before
connecting to services if it is absent, blank, or shorter than 32 characters.
It has no fallback. HTTP access/refresh tokens and WebSocket authentication all
use the validated value. The performance test client must use the same secret.

Run `./scripts/generate_jwt_secret.sh` to generate 32 random bytes represented as
64 hexadecimal characters. Supply the result through a secret manager, Docker
secret, or an untracked local `.env` file with restricted permissions. Use a
different key for each environment and for other purposes such as CSRF or webhook
verification. Do not put secrets in source code, logs, PRs, or shared examples.

Startup warns about common placeholder/password words and values with fewer
than eight distinct characters, without logging the secret. This heuristic is
not an entropy guarantee: use a cryptographic generator even when no warning
appears. A long weak value meets the length requirement but still needs replacing.

Rotate keys when compromised and according to your deployment policy. All
instances must receive the same new value. Rotation invalidates existing JWTs,
including refresh tokens, so plan for users to sign in again and WebSocket clients
to reconnect. The application does not currently support overlapping signing keys.

## CSRF protection

When `CSRF_SECRET` is configured, POST, PUT, PATCH, and DELETE requests must send
that exact value in `X-CSRF-Token`. Verification uses
`crypto/subtle.ConstantTimeCompare` rather than string equality. For equal-length
tokens the comparison does not depend on the position of the first mismatch;
tokens of a different length are rejected. GET, HEAD, and OPTIONS remain exempt.
An unset `CSRF_SECRET` preserves the existing behavior of disabling this middleware.

Generate an independent random CSRF value and deliver it only to trusted clients
over HTTPS. This middleware uses a shared token, not a per-session synchronizer
token protocol; configuring it does not replace origin controls, secure cookie
settings, or careful management of which clients receive the token.

Regression tests cover every protected method, equal-length mismatches at the
first/middle/last byte, missing/truncated/longer tokens, and the standard-library
comparison primitive. They avoid flaky wall-clock timing assertions.

## Database queries

Bind values with SQL placeholders, including filters and date granularity.
See [Development](DEVELOPMENT.md) for examples, the SQL review checklist, and the
Semgrep check used by CI.
