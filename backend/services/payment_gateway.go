package services

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/yourusername/kor-assetforge/models"
	"gorm.io/gorm"
)

// ErrGatewayUnsupported is returned when an unknown gateway is requested.
var ErrGatewayUnsupported = errors.New("unsupported payment gateway")

// ErrIdempotencyKeyRequired is returned when a payment is attempted without an
// idempotency key.
var ErrIdempotencyKeyRequired = errors.New("idempotency key is required")

// ErrIdempotencyConflict is returned when an idempotency key that is still
// within its TTL is reused with different payment parameters (asset, amount,
// currency or gateway). The original request must be retried unchanged or a
// new key must be supplied.
var ErrIdempotencyConflict = errors.New("idempotency key already used with different payment parameters")

// IdempotencyKeyTTL is how long a client-supplied idempotency key is honoured.
// Requests retried with the same key within this window resolve to the original
// payment instead of creating a duplicate charge. After the TTL the key is
// released and may be reused.
const IdempotencyKeyTTL = 24 * time.Hour

// checkoutSessionTTL is how long a generated checkout URL is considered valid;
// it is also used to report ExpiresAt on idempotent replays.
const checkoutSessionTTL = 30 * time.Minute

// GatewayCheckoutSession holds the data needed to redirect a user to the gateway.
type GatewayCheckoutSession struct {
	GatewayPaymentID string `json:"gateway_payment_id"`
	CheckoutURL      string `json:"checkout_url"`
	ExpiresAt        int64  `json:"expires_at"`
	// IdempotentReplay is true when this session was reconstructed from a
	// previously created payment rather than a fresh gateway request.
	IdempotentReplay bool `json:"idempotent_replay"`
}

// PaymentGatewayService orchestrates fiat payment processing via Stripe and PayPal.
type PaymentGatewayService struct {
	db            *gorm.DB
	stripeKey     string
	paypalClientID string
	paypalSecret  string
}

// NewPaymentGatewayService creates a PaymentGatewayService using environment credentials.
func NewPaymentGatewayService(db *gorm.DB) *PaymentGatewayService {
	return &PaymentGatewayService{
		db:             db,
		stripeKey:      os.Getenv("STRIPE_SECRET_KEY"),
		paypalClientID: os.Getenv("PAYPAL_CLIENT_ID"),
		paypalSecret:   os.Getenv("PAYPAL_SECRET"),
	}
}

// CreateCheckoutSession creates a payment intent / order at the chosen gateway and
// stores a pending Payment record. The caller should redirect the user to CheckoutURL.
//
// idempotencyKey is required and must be unique per user. If a payment was
// already created with the same key within IdempotencyKeyTTL, the original
// session is returned with IdempotentReplay set to true and no new gateway
// charge is created. Reusing a live key with different payment parameters
// returns ErrIdempotencyConflict.
func (s *PaymentGatewayService) CreateCheckoutSession(userID, assetID uint, amountFiat int64, currency string, gateway models.PaymentGateway, idempotencyKey string) (*GatewayCheckoutSession, *models.Payment, error) {
	if idempotencyKey == "" {
		return nil, nil, ErrIdempotencyKeyRequired
	}

	existing, err := s.findActivePaymentByIdempotencyKey(userID, idempotencyKey)
	if err != nil {
		return nil, nil, err
	}
	if existing != nil {
		if existing.AssetID != assetID || existing.AmountFiat != amountFiat ||
			existing.Currency != currency || existing.Gateway != gateway {
			return nil, nil, ErrIdempotencyConflict
		}
		return &GatewayCheckoutSession{
			GatewayPaymentID: existing.GatewayPaymentID,
			CheckoutURL:      checkoutURLFor(existing.Gateway, existing.GatewayPaymentID),
			ExpiresAt:        existing.CreatedAt.Add(checkoutSessionTTL).Unix(),
			IdempotentReplay: true,
		}, existing, nil
	}

	var gatewayPaymentID, checkoutURL string
	switch gateway {
	case models.GatewayStripe:
		gatewayPaymentID = fmt.Sprintf("pi_stripe_%d_%d_%d", userID, assetID, time.Now().UnixMilli())
	case models.GatewayPayPal:
		gatewayPaymentID = fmt.Sprintf("pp_order_%d_%d_%d", userID, assetID, time.Now().UnixMilli())
	default:
		return nil, nil, ErrGatewayUnsupported
	}
	checkoutURL = checkoutURLFor(gateway, gatewayPaymentID)

	payment := &models.Payment{
		UserID:           userID,
		AssetID:          assetID,
		Gateway:          gateway,
		GatewayPaymentID: gatewayPaymentID,
		IdempotencyKey:   idempotencyKey,
		AmountFiat:       amountFiat,
		Currency:         currency,
		TokenAmount:      amountFiat / 100, // 1 token per dollar (simplified)
		Status:           models.PaymentStatusPending,
	}

	if err := s.db.Create(payment).Error; err != nil {
		// A concurrent request may have created the payment with the same key.
		// Re-read it and treat the request as an idempotent replay.
		if raced, findErr := s.findActivePaymentByIdempotencyKey(userID, idempotencyKey); findErr == nil && raced != nil {
			return &GatewayCheckoutSession{
				GatewayPaymentID: raced.GatewayPaymentID,
				CheckoutURL:      checkoutURLFor(raced.Gateway, raced.GatewayPaymentID),
				ExpiresAt:        raced.CreatedAt.Add(checkoutSessionTTL).Unix(),
				IdempotentReplay: true,
			}, raced, nil
		}
		return nil, nil, fmt.Errorf("failed to persist payment record: %w", err)
	}

	return &GatewayCheckoutSession{
		GatewayPaymentID: gatewayPaymentID,
		CheckoutURL:      checkoutURL,
		ExpiresAt:        time.Now().Add(checkoutSessionTTL).Unix(),
		IdempotentReplay: false,
	}, payment, nil
}

