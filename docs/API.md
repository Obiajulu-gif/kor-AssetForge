# kor-AssetForge API Reference

This document describes cross-cutting conventions for the kor-AssetForge REST
API, with a focus on pagination. For endpoint-specific details see the Swagger
UI served at `/swagger/index.html` and the [API Client SDK](API_CLIENT_SDK.md).

## Base URLs

| Version | Base path   | Status                      |
| ------- | ----------- | --------------------------- |
| v1      | `/api/v1`   | Deprecated (sunset 2026-12-31) |
| v2      | `/api/v2`   | Current                     |

See [API_MIGRATION_V1_TO_V2.md](API_MIGRATION_V1_TO_V2.md) for migration details.

## Authentication

Protected endpoints require a bearer token:

```
Authorization: Bearer <access_token>
```

## Pagination

List endpoints accept the following query parameters:

| Parameter   | Type | Default | Maximum | Description                         |
| ----------- | ---- | ------- | ------- | ----------------------------------- |
| `page`      | int  | `1`     | —       | 1-based page number.                |
| `limit`     | int  | per endpoint | **100** | Number of records per page.   |
| `page_size` | int  | per endpoint | **100** | Alias for `limit` (v2 endpoints). |

### Maximum page size

The API enforces a **global maximum page size of 100 records** (`MAX_PAGE_SIZE`).
This bound protects database and network resources from requests that would
return thousands of records in a single response.

Behaviour:

- A missing `limit`/`page_size` is allowed; the endpoint applies its own default.
- A requested page size of **100 or fewer** is honoured (subject to the
  endpoint default when no value is supplied).
- A requested page size **greater than 100** is rejected with
  **HTTP 400 Bad Request** before the handler runs.
- A malformed (non-numeric) value is left to the handler's default.

Example rejection:

```http
GET /api/v1/assets?limit=5000 HTTP/1.1
```

```json
{
  "error": "requested page size exceeds the maximum allowed",
  "param": "limit",
  "max": 100,
  "message": "reduce limit/page_size to 100 or fewer"
}
```

### Enforcement

Page-size limits are enforced in two layers:

1. **Pagination middleware** (`backend/middleware/pagination.go`) runs globally
   and returns `400` for any request whose `limit`/`page_size` exceeds
   `MAX_PAGE_SIZE`.
2. **Handler-level clamping** (`utils.ClampPageSize`, `handlers.clampPageLimit`)
   caps values defensively so a handler can never issue an unbounded query even
   if the middleware is bypassed.

The shared bound is defined once in `backend/constants/constants.go`.

### Paginated response envelope

Handlers that use the shared pagination helper (`utils.Paginate`) return:

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

A standard `Link` response header is also emitted with `self`, `first`, `last`
(and `prev`/`next` where applicable) relations.

### Error format

Errors are returned as JSON with an `error` field:

```json
{ "error": "requested page size exceeds the maximum allowed" }
```

Some endpoints use a richer envelope; see the endpoint's Swagger definition.

## Rate limiting

Most endpoints are rate limited. When a limit is exceeded the API responds with
`429 Too Many Requests` and a `Retry-After` header.

## Support

For API support, open an issue at
<https://github.com/barry01-hash/kor-AssetForge/issues> or email
`api-support@assetforge.example`.
