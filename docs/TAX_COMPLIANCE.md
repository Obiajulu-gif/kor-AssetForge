# Tax Calculation Methodology and Compliance

This document describes how kor-AssetForge records taxable events, calculates
capital gains and withholding, and produces tax reports. It is the authoritative
reference for the behaviour implemented in `backend/services/tax_calculator.go`,
`backend/services/tax_service.go`, and `backend/models/tax.go`, and complements
[SUPPORTED_JURISDICTIONS.md](SUPPORTED_JURISDICTIONS.md).

> **Not tax or legal advice.** The code and this document implement a simplified
> model. Consult a qualified tax professional before relying on any figure for a
> real filing.

## Scope

kor-AssetForge is a marketplace for tokenised real-world assets. The tax
surface covered here is:

- **Capital gains/losses** on token disposals (sales, trades).
- **Withholding** on distributions (dividends) and certain payments.
- **Annual reports** summarising a user's activity for a tax year.
- **US 1099-NEC** information returns for nonemployee compensation.

The platform currently reports **summary totals only**. It does not file returns
with any tax authority on a user's behalf.

## Units and representation

All monetary amounts are stored and calculated as `int64` **stroops**, the
smallest indivisible unit of a Stellar asset (1 asset unit = 10^7 stroops) or
the minor unit of a fiat currency (cents for USD). The calculator never mixes
denominations; callers pass values in a consistent unit and receive stroops.

| Field | Meaning |
|-------|---------|
| `cost_basis_stroops` | Per-unit acquisition cost |
| `sale_price_stroops` | Per-unit disposal price |
| `quantity` | Units disposed of |
| `capital_gain_loss_stroops` | Realised result of the disposal |
| `withholding_amount_stroops` | Tax withheld from a payment/distribution |
| `withholding_rate` | Rate as a decimal fraction (0.10 = 10%) |

## Calculation methodology

### Capital gains and losses

Implemented in `TaxCalculator.CalculateCapitalGain`
(`backend/services/tax_calculator.go`), called by
`TaxHandler.RecordTaxEvent`:

```
capital_gain_loss = (sale_price - cost_basis) * quantity
```

- A positive result is a gain, a negative result is a loss. This is the IRS
  "amount realised minus adjusted basis" formula for a **single lot**.
- `cost_basis` is **supplied by the caller**; the platform does not currently
  select lots FIFO or by specific identification, and does not track holding
  periods. The stored `TaxRecord` retains cost basis, sale price, quantity, and
  transaction date so the figure can be independently recomputed.
- No wash-sale disallowance, straddle, or minimum-holding-period rules are
  applied.

### Withholding

Implemented in `TaxCalculator.CalculateWithholding`:

```
withheld = gross * rate_basis_points / 10000
```

- One basis point (bp) = 1/100 of a percent, so 10000 bp = 100%.
- Integer division truncates toward zero. This mirrors the on-chain formula in
  `contracts/src/dividend_distributor.rs`
  (`withheld = gross * tax_withholding_rate / 10000`), keeping API and contract
  figures consistent.
- `TaxService.RecordWithholding` stores the withheld amount and the rate as a
  decimal fraction; convert from basis points before recording.

### Annual aggregation

`TaxService.GenerateTaxReport` sums, over the UTC calendar year
`[Jan 1 00:00:00, Dec 31 23:59:59.999999999]`:

| Report field | Source |
|--------------|--------|
| `total_gain_stroops` | `SUM(capital_gain_loss_stroops)` where positive |
| `total_loss_stroops` | `SUM(ABS(capital_gain_loss_stroops))` where negative |
| `net_gain_loss_stroops` | gains − losses |
| `total_income_stroops` | `SUM(withholding_amount_stroops)` |
| `transaction_count` | count of tax records in the period |

Reports are stored with status `draft`. `GetTaxSummary` exposes the same
aggregation for an arbitrary window without persisting anything.

### 1099-NEC generation

`TaxService.Generate1099Form` builds a **draft** US 1099-NEC from a report with
this box mapping:

| 1099-NEC box | Value |
|--------------|-------|
| Box 1a — Nonemployee compensation | `total_income_stroops` |
| Box 3 — Other income | `net_gain_loss_stroops` |
| Box 5 — Federal tax withheld | `total_withheld_tax_stroops` |

Filer name is "kor-AssetForge Marketplace"; `FilerTIN` is a placeholder and must
be replaced with the platform's real EIN before filing.

## Supported jurisdictions

The implementation is **US-federal oriented** and single-jurisdiction. See
[SUPPORTED_JURISDICTIONS.md](SUPPORTED_JURISDICTIONS.md) for the full list, the
per-jurisdiction status table, and the process for adding a jurisdiction.

## Integration with tax services

### Current state

