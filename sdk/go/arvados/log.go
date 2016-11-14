package arvados

import (
	"time"
)

// Log is an arvados#log record
type Log struct {
	ID              uint64                 `json:"id"`
	UUID            string                 `json:"uuid"`
	ObjectUUID      string                 `json:"object_uuid"`
	ObjectOwnerUUID string                 `json:"object_owner_uuid"`
	EventType       string                 `json:"event_type"`
	Properties      map[string]interface{} `json:"properties"`
	CreatedAt       *time.Time             `json:"created_at,omitempty"`
}
