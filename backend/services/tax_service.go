package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/yourusername/kor-assetforge/models"
	"gorm.io/gorm"
)

// TaxService provides tax record keeping and reporting for kor-AssetForge.
//
// It records taxable events (disposals, withholdings), aggregates them into
// annual reports, generates US 1099-NEC forms, and produces export artifacts.
// The pure arithmetic it depends on lives in TaxCalculator; this type owns the
// persistence and aggregation.
//
// All monetary values are int64 stroops (see TaxCalculator for the unit
// definition). The methodology, supported jurisdictions, and current limitations
// are documented in docs/TAX_COMPLIANCE.md and docs/SUPPORTED_JURISDICTIONS.md.
type TaxService struct {
	db *gorm.DB
}

func NewTaxService(db *gorm.DB) *TaxService {
	return &TaxService{db: db}
}

// RecordTaxEvent persists a taxable event. Callers are expected to have already
// populated CapitalGainLossStroops (typically via CalculateCapitalGain) so the
// stored figure can be audited against the inputs (cost basis, sale price,
// quantity, transaction date) that remain on the record.
func (ts *TaxService) RecordTaxEvent(record *models.TaxRecord) error {
	if record.UserID == 0 || record.AssetID == 0 {
		return errors.New("user_id and asset_id are required")
	}
	return ts.db.Create(record).Error
}

// CalculateCapitalGain returns the realised gain or loss, in stroops, for a
// disposal: (salePrice - costBasis) * quantity. See
// TaxCalculator.CalculateCapitalGain for the formula, unit conventions, and
// the lot-tracking/wash-sale rules that are intentionally out of scope.
func (ts *TaxService) CalculateCapitalGain(costBasis, salePrice, quantity int64) int64 {
	return NewTaxCalculator().CalculateCapitalGain(costBasis, salePrice, quantity)
}

