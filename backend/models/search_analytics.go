package models

import "time"

// SearchAnalytics stores a durable record of an asset search. Query and
// filters are kept separately so operators can aggregate them without
// parsing request logs.
type SearchAnalytics struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	EventID     string    `gorm:"uniqueIndex;not null" json:"event_id"`
	Query       string    `gorm:"index" json:"query"`
	Filters     string    `gorm:"type:jsonb;not null" json:"filters"`
	ResultCount int64     `json:"result_count"`
	TookMs      float64   `json:"took_ms"`
	UserID      string    `gorm:"index" json:"user_id,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}
