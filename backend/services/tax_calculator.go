package services

// TaxCalculator contains the pure, side-effect-free arithmetic behind
// kor-AssetForge's tax reporting. It performs no database access and no I/O so
// that the rules below can be unit tested and audited in isolation.
//
// # Units
//
// Every monetary argument is expressed in **stroops**, the smallest indivisible
// unit of a Stellar asset (1 unit = 10^7 stroops, i.e. 7 decimal places), or in
// the smallest unit of the relevant fiat currency (cents for USD). The
// calculator never converts between units; callers are responsible for passing
// values in a consistent denomination. All results are returned as int64
// stroops.
//
// # Scope and limitations
//
// The rules implemented here are a simplified, jurisdiction-agnostic core. See
// docs/TAX_COMPLIANCE.md and docs/SUPPORTED_JURISDICTIONS.md for the full
// methodology, supported jurisdictions, and the list of known gaps (e.g. cost
// basis is caller-supplied rather than derived from FIFO/lot tracking, and no
// wash-sale or minimum-holding-period rules are applied).
type TaxCalculator struct{}

// NewTaxCalculator constructs a TaxCalculator. The type is stateless, so a
// zero value is equally usable.
func NewTaxCalculator() *TaxCalculator {
	return &TaxCalculator{}
}

// CalculateCapitalGain returns the realised capital gain or loss, in stroops,
// for a single disposal.
//
//	gain = (salePrice - costBasis) * quantity
//
// Where:
//   - costBasis is the per-unit acquisition cost in stroops,
//   - salePrice is the per-unit disposal price in stroops,
//   - quantity is the number of units disposed of.
//
// A positive result is a gain; a negative result is a loss. This matches the
// IRS "amount realised minus adjusted basis" formula for a single lot and is
// what is persisted on models.TaxRecord.CapitalGainLossStroops. It does not
// apply lot selection (FIFO/specific identification), wash-sale disallowance,
// or the distinction between short- and long-term holding periods.
func (c *TaxCalculator) CalculateCapitalGain(costBasisStroops, salePriceStroops, quantity int64) int64 {
	return (salePriceStroops - costBasisStroops) * quantity
}

// CalculateWithholding returns the tax to withhold from a gross payment, in
// stroops, given an integer rate in basis points.
//
//	withheld = gross * rateBasisPoints / 10000
//
// One basis point (bp) is 1/100 of a percent, so 10000 bp = 100%. Integer
// division truncates toward zero, mirroring the on-chain withholding formula in
// contracts/src/dividend_distributor.rs. Non-positive gross amounts or rates
// yield zero so callers never withhold from nothing.
func (c *TaxCalculator) CalculateWithholding(grossStroops, rateBasisPoints int64) int64 {
	if grossStroops <= 0 || rateBasisPoints <= 0 {
		return 0
	}
	return (grossStroops * rateBasisPoints) / 10000
}

// NetGainLoss returns totalGain - totalLoss. Callers pass losses as positive
// magnitudes (see TaxService.GenerateTaxReport).
func (c *TaxCalculator) NetGainLoss(totalGainStroops, totalLossStroops int64) int64 {
	return totalGainStroops - totalLossStroops
}
