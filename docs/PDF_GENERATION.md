# PDF Generation

This document describes PDF generation functionality in the AssetForge frontend
using jsPDF.

## Overview

The frontend provides client-side PDF report generation for:

- **Portfolio reports**: Asset holdings with gains/losses
- **Transaction history**: Complete transaction records
- **Tax documents**: Capital gains/losses for tax reporting

PDF generation is handled by `frontend/src/lib/pdf-generator.ts` using
[jsPDF 2.5.x](https://github.com/parallax/jsPDF) and
[jspdf-autotable](https://github.com/simonbengtsson/jsPDF-AutoTable) for table
rendering.

## Usage

### Programmatic Generation

```typescript
import { generatePDF, ReportType, PortfolioRow } from "@/lib/pdf-generator";

const portfolioData: PortfolioRow[] = [
  {
    asset: "Stellar Lumens",
    symbol: "XLM",
    quantity: 10000,
    avgCost: 0.12,
    currentPrice: 0.15,
    totalValue: 1500,
    gainLoss: 300,
    gainLossPct: 25.0,
  },
  // ... more rows
];

await generatePDF({
  type: "portfolio",
  title: "Q1 2026 Portfolio Report",
  subtitle: "All Holdings",
  portfolioRows: portfolioData,
  branding: {
    companyName: "AssetForge",
    primaryColor: [99, 102, 241], // RGB
    logoDataUrl: "data:image/png;base64,...",
  },
  chartImageDataUrl: "data:image/png;base64,...", // optional chart
  onProgress: (pct) => console.log(`${pct}% complete`),
});
```

### Using the Export Button Component

```tsx
import { ExportButton } from "@/components/ExportButton";

<ExportButton
  reportType="portfolio"
  title="My Portfolio"
  portfolioRows={data}
  branding={brandingOptions}
/>
```

The button handles loading states, progress display, and error messaging
automatically.

## Report Types

### Portfolio Report

Displays current holdings with:
- Asset name and symbol
- Quantity held
- Average cost basis and current price
- Total value
- Gain/loss amount and percentage (color-coded)
- Summary statistics (total value, total gain/loss)

### Transaction Report

Lists all transactions with:
- Date, type, asset
- Quantity and price
- Total amount
- Status

### Tax Document

Capital gains/losses for tax filing:
- Asset acquired and disposed dates
- Proceeds and cost basis
- Gain/loss (color-coded)
- Short-term vs long-term classification
- Net capital gain/loss summary

## Customization

### Branding

```typescript
const branding: BrandingOptions = {
  companyName: "Your Company",        // Header title
  primaryColor: [99, 102, 241],       // RGB for header/accents
  logoDataUrl: "data:image/png;...",  // Base64 PNG/JPEG logo (14×14mm)
};
```

### Chart Embedding

Pass a base64-encoded PNG chart image via `chartImageDataUrl`. The chart is
rendered at full page width (minus margins) before the data table.

```typescript
const canvas = document.querySelector("canvas");
const chartDataUrl = canvas.toDataURL("image/png");

await generatePDF({
  type: "portfolio",
  chartImageDataUrl: chartDataUrl,
  // ...
});
```

## PDF Output

Generated PDFs include:

- **Header bar** with company branding and generation date
- **Title section** with report type and optional subtitle
- **Optional chart** (if provided)
- **Data table** with alternating row colors and color-coded gains/losses
- **Footer** on every page with confidentiality notice and page numbers

PDFs are automatically downloaded with a timestamped filename:
`{company}-{type}-report-{timestamp}.pdf`

## Browser Compatibility

jsPDF works in all modern browsers:
- Chrome/Edge 90+
- Firefox 88+
- Safari 14+

For large reports (>1000 rows), generation may take several seconds. The
`onProgress` callback provides user feedback during generation.

## Known Limitations

- **Logo size**: Fixed at 14×14mm. Provide square logos for best results.
- **Chart dimensions**: Fixed aspect ratio (page width × 60mm height). Provide
  appropriately sized chart images.
- **Font support**: Uses built-in Helvetica. Custom fonts require additional
  configuration (see jsPDF documentation).
- **Memory**: Very large reports (>5000 rows) may cause performance issues in
  some browsers.

## Upgrade History

**v2.5.2** (Issue #435): Upgraded from jsPDF 4.2.1 to 2.5.2 for security fixes
and stability. No breaking API changes; existing PDF generation code continues
to work without modification.

## References

- [jsPDF Documentation](https://github.com/parallax/jsPDF)
- [jsPDF-AutoTable Documentation](https://github.com/simonbengtsson/jsPDF-AutoTable)