// GenerateTaxReport builds and stores an annual tax report for a user.
//
// The reporting period is the calendar year [Jan 1 00:00:00 UTC, Dec 31
// 23:59:59.999999999 UTC] of taxYear. It aggregates:
//   - total capital gains: SUM(capital_gain_loss_stroops) over records with a
//     positive value,
//   - total capital losses: SUM(ABS(capital_gain_loss_stroops)) over records
//     with a negative value,
//   - total withheld income: SUM(withholding_amount_stroops) from TaxWithholding,
//   - transaction count: number of TaxRecord rows in the period.
//
// NetGainLossStroops is total gains minus total losses. The report is stored
// with ReportStatus "draft" and must be reviewed/approved before use; it is not
// automatically filed with any tax authority.
func (ts *TaxService) GenerateTaxReport(userID uint, taxYear int) (*models.TaxReport, error) {
	startDate := time.Date(taxYear, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(taxYear, 12, 31, 23, 59, 59, 999999999, time.UTC)

	var totalGain, totalLoss, totalIncome int64
	var recordCount int64

	ts.db.Model(&models.TaxRecord{}).
		Where("user_id = ? AND transaction_date >= ? AND transaction_date <= ? AND capital_gain_loss_stroops > ?",
			userID, startDate, endDate, 0).
		Select("SUM(capital_gain_loss_stroops)").
		Row().Scan(&totalGain)

	ts.db.Model(&models.TaxRecord{}).
		Where("user_id = ? AND transaction_date >= ? AND transaction_date <= ? AND capital_gain_loss_stroops < ?",
			userID, startDate, endDate, 0).
		Select("SUM(ABS(capital_gain_loss_stroops))").
		Row().Scan(&totalLoss)

	ts.db.Model(&models.TaxRecord{}).
		Where("user_id = ? AND transaction_date >= ? AND transaction_date <= ?",
			userID, startDate, endDate).
		Count(&recordCount)

	ts.db.Model(&models.TaxWithholding{}).
		Where("user_id = ? AND withholding_date >= ? AND withholding_date <= ?",
			userID, startDate, endDate).
		Select("SUM(withholding_amount_stroops)").
		Row().Scan(&totalIncome)

	netGainLoss := totalGain - totalLoss

	report := &models.TaxReport{
		UserID:             userID,
		TaxYear:            taxYear,
		PeriodStart:        startDate,
		PeriodEnd:          endDate,
		TotalGainStroops:   totalGain,
		TotalLossStroops:   totalLoss,
		NetGainLossStroops: netGainLoss,
		TotalIncomeStroops: totalIncome,
		TransactionCount:   recordCount,
		ReportStatus:       "draft",
		GeneratedAt:        time.Now(),
	}

	if err := ts.db.Create(report).Error; err != nil {
		return nil, err
	}

	return report, nil
}

func (ts *TaxService) GetTaxReport(userID uint, taxYear int) (*models.TaxReport, error) {
	var report models.TaxReport
	if err := ts.db.Where("user_id = ? AND tax_year = ?", userID, taxYear).
		Preload("Records").
		First(&report).Error; err != nil {
		return nil, err
	}
	return &report, nil
}

func (ts *TaxService) ListTaxReports(userID uint) ([]models.TaxReport, error) {
	var reports []models.TaxReport
	if err := ts.db.Where("user_id = ?", userID).
		Order("tax_year DESC").
		Find(&reports).Error; err != nil {
		return nil, err
	}
	return reports, nil
}

// Generate1099Form creates a draft US 1099-NEC information return for a stored
// tax report.
//
// Box mapping (simplified; see docs/TAX_COMPLIANCE.md):
//   - Box 1a (nonemployee compensation) <- TotalIncomeStroops
//   - Box 3 (other income)               <- NetGainLossStroops
//   - Box 5 (federal tax withheld)       <- TotalWithheldTaxStroops
//
// The form is stored with FormStatus "draft" and placeholder filer details; it
// is not transmitted to the IRS. FilerTIN must be set to the platform's real
// employer identification number before filing.
func (ts *TaxService) Generate1099Form(reportID uint, userID uint) (*models.Tax1099Form, error) {
	var report models.TaxReport
	if err := ts.db.First(&report, reportID).Error; err != nil {
		return nil, err
	}

	var user models.User
	if err := ts.db.First(&user, userID).Error; err != nil {
		return nil, err
	}

	formData := map[string]interface{}{
		"box_1a": report.TotalIncomeStroops,
		"box_3":  report.NetGainLossStroops,
		"box_5":  report.TotalWithheldTaxStroops,
	}

	formDataJSON, _ := json.Marshal(formData)

	form := &models.Tax1099Form{
		UserID:           userID,
		TaxReportID:      reportID,
		FormType:         "1099-NEC",
		FilerName:        "kor-AssetForge Marketplace",
		FilerTIN:         "00-0000000",
		RecipientAddress: "See Tax Report",
		TotalIncome:      report.TotalIncomeStroops,
		FormData:         string(formDataJSON),
		FormStatus:       "draft",
		CreatedAt:        time.Now(),
	}

	if err := ts.db.Create(form).Error; err != nil {
		return nil, err
	}

	return form, nil
}

func (ts *TaxService) Get1099Forms(userID uint) ([]models.Tax1099Form, error) {
	var forms []models.Tax1099Form
	if err := ts.db.Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&forms).Error; err != nil {
		return nil, err
	}
	return forms, nil
}

// RecordWithholding persists tax withheld from a payment or distribution.
//
// WithholdingAmountStroops is the amount actually withheld (compute it with
// TaxCalculator.CalculateWithholding if a rate is known) and must be positive.
// WithholdingRate is the rate applied as a decimal fraction (e.g. 0.10 for
// 10%); note the on-chain dividend distributor expresses the same rate in basis
// points, so convert before calling. WithholdingDate defaults to now when unset.
func (ts *TaxService) RecordWithholding(withholding *models.TaxWithholding) error {
	if withholding.UserID == 0 || withholding.WithholdingAmountStroops <= 0 {
		return errors.New("invalid withholding parameters")
	}
	if withholding.WithholdingDate.IsZero() {
		withholding.WithholdingDate = time.Now()
	}
	return ts.db.Create(withholding).Error
}

