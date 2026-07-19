package domain

import (
	"time"

	"github.com/google/uuid"
)

// NewsEvent is a server-generated newspaper entry (§2, §9).
type NewsEvent struct {
	ID            uuid.UUID
	Headline      string
	Body          *string
	Category      *string
	RelatedCardID *uuid.UUID
	CreatedAt     time.Time
}
