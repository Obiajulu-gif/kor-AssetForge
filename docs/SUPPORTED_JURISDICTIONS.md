# Supported Tax Jurisdictions

kor-AssetForge's tax calculation and reporting are **single-jurisdiction
(US federal)** today. This document lists what is supported, what is not, and
how a new jurisdiction would be added. See
[TAX_COMPLIANCE.md](TAX_COMPLIANCE.md) for the calculation methodology.

> **Not tax or legal advice.** Jurisdictional tax rules change frequently.
> Confirm current requirements with a qualified professional before relying on
> this list.

## Summary

| Jurisdiction | Tax model | Status | Notes |
|--------------|-----------|--------|-------|
| **United States (federal)** | Capital gains (single-lot), nonemployee compensation, withholding | **Supported (partial)** | Default and only implemented jurisdiction. See details below. |
| United States (state) | Varies by state | **Not supported** | No state-level sourcing or rates. |
| European Union | VAT/GST and member-state capital gains | **Not supported** | No VAT/GST determination. |
| United Kingdom | CGT, VAT | **Not supported** | — |
| Canada | Capital gains inclusion, GST/HST | **Not supported** | — |
| Australia | CGT, GST | **Not supported** | — |
| All other jurisdictions | — | **Not supported** | Not researched or implemented. |

## United States (federal) — supported (partial)

What is implemented:

- **Capital gains/losses** for a single disposal, using
  `(sale price − cost basis) × quantity`, in stroops. See
  [TAX_COMPLIANCE.md#calculation-methodology](TAX_COMPLIANCE.md).
- **Annual aggregation** per UTC calendar year (`TaxReport`).
- **1099-NEC draft generation** (Box 1a, Box 3, Box 5 mapping).
- **Withholding recording** for distributions and payments.

What is **not** implemented even for the US:

- Lot selection (FIFO vs. specific identification) and short-term/long-term
  holding-period splits.
- Wash-sale disallowance and other loss-deferral rules.
- State and local tax determination.
- IRS e-filing / MeF transmission (forms are generated as drafts only).
- Real filer EIN — `Tax1099Form.FilerTIN` is a placeholder.

## Withholding rates

Withholding is **jurisdiction-agnostic** at the calculation layer: the rate is
an integer number of basis points supplied by the caller
(`TaxCalculator.CalculateWithholding`). At the contract layer,
`contracts/src/dividend_distributor.rs` applies the same basis-point formula
using a `tax_withholding_rate` configured per distribution. This means a
jurisdiction's rate can be applied today without a code change, but the platform
does not automatically choose the correct rate for a user's jurisdiction.

| Source of rate | Where set | Format |
|----------------|-----------|--------|
| API withholding record | `TaxWithholding.WithholdingRate` | Decimal fraction (0.10 = 10%) |
| On-chain distribution | `dividend_distributor::tax_withholding_rate` | Basis points (1000 = 10%) |

## How jurisdiction support would be added

1. **Define the rule set**: capital-gain formula variations, holding-period
   thresholds, rate tables, and reporting forms for the jurisdiction.
2. **Model it**: add a jurisdiction resolver/rate table and extend
   `TaxCalculator` (or add a sibling calculator) without changing existing
   behaviour for US users.
3. **Persist jurisdiction** on `TaxRecord`/`TaxWithholding` (or resolve it from
   the user's verified address).
4. **Report**: add jurisdiction-specific report/form generators.
5. **Document**: update this file, TAX_COMPLIANCE.md, and any applicable
   certifications.
6. **Test**: add calculation fixtures and edge cases for the new jurisdiction.

Until a jurisdiction is explicitly listed as supported above, figures produced
by this platform are **not** valid for that jurisdiction's filing.

## Related files

| File | Purpose |
|------|---------|
| `backend/services/tax_calculator.go` | Pure calculation rules |
| `backend/services/tax_service.go` | Aggregation and reporting |
| `backend/models/tax.go` | Tax data model |
| `contracts/src/dividend_distributor.rs` | On-chain withholding formula |
| `docs/TAX_COMPLIANCE.md` | Full methodology and compliance posture |
| `docs/ACCREDITED_INVESTOR_VERIFICATION.md` | Jurisdiction gaps for accreditation |
