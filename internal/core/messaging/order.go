package core_messaging

import (
	"avitoKitchen/internal/core/domain"
	"time"

	"github.com/google/uuid"
)

type Reason string

const (
	RetryReasonOrderNotFound      Reason = "order_not_found"
	RetryReasonCompanyUnavailable Reason = "company_unavailable"
	RetryReasonCompanyRejected    Reason = "company_rejected"
	RetryReasonInvalidCompanyURL  Reason = "invalid_company_url"
)

type OrderCreatedEvent struct {
	OrderID     uuid.UUID `json:"order_id"`
	CreatedAt   time.Time `json:"created_at"`
	RetryAt     time.Time `json:"retry_at,omitempty"`
	RetryReason Reason    `json:"retry_reason"`

	Order *domain.OrderForSend `json:"order"`
}
