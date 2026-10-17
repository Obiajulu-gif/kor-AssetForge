# API v1 → v2 Migration Guide

This guide explains how to migrate from **API v1** to **API v2** of
kor-AssetForge. API v1 is **deprecated** and will be removed on the sunset date
below. The machine-readable equivalent of this guide is available at runtime:

```http
GET /api/version/migration-guide
```

> **TL;DR** — change `/api/v1` → `/api/v2`, rename `limit` → `page_size`, and
> read list data from the `data` field of the response envelope.

## 1. Timeline

| Date       | Milestone                                                              |
| ---------- | ---------------------------------------------------------------------- |
| 2025-01-01 | v2 GA. v1 marked deprecated and returns `Deprecation`/`Sunset` headers. |
| 2026-06-30 | v1 feature freeze. Only critical security fixes are accepted on v1.     |
| 2026-12-31 | **v1 sunset.** v1 endpoints are removed and return `410 Gone`.          |

The `Sunset` HTTP header on v1 responses is set to
`Sat, 31 Dec 2026 23:59:59 GMT`.

## 2. Breaking changes summary

1. **Base path** moved from `/api/v1` to `/api/v2` for migrated resources.
2. **Response envelope** — v2 wraps resources and collections in a `data`
   field and adds an `api_version` marker.
3. **Pagination fields** — v1 used `limit` and a nested `links` object; v2 uses
   `page_size`. Both are capped at the global maximum of **100**.
4. **Consistent errors** — v2 uses a uniform `{ "error": string }` object.
5. **Version signaling** — v1 responses always carry `Deprecation: true` and
   `Sunset` headers. During the transition you may send `X-API-Version: v2`.
6. **Version conflicts** — supplying a path version and a conflicting header
   version returns `400 Bad Request`.

## 3. Endpoint mapping (v1 → v2)

| v1 endpoint                     | v2 endpoint                     | Notes                                             |
| ------------------------------- | ------------------------------- | ------------------------------------------------- |
| `GET /api/v1/assets`            | `GET /api/v2/assets`            | `limit` → `page_size`; response wrapped in `data`. |
| `GET /api/v1/assets/:id`        | `GET /api/v2/assets/:id`        | Resource wrapped in a `data` object.              |
| `GET /api/v1/health`            | `GET /health`                   | Health endpoints are versionless.                 |
| `GET /api/v1/marketplace`       | `GET /api/v2/assets` (filtered) | Marketplace listings are exposed as assets in v2. |

Endpoints not listed above (for example `GET /api/v1/compliance/report`) have no
v2 equivalent **yet** and continue to work on v1 until they are migrated. They
will be added to this table as they move.

## 4. Request format changes

| Concern          | v1                                   | v2                                          |
| ---------------- | ------------------------------------ | ------------------------------------------- |
| Page size param  | `?limit=25`                          | `?page_size=25`                             |
| Page param       | `?page=1`                            | `?page=1`                                   |
| Max page size    | 100 (enforced)                       | 100 (enforced globally)                     |
| Version signal   | path only                            | `X-API-Version: v2` or `Accept` negotiation |
| Content type     | `application/json`                   | `application/json`                          |

### Version negotiation

Send one of the following to select v2 without changing the path (useful during
the transition):

```http
X-API-Version: v2
```

```http
Accept: application/vnd.assetforge.v2+json
```

If the path version and the negotiated version disagree, the API responds with
`400 Bad Request`:

```json
{
  "error": "conflicting API versions",
  "path_version": "v1",
  "requested_version": "v2"
}
```

## 5. Response format changes

### v1 list response (deprecated)

```json
{
  "limit": 10,
  "page": 1,
  "total": 128,
  "total_pages": 13,
  "has_next": true,
  "has_prev": false,
  "links": {
    "self": "/api/v1/assets?page=1&limit=10",
    "first": "/api/v1/assets?page=1&limit=10",
    "next": "/api/v1/assets?page=2&limit=10",
    "last": "/api/v1/assets?page=13&limit=10"
  },
  "data": []
}
```

### v2 list response

```json
{
  "data": [],
  "page": 1,
  "page_size": 10,
  "total_count": 128,
  "api_version": "v2"
}
```

Key differences:

- `limit` → `page_size`
- `total` → `total_count`
- The nested `links` object is not present in v2; use `page`/`page_size` to
  build the next request, or follow the `Link` response header where provided.
- `api_version` is always present and set to `"v2"`.

### v1 single-resource response (deprecated)

```json
{ "id": 42, "name": "Test Property 1", "symbol": "TEST1" }
```

### v2 single-resource response

```json
{
  "data": { "id": 42, "name": "Test Property 1", "symbol": "TEST1" },
  "api_version": "v2"
}
```

### Errors

v1 and v2 both return an `error` string, but v2 guarantees a consistent shape:

```json
{ "error": "requested page size exceeds the maximum allowed" }
```

## 6. Code examples

### Before (v1)

```bash
curl -s "https://api.example.com/api/v1/assets?page=1&limit=50" \
  -H "Authorization: Bearer $TOKEN"
```

```json
{
  "limit": 50,
  "page": 1,
  "total": 128,
  "links": { "next": "/api/v1/assets?page=2&limit=50" },
  "data": []
}
```

### After (v2)

```bash
curl -s "https://api.example.com/api/v2/assets?page=1&page_size=50" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Accept: application/vnd.assetforge.v2+json"
```

```json
{
  "data": [],
  "page": 1,
  "page_size": 50,
  "total_count": 128,
  "api_version": "v2"
}
```

### JavaScript (fetch)

```js
// v1 (deprecated)
const v1 = await fetch(`/api/v1/assets?page=1&limit=50`, {
  headers: { Authorization: `Bearer ${token}` },
}).then((r) => r.json());
const itemsV1 = v1.data;

// v2
const v2 = await fetch(`/api/v2/assets?page=1&page_size=50`, {
  headers: {
    Authorization: `Bearer ${token}`,
    Accept: "application/vnd.assetforge.v2+json",
  },
}).then((r) => r.json());
const itemsV2 = v2.data;
const total = v2.total_count;
```

## 7. Migration checklist

- [ ] Replace `/api/v1` prefixes with `/api/v2` for migrated endpoints.
- [ ] Rename `limit` query parameters to `page_size`.
- [ ] Read collections from the `data` field.
- [ ] Read `total_count`/`page_size` instead of `total`/`limit`.
- [ ] Update error handling to read the `error` string field.
- [ ] Add `X-API-Version: v2` or `Accept` negotiation if you cannot change paths
      immediately.
- [ ] Remove any code that relies on v1 `Deprecation`/`Sunset` headers.
- [ ] Run your integration tests against `/api/v2` before the sunset date.

## 8. Support

- **Email:** `api-support@assetforge.example`
- **Issues:** <https://github.com/parkerwinner/kor-AssetForge/issues>
- **Runtime metadata:** `GET /api/version/migration-guide`
- **General API reference:** [API.md](API.md)
- **API changelog:** [api/CHANGELOG.md](api/CHANGELOG.md)

When opening a migration issue, include the endpoint, the v1 request/response,
and the error you observed so we can help quickly.
