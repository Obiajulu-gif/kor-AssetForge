# API Changelog

All notable changes to the kor-AssetForge HTTP API are documented here. This
changelog covers API-surface changes only; application changes live in the
top-level [CHANGELOG.md](../../CHANGELOG.md).

The API follows the versioning policy described in
[API_MIGRATION_V1_TO_V2.md](../API_MIGRATION_V1_TO_V2.md): breaking changes are
introduced in a new version (`/api/vN`) and the previous version is deprecated
with a published sunset date.

## API v2

### Added

- `/api/v2/assets` — paginated, filterable asset listing with `asset_type`,
  `min_price`, and `max_price` filters.
- `/api/v2/assets/:id` — single asset retrieval.
- Consistent response envelope: resources and collections wrapped in a `data`
  field with an `api_version` marker.
- Standardised pagination fields: `page`, `page_size`, `total_count`.
- `GET /api/version/migration-guide` — machine-readable v1 → v2 migration data.

### Changed

- Page-size query parameter is `page_size` (was `limit` in v1).
- Page size is capped at `MAX_PAGE_SIZE` (100); oversized requests return
  `400 Bad Request`.
- Errors use a uniform `{ "error": string }` object.

### Security

- Global pagination limits prevent resource-exhaustion via oversized list
  requests.

## API v1

### Deprecated

- **The entire `/api/v1` surface is deprecated.** Responses now include
  `Deprecation: true` and `Sunset: Sat, 31 Dec 2026 23:59:59 GMT` headers.
- Migration target is `/api/v2`. See the migration guide.

### Notes

- v1 remains functional until the sunset date and then returns `410 Gone`.

## Unreleased

### Added

- Initial API changelog.
- Pagination maximum (`MAX_PAGE_SIZE = 100`) enforced globally
  ([#302](https://github.com/barry01-hash/kor-AssetForge/issues/302)).
- Comprehensive v1 → v2 migration guide and enhanced migration endpoint
  ([#301](https://github.com/barry01-hash/kor-AssetForge/issues/301)).
