# Payment Idempotency

kor-AssetForge uses **idempotency keys** on fiat payment creation so that a
client can safely retry a request without risking a duplicate charge. This
document describes the contract, the server-side implementation, and how to
integrate with it.

## Why

Fiat payment creation talks to an external gateway (Stripe / PayPal) and writes
a `Payment` row. If the client times out or the network drops after the request
reaches the server, the client cannot tell whether the charge was created. Naive
retries would create a second `Payment` and a second gateway charge. An
idempotency key makes the operation *idempotent*: replaying the same request
returns the original result.

## Contract

### Supplying a key

Every `POST /api/v1/payments` request **must** carry an idempotency key, supplied
either as a header (preferred, Stripe-style) or in the request body:

| Location | Name | Notes |
|----------|------|-------|
| Header | `Idempotency-Key` (or `X-Idempotency-Key`) | Takes precedence over the body field. |
| Body | `idempotency_key` | JSON string, 8–255 characters. |

Keys must be **8–255 characters**. A key that is missing or too short is
rejected with `400 Bad Request`. Clients should generate a fresh,
high-entropy key (e.g. a UUID) once per logical payment and reuse it verbatim on
every retry of that payment.

```http
POST /api/v1/payments
Authorization: Bearer <token>
Idempotency-Key: 8f14e45f-ea8b-4d0e-9f5a-2b6c1d7e3a90
Content-Type: application/json

{
  "asset_id": 42,
  "amount_fiat": 15000,
  "currency": "USD",
  "gateway": "stripe"
}
```

### Responses

| Scenario | Status | Body |
|----------|--------|------|
| First request | `201 Created` | Payment session, `idempotent_replay: false` |
| Retry with the same key **and** identical parameters (within TTL) | `200 OK` | The **same** payment session, `idempotent_replay: true` |
| Same key reused within TTL with **different** parameters | `409 Conflict` | `idempotency key already used with different payment parameters` |
| Missing / malformed key | `400 Bad Request` | Validation error |

Successful response body:

```json
{
  "payment_id": 101,
  "gateway_payment_id": "pi_stripe_42_7_1730000000000",
  "checkout_url": "https://checkout.stripe.com/pay/pi_stripe_42_7_1730000000000",
  "expires_at": 1730001800,
  "status": "pending",
  "idempotency_key": "8f14e45f-ea8b-4d0e-9f5a-2b6c1d7e3a90",
  "idempotent_replay": false
}
```

`amount_fiat` is in the currency's minor unit (e.g. cents). Parameter matching
covers `asset_id`, `amount_fiat`, `currency`, and `gateway`.

### Key lifetime

Keys are honoured for **24 hours** (`services.IdempotencyKeyTTL`). After that
window the original payment no longer counts as a replay: the stored key is
released and a request with the same key creates a new payment. Retries should
therefore happen well within 24 hours.

### Scope

Idempotency is **per user**: the uniqueness constraint is
`(user_id, idempotency_key)`. Two different users may use the same key value
without interfering.

## Implementation

| Layer | File | Behaviour |
|-------|------|-----------|
| Schema | `backend/migrations/sql/0042_add_payment_idempotency.up.sql` | Adds `payments.idempotency_key` (NOT NULL) and the unique index `idx_payments_user_idempotency_key (user_id, idempotency_key)`. Existing rows are backfilled to `legacy_<id>`. |
| Model | `backend/models/payment.go` | `Payment.IdempotencyKey` with the composite unique index. |
| Service | `backend/services/payment_gateway.go` | `CreateCheckoutSession` requires a key, looks up an active payment, returns it as a replay, or returns `ErrIdempotencyConflict` / `ErrGatewayUnsupported`. Expired keys are tombstoned so they can be reused. |
| Handler | `backend/handlers/payment.go` | Resolves the key from header/body, maps service errors to `400`/`409`, and sets `idempotent_replay` + `200` vs `201`. |
| Tests | `backend/handlers/payment_test.go` | Covers missing keys, replay, conflicts, TTL expiry, per-user scoping, and access control. |

### Concurrency

The unique index is the source of truth. If two requests with the same key race,
one insert wins and the loser's `Create` fails; the service then re-reads the
winning payment and returns it as an idempotent replay instead of surfacing a
database error.

## Client guidance

1. Generate the key **before** sending the first request.
2. Persist the key alongside your own order/attempt identifier.
3. On timeout or connection error, retry with the same body and key.
4. Treat `409 Conflict` as a client bug: the same key was sent with different
   parameters.
5. Do **not** reuse a key for a genuinely new payment after success — generate a
   new one (unless the 24-hour TTL has elapsed).

## Related files

| File | Purpose |
|------|---------|
| `backend/handlers/payment.go` | Payment HTTP handlers |
| `backend/services/payment_gateway.go` | Gateway orchestration and idempotency logic |
| `backend/models/payment.go` | `Payment` model |
| `backend/migrations/sql/0035_create_payments.up.sql` | Base payments schema |
| `backend/migrations/sql/0042_add_payment_idempotency.up.sql` | Idempotency column and index |
| `docs/MULTI_CURRENCY_SUPPORT.md` | Currency handling for payments |
