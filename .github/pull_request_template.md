## Changes

Describe the problem and resulting behavior. Link the issues this resolves.

## Validation

List the checks run and any skipped checks or known limitations.

## Review checklist

- [ ] SQL values use bound parameters; SQL text contains no request-derived concatenation or formatting.
- [ ] Dynamic identifiers and sort choices use explicit allowlists; embedded migration/seed SQL remains build-time trusted content.
- [ ] `scripts/check_sql_safety.sh` passes, including the unsafe-query detection fixtures.
- [ ] Configuration examples and deployment/security documentation reflect any required settings.