Tax calculation is performed **in-process** against the platform database. There
is **no third-party tax engine integration today** (no Avalara, Vertex, TaxJar,
or IRS e-file endpoints). The gateway/off-ramp providers (Stripe, PayPal) are
used for payments, not tax determination.

Integration points that already exist or are consumable by an external system:

| Interface | Purpose |
|-----------|---------|
| `GET /api/v1/tax/summary` | Machine-readable gain/loss/withholding totals for a window |
| `GET /api/v1/tax/reports` | List stored annual reports |
| `POST /api/v1/tax/exports` | Request a PDF/CSV/JSON export artifact |
| `report_scheduler.go` (`tax_summary`) | Periodic scheduled summaries |
| `models.TaxRecord` / `models.TaxWithholding` | Source-of-truth rows for reconciliation |

> **Note:** the tax routes in `backend/handlers/tax.go` are currently defined but
> not registered in `backend/main.go`. Wiring them up is tracked as a known gap;
> until then only the scheduled `tax_summary` consumer exercises `TaxService`.

### Adding a third-party tax service

1. Implement a provider client behind an interface (e.g. `TaxProvider`) in
   `backend/services/`.
2. Feed it `TaxRecord`/`TaxWithholding` rows (or `GetTaxSummary` output).
3. Keep `TaxCalculator` as the canonical local calculator so figures can be
   audited even when an external engine is unavailable.
4. Document the provider, its jurisdiction coverage, and any data-sharing
   agreement in this file and in SUPPORTED_JURISDICTIONS.md.

## Data model

| Table | Model | Purpose |
|-------|-------|---------|
| `tax_records` | `models.TaxRecord` | Individual taxable disposals |
| `tax_reports` | `models.TaxReport` | Annual aggregated reports |
| `tax_1099_forms` | `models.Tax1099Form` | Draft 1099-NEC returns |
| `tax_withholdings` | `models.TaxWithholding` | Withholding events |
| `tax_exports` | `models.TaxExport` | Export artifacts |

Created by migration
`backend/migrations/sql/0015_create_tax_tables.up.sql`.

## Compliance certifications and standards

The platform is designed to align with the following frameworks. Formal
third-party audits and regulatory registrations should be completed before
production use.

| Standard / obligation | Applicability | Status |
|-----------------------|---------------|--------|
| IRS Form 1099-NEC | US nonemployee compensation reporting | Draft generation implemented; e-file not implemented |
| IRS capital gains rules (single-lot) | US user disposals | Simplified calculation implemented; lot tracking pending |
| FATF Recommendation 11 / AML record-keeping | Retention of tax/compliance records | Aligned with `docs/KYC_SECURITY.md` retention policy |
| GDPR (EU 2016/679) | EU user data, access/erasure | Architecture compliant; tax records retained under legal obligation |
| SOC 2 Type II | Security/confidentiality controls | Controls documented; formal audit pending |

## Worked examples

### Example 1 — capital gain

Acquire 100 units at 1.50 each, sell at 2.00 each:

```
gain = (200 - 150) * 100 = 5000 stroops
```

### Example 2 — capital loss

Acquire 100 units at 2.00 each, sell at 1.50 each:

```
gain = (150 - 200) * 100 = -5000 stroops  (a 5000-stroop loss)
```

### Example 3 — withholding

Withhold 10% (1000 bp) from a 50,000-stroop distribution:

```
withheld = 50000 * 1000 / 10000 = 5000 stroops
```

### Example 4 — record an event (API)

```http
POST /api/v1/tax/records
Content-Type: application/json

{
  "user_id": 42,
  "asset_id": 7,
  "transaction_type": "sale",
  "quantity": 100,
  "cost_basis_stroops": 150,
  "sale_price_stroops": 200,
  "transaction_date": "2026-03-14T12:00:00Z"
}
```

`capital_gain_loss_stroops` is computed as `5000` and stored on the record.

### Example 5 — generate an annual report

```http
POST /api/v1/tax/reports
Content-Type: application/json

{ "user_id": 42, "tax_year": 2026 }
```

Response (`201`) contains `total_gain_stroops`, `total_loss_stroops`,
`net_gain_loss_stroops`, `total_income_stroops`, and `transaction_count`.

## Known gaps

- No lot selection (FIFO/specific identification) or holding-period tracking.
- No wash-sale, straddle, or constructive-sale adjustments.
- Withholding rates are caller-supplied, not derived from a jurisdiction table.
- Tax routes (`backend/handlers/tax.go`) are not yet registered in `main.go`.
- No third-party tax engine or IRS e-file integration.
- Single jurisdiction (US federal); no state or non-US rules.
- Draft reports/forms require manual review; no approval workflow is enforced.

None of the above should be read as tax advice. Confirm current requirements
with a qualified professional before relying on this document for a filing.