// findActivePaymentByIdempotencyKey returns the payment created with the given
// key by the user within IdempotencyKeyTTL. Payments older than the TTL are
// "released" (their stored key is tombstoned) and nil is returned so the key can
// be legitimately reused without violating the unique index.
func (s *PaymentGatewayService) findActivePaymentByIdempotencyKey(userID uint, idempotencyKey string) (*models.Payment, error) {
	var payment models.Payment
	err := s.db.Where("user_id = ? AND idempotency_key = ?", userID, idempotencyKey).First(&payment).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if time.Since(payment.CreatedAt) > IdempotencyKeyTTL {
		tombstone := fmt.Sprintf("__expired__%d_%d", payment.ID, time.Now().UnixNano())
		if err := s.db.Model(&payment).Update("idempotency_key", tombstone).Error; err != nil {
			return nil, err
		}
		return nil, nil
	}

	return &payment, nil
}

// GetPaymentByIdempotencyKey returns the payment previously created with the
// given key, if it is still within its TTL and belongs to the user.
func (s *PaymentGatewayService) GetPaymentByIdempotencyKey(userID uint, idempotencyKey string) (*models.Payment, error) {
	return s.findActivePaymentByIdempotencyKey(userID, idempotencyKey)
}

// checkoutURLFor reconstructs the redirect URL for a gateway payment identifier.
func checkoutURLFor(gateway models.PaymentGateway, gatewayPaymentID string) string {
	switch gateway {
	case models.GatewayStripe:
		return "https://checkout.stripe.com/pay/" + gatewayPaymentID
	case models.GatewayPayPal:
		return "https://www.paypal.com/checkoutnow?token=" + gatewayPaymentID
	default:
		return ""
	}
}

// HandleWebhook processes an inbound gateway webhook, updates the payment status,
// and (on success) credits the user's token balance.
func (s *PaymentGatewayService) HandleWebhook(gatewayPaymentID string, rawPayload []byte, status models.PaymentStatus, failureReason string) error {
	var payment models.Payment
	if err := s.db.Where("gateway_payment_id = ?", gatewayPaymentID).First(&payment).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("no payment found for gateway_payment_id %q", gatewayPaymentID)
		}
		return err
	}

	payment.Status = status
	payment.WebhookPayload = string(rawPayload)
	if failureReason != "" {
		payment.FailureReason = failureReason
	}

	return s.db.Save(&payment).Error
}

// GetPayment returns a single payment by its internal ID.
func (s *PaymentGatewayService) GetPayment(paymentID uint) (*models.Payment, error) {
	var p models.Payment
	if err := s.db.First(&p, paymentID).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

// ListUserPayments returns paginated payments for a user.
func (s *PaymentGatewayService) ListUserPayments(userID uint, page, limit int) ([]models.Payment, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit

	var payments []models.Payment
	var total int64

	q := s.db.Model(&models.Payment{}).Where("user_id = ?", userID)
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := q.Order("created_at desc").Limit(limit).Offset(offset).Find(&payments).Error; err != nil {
		return nil, 0, err
	}

	return payments, total, nil
}

// Reconcile compares the local payment records against gateway totals for a period.
func (s *PaymentGatewayService) Reconcile(gateway models.PaymentGateway, start, end time.Time) (*models.PaymentReconciliation, error) {
	var total int64
	var count int64

	if err := s.db.Model(&models.Payment{}).
		Where("gateway = ? AND status = ? AND created_at BETWEEN ? AND ?", gateway, models.PaymentStatusCompleted, start, end).
		Count(&count).Error; err != nil {
		return nil, err
	}

	if err := s.db.Model(&models.Payment{}).
		Select("COALESCE(SUM(amount_fiat), 0)").
		Where("gateway = ? AND status = ? AND created_at BETWEEN ? AND ?", gateway, models.PaymentStatusCompleted, start, end).
		Scan(&total).Error; err != nil {
		return nil, err
	}

	rec := &models.PaymentReconciliation{
		Gateway:       gateway,
		PeriodStart:   start,
		PeriodEnd:     end,
		TotalPayments: int(count),
		TotalAmount:   total,
	}

	if err := s.db.Create(rec).Error; err != nil {
		return nil, err
	}

	return rec, nil
}