func (ts *TaxService) GetWithholdings(userID uint) ([]models.TaxWithholding, error) {
	var withholdings []models.TaxWithholding
	if err := ts.db.Where("user_id = ?", userID).
		Order("withholding_date DESC").
		Find(&withholdings).Error; err != nil {
		return nil, err
	}
	return withholdings, nil
}

func (ts *TaxService) ExportTaxReport(reportID uint, userID uint, format string) (*models.TaxExport, error) {
	var report models.TaxReport
	if err := ts.db.First(&report, reportID).Error; err != nil {
		return nil, fmt.Errorf("tax report not found")
	}

	fileName := fmt.Sprintf("tax_report_%d_%d.%s", userID, report.TaxYear, format)
	filePath := fmt.Sprintf("/exports/%s", fileName)

	export := &models.TaxExport{
		UserID:      userID,
		TaxReportID: reportID,
		ExportType:  "tax_report",
		FileName:    fileName,
		FilePath:    filePath,
		FileFormat:  format,
		ExportedAt:  time.Now(),
	}

	if err := ts.db.Create(export).Error; err != nil {
		return nil, err
	}

	if err := ts.db.Model(&report).Update("exported_at", time.Now()).Error; err != nil {
		return nil, err
	}

	return export, nil
}

func (ts *TaxService) GetTaxExports(userID uint) ([]models.TaxExport, error) {
	var exports []models.TaxExport
	if err := ts.db.Where("user_id = ?", userID).
		Order("exported_at DESC").
		Find(&exports).Error; err != nil {
		return nil, err
	}
	return exports, nil
}

func (ts *TaxService) UpdateTaxReportStatus(reportID uint, status string) error {
	return ts.db.Model(&models.TaxReport{}).Where("id = ?", reportID).Update("report_status", status).Error
}

// GetTaxSummary returns aggregate gain/loss/withholding totals for an arbitrary
// date window, used by scheduled reports (see report_scheduler.go) and the tax
// summary endpoint. Unlike GenerateTaxReport it does not persist anything and
// accepts caller-supplied RFC3339 bounds. Returned keys:
// total_gain_stroops, total_loss_stroops (positive magnitude),
// net_gain_loss_stroops, total_withheld_stroops.
func (ts *TaxService) GetTaxSummary(userID uint, startDate, endDate time.Time) (map[string]interface{}, error) {
	var totalGain, totalLoss, totalWithheld int64

	ts.db.Model(&models.TaxRecord{}).
		Where("user_id = ? AND transaction_date >= ? AND transaction_date <= ? AND capital_gain_loss_stroops > ?",
			userID, startDate, endDate, 0).
		Select("SUM(capital_gain_loss_stroops)").
		Row().Scan(&totalGain)

	ts.db.Model(&models.TaxRecord{}).
		Where("user_id = ? AND transaction_date >= ? AND transaction_date <= ? AND capital_gain_loss_stroops < ?",
			userID, startDate, endDate, 0).
		Select("SUM(ABS(capital_gain_loss_stroops))").
		Row().Scan(&totalLoss)

	ts.db.Model(&models.TaxWithholding{}).
		Where("user_id = ? AND withholding_date >= ? AND withholding_date <= ?",
			userID, startDate, endDate).
		Select("SUM(withholding_amount_stroops)").
		Row().Scan(&totalWithheld)

	netGainLoss := totalGain - totalLoss

	return map[string]interface{}{
		"total_gain_stroops":     totalGain,
		"total_loss_stroops":     totalLoss,
		"net_gain_loss_stroops":  netGainLoss,
		"total_withheld_stroops": totalWithheld,
	}, nil
}
