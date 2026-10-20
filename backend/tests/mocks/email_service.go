package mocks

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

// SentEmail records a dispatched email for test assertions.
type SentEmail struct {
	Type        string
	ToEmail     string
	ToName      string
	Subject     string
	Body        string
	Token       string
	Status      string
	TxHash      string
	Amount      int64
	AssetID     uint
	FromAddress string
	ToAddress   string
	RequestID   uint
	ExpiresAt   time.Time
	Recipients  []string
	Format      string
	FileName    string
	HTML        string
	PlainText   string
	SentAt      time.Time
}

// MockEmailService provides an in-memory email service implementation for testing without SMTP/SendGrid/SES.
type MockEmailService struct {
	mu           sync.RWMutex
	SentEmails   []SentEmail
	ShouldFail   bool
	FailError    error
	FailNextN    int
	AttemptsMade int
}

func NewMockEmailService() *MockEmailService {
	return &MockEmailService{
		SentEmails: make([]SentEmail, 0),
	}
}

func (m *MockEmailService) checkFailure() error {
	m.AttemptsMade++
	if m.ShouldFail {
		if m.FailError != nil {
			return m.FailError
		}
		return errors.New("email provider service unavailable (500)")
	}
	if m.FailNextN > 0 {
		m.FailNextN--
		if m.FailError != nil {
			return m.FailError
		}
		return errors.New("transient email provider failure (429 Rate Limit)")
	}
	return nil
}

func (m *MockEmailService) SendVerificationEmail(toEmail, toName, verificationToken string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err := m.checkFailure(); err != nil {
		return err
	}

	m.SentEmails = append(m.SentEmails, SentEmail{
		Type:    "verification",
		ToEmail: toEmail,
		ToName:  toName,
		Token:   verificationToken,
		Subject: "Verify your email address",
		SentAt:  time.Now(),
	})
	return nil
}

func (m *MockEmailService) SendKYCStatusUpdate(toEmail, toName, status, reviewNotes string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err := m.checkFailure(); err != nil {
		return err
	}

	m.SentEmails = append(m.SentEmails, SentEmail{
		Type:    "kyc_update",
		ToEmail: toEmail,
		ToName:  toName,
		Status:  status,
		Body:    reviewNotes,
		Subject: fmt.Sprintf("KYC Verification Status: %s", status),
		SentAt:  time.Now(),
	})
	return nil
}

func (m *MockEmailService) SendTransactionConfirmation(toEmail, toName, txHash string, amount int64, assetID uint, fromAddress, toAddress string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err := m.checkFailure(); err != nil {
		return err
	}

	m.SentEmails = append(m.SentEmails, SentEmail{
		Type:        "transaction_confirmation",
		ToEmail:     toEmail,
		ToName:      toName,
		TxHash:      txHash,
		Amount:      amount,
		AssetID:     assetID,
		FromAddress: fromAddress,
		ToAddress:   toAddress,
		Subject:     "Transaction Confirmation",
		SentAt:      time.Now(),
	})
	return nil
}

func (m *MockEmailService) SendApprovalPendingEmail(toEmail, toName string, requestID uint, expiresAt time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err := m.checkFailure(); err != nil {
		return err
	}

	m.SentEmails = append(m.SentEmails, SentEmail{
		Type:      "approval_pending",
		ToEmail:   toEmail,
		ToName:    toName,
		RequestID: requestID,
		ExpiresAt: expiresAt,
		Subject:   "Action Required: Approval Pending",
		SentAt:    time.Now(),
	})
	return nil
}

func (m *MockEmailService) SendScheduledReportEmail(recipients []string, reportName, format, fileName, body string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err := m.checkFailure(); err != nil {
		return err
	}

	m.SentEmails = append(m.SentEmails, SentEmail{
		Type:       "scheduled_report",
		Recipients: recipients,
		Subject:    fmt.Sprintf("Report: %s", reportName),
		Format:     format,
		FileName:   fileName,
		Body:       body,
		SentAt:     time.Now(),
	})
	return nil
}

func (m *MockEmailService) SendGDPRDataExportLink(toEmail, toName, downloadLink string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err := m.checkFailure(); err != nil {
		return err
	}

	m.SentEmails = append(m.SentEmails, SentEmail{
		Type:    "gdpr_export",
		ToEmail: toEmail,
		ToName:  toName,
		Body:    downloadLink,
		Subject: "Your Requested GDPR Data Export",
		SentAt:  time.Now(),
	})
	return nil
}

func (m *MockEmailService) SendCustomEmail(toEmail, toName, subject, html, plainText string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err := m.checkFailure(); err != nil {
		return err
	}

	m.SentEmails = append(m.SentEmails, SentEmail{
		Type:      "custom",
		ToEmail:   toEmail,
		ToName:    toName,
		Subject:   subject,
		HTML:      html,
		PlainText: plainText,
		SentAt:    time.Now(),
	})
	return nil
}

// TotalSent returns the count of dispatched emails.
func (m *MockEmailService) TotalSent() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.SentEmails)
}

// Reset clears recorded emails and failure settings.
func (m *MockEmailService) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.SentEmails = make([]SentEmail, 0)
	m.ShouldFail = false
	m.FailError = nil
	m.FailNextN = 0
	m.AttemptsMade = 0
}
