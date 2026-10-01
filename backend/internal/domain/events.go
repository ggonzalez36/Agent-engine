package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// CloudEvent representa el sobre canónico conforme al estándar CloudEvents v1.0.
type CloudEvent struct {
	SpecVersion     string          `json:"specversion"`
	ID              string          `json:"id"`
	Source          string          `json:"source"`
	Type            string          `json:"type"`
	DataContentType string          `json:"datacontenttype"`
	Time            time.Time       `json:"time"`
	CorrelationID   string          `json:"correlation_id"`
	CausationID     string          `json:"causation_id"`
	Data            json.RawMessage `json:"data"`
}

// OrderCreatedPayload datos específicos asociados a la creación de una orden.
type OrderCreatedPayload struct {
	OrderID     uuid.UUID `json:"order_id"`
	CustomerID  string    `json:"customer_id"`
	AmountCents int64     `json:"amount_cents"`
	Currency    string    `json:"currency"`
	Status      string    `json:"status"`
}

// NewCloudEvent construye un evento inmutable listo para serialización en outbox_events.
func NewCloudEvent(
	eventType string,
	source string,
	correlationID string,
	causationID string,
	data any,
) (*CloudEvent, []byte, error) {
	raw, err := json.Marshal(data)
	if err != nil {
		return nil, nil, err
	}

	event := &CloudEvent{
		SpecVersion:     "1.0",
		ID:              "evt_" + uuid.NewString(),
		Source:          source,
		Type:            eventType,
		DataContentType: "application/json",
		Time:            time.Now().UTC(),
		CorrelationID:   correlationID,
		CausationID:     causationID,
		Data:            raw,
	}

	envelopeBytes, err := json.Marshal(event)
	if err != nil {
		return nil, nil, err
	}

	return event, envelopeBytes, nil
}
