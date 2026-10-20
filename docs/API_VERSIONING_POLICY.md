# API Versioning Policy

Short version of the strategy referenced in issue #304; expand into a
dedicated support-lifecycle doc if/when this needs more detail.

- **Scheme:** URL path versioning — `/api/v1/...`, `/api/v2/...`. The version
  is part of the route, never negotiated via headers or query params.
- **Support window:** a major version is supported for **12 months** after
  the next major version ships, then marked deprecated and later removed.
- **Deprecation process:** a deprecated version returns a
  `Deprecation: true` response header and is announced in `CHANGELOG.md`
  before removal, not silently dropped.
- **Breaking-change criteria:** removing/renaming a field or endpoint,
  changing a field's type or semantics, or tightening validation in a way
  that rejects previously-valid requests — any of these require a new major
  version, not a patch to the current one.
- **Future `v3`:** planned when a breaking change is actually needed (e.g. a
  new auth model), not on a fixed schedule.
